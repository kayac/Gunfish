// Package e2e runs the gunfish binary against APNs and FCM v1 mock servers.
package e2e

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/kayac/Gunfish/mock"
)

// apnsMockAddr is the APNs host which gunfish uses in the test environment (gunfish.MockServer).
const apnsMockAddr = "127.0.0.1:2195"

const fcmProjectID = "e2e"

type received struct {
	Path   string
	Header http.Header
	Body   any
}

type recorder struct {
	mu   sync.Mutex
	reqs []received
}

func (r *recorder) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var body any
		if err := json.Unmarshal(b, &body); err != nil {
			body = string(b)
		}
		r.mu.Lock()
		r.reqs = append(r.reqs, received{Path: req.URL.Path, Header: req.Header.Clone(), Body: body})
		r.mu.Unlock()
		req.Body = io.NopCloser(bytes.NewReader(b))
		next.ServeHTTP(w, req)
	})
}

func (r *recorder) snapshot() []received {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]received(nil), r.reqs...)
}

func TestE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	dir := t.TempDir()

	bin := buildGunfish(t, dir)
	certFile, keyFile := generateCert(t, dir)

	// APNs mock server (TLS, HTTP/2) on the fixed address used by -environment test.
	apnsRec := &recorder{}
	apnsLis, err := net.Listen("tcp", apnsMockAddr)
	if err != nil {
		t.Fatalf("cannot listen on %s for APNs mock server: %s", apnsMockAddr, err)
	}
	apnsSrv := httptest.NewUnstartedServer(apnsRec.wrap(mock.APNsMockServer(false)))
	apnsSrv.Listener.Close()
	apnsSrv.Listener = apnsLis
	apnsSrv.EnableHTTP2 = true
	apnsSrv.StartTLS()
	t.Cleanup(apnsSrv.Close)

	// FCM v1 mock server
	fcmRec := &recorder{}
	fcmSrv := httptest.NewServer(fcmRec.wrap(mock.FCMv1MockServer(fcmProjectID, false)))
	t.Cleanup(fcmSrv.Close)

	// The error hook may run concurrently, so each invocation writes its own file.
	hookDir := filepath.Join(dir, "hook")
	if err := os.Mkdir(hookDir, 0755); err != nil {
		t.Fatal(err)
	}
	port := freePort(t)
	confFile := filepath.Join(dir, "gunfish.toml")
	conf := fmt.Sprintf(`
[provider]
port = %d
worker_num = 2
queue_size = 128
max_request_size = 1000
max_connections = 100
error_hook = "cat > %[2]s/.tmp.$$ && mv %[2]s/.tmp.$$ %[2]s/$$.json"

[apns]
cert_file = "%s"
key_file = "%s"

[fcm_v1]
enabled = true
endpoint = "%s/v1/projects"
projectid = "%s"
`, port, hookDir, certFile, keyFile, fcmSrv.URL, fcmProjectID)
	if err := os.WriteFile(confFile, []byte(conf), 0644); err != nil {
		t.Fatal(err)
	}

	var logs bytes.Buffer
	cmd := exec.Command(bin, "-c", confFile, "-environment", "test", "-log-format", "json", "-log-level", "debug")
	cmd.Stdout = &logs
	cmd.Stderr = &logs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	stopped := false
	t.Cleanup(func() {
		if !stopped {
			cmd.Process.Kill()
			<-exited
		}
		if t.Failed() {
			t.Logf("gunfish logs:\n%s", logs.String())
		}
	})

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	waitForReady(t, base+"/stats/app", exited)

	t.Run("push apns", func(t *testing.T) {
		body := `[
  {"token": "e2e-apns-1", "payload": {"aps": {"alert": {"title": "t1", "body": "b1"}, "sound": "default", "badge": 1}, "custom": "x"}, "header": {"apns-topic": "com.example.e2e", "apns-push-type": "alert"}},
  {"token": "e2e-apns-2", "payload": {"aps": {"alert": "simple", "content-available": 1}}},
  {"token": "baddevicetoken", "payload": {"aps": {"alert": "bad"}}},
  {"token": "unregistered", "payload": {"aps": {"alert": "unregistered"}}},
  {"token": "missingtopic", "payload": {"aps": {"alert": "missing topic"}}}
]`
		post(t, base+"/push/apns", "application/json", body)
	})

	t.Run("push apns form", func(t *testing.T) {
		v := url.Values{}
		v.Set("json", `[{"token": "e2e-apns-form", "payload": {"aps": {"alert": "form"}}}]`)
		post(t, base+"/push/apns", "application/x-www-form-urlencoded", v.Encode())
	})

	t.Run("push fcm v1", func(t *testing.T) {
		body := `{"message": {"token": "e2e-fcm-1", "notification": {"title": "t1", "body": "b1", "image": "https://example.com/i.png"}, "data": {"k": "v"}, "android": {"priority": "high", "ttl": "60s", "notification": {"channel_id": "ch"}}}}
{"message": {"token": "e2e-fcm-2", "apns": {"headers": {"apns-priority": "10"}, "payload": {"aps": {"alert": {"title": "t2"}, "badge": 2}, "custom": "y"}}}}
{"message": {"token": "UNREGISTERED", "notification": {"title": "unregistered"}}}
{"message": {"token": "INVALID_ARGUMENT", "notification": {"title": "invalid"}}}
`
		post(t, base+"/push/fcm/v1", "application/json", body)
	})

	// Wait for all notifications to be processed.
	wantStats := map[string]float64{
		"req_count":          3,
		"sent_count":         8, // APNs error responses are also counted as sent
		"err_count":          5,
		"retry_count":        0,
		"workers":            2,
		"queue_size":         0,
		"workers_queue_size": 0,
		"retry_queue_size":   0,
		"cmdq_queue_size":    0,
	}
	var statsDiff string
	deadline := time.Now().Add(30 * time.Second)
	for {
		statsDiff = cmp.Diff(wantStats, pick(getJSON(t, base+"/stats/app"), wantStats))
		if statsDiff == "" && len(apnsRec.snapshot()) >= 6 && len(fcmRec.snapshot()) >= 4 && len(readHooks(t, hookDir)) >= 5 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out: apns=%d fcm=%d hook=%d stats diff (-want +got):\n%s",
				len(apnsRec.snapshot()), len(fcmRec.snapshot()), len(readHooks(t, hookDir)), statsDiff)
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Run("apns requests", func(t *testing.T) {
		got := map[string]any{}
		for _, r := range apnsRec.snapshot() {
			header := map[string]any{}
			for _, k := range []string{"Apns-Topic", "Apns-Push-Type", "Apns-Id", "Authorization"} {
				if v := r.Header.Get(k); v != "" {
					header[k] = v
				}
			}
			got[r.Path] = map[string]any{"header": header, "body": r.Body}
		}
		want := mustJSON(t, `{
  "/3/device/e2e-apns-1": {
    "header": {"Apns-Topic": "com.example.e2e", "Apns-Push-Type": "alert"},
    "body": {"aps": {"alert": {"title": "t1", "body": "b1"}, "sound": "default", "badge": 1}, "custom": "x"}
  },
  "/3/device/e2e-apns-2": {"header": {}, "body": {"aps": {"alert": "simple", "content-available": 1}}},
  "/3/device/baddevicetoken": {"header": {}, "body": {"aps": {"alert": "bad"}}},
  "/3/device/unregistered": {"header": {}, "body": {"aps": {"alert": "unregistered"}}},
  "/3/device/missingtopic": {"header": {}, "body": {"aps": {"alert": "missing topic"}}},
  "/3/device/e2e-apns-form": {"header": {}, "body": {"aps": {"alert": "form"}}}
}`)
		if diff := cmp.Diff(want, any(got)); diff != "" {
			t.Errorf("APNs requests mismatch (-want +got):\n%s", diff)
		}
		if n := len(apnsRec.snapshot()); n != 6 {
			t.Errorf("APNs request count: want 6, got %d", n)
		}
	})

	t.Run("fcm requests", func(t *testing.T) {
		got := map[string]any{}
		for _, r := range fcmRec.snapshot() {
			if r.Path != "/v1/projects/"+fcmProjectID+"/messages:send" {
				t.Errorf("unexpected FCM request path: %s", r.Path)
			}
			got[r.Header.Get("Authorization")] = r.Body
		}
		want := mustJSON(t, `{
  "Bearer e2e-fcm-1": {"message": {"token": "e2e-fcm-1", "notification": {"title": "t1", "body": "b1", "image": "https://example.com/i.png"}, "data": {"k": "v"}, "android": {"priority": "high", "ttl": "60s", "notification": {"channel_id": "ch"}}}},
  "Bearer e2e-fcm-2": {"message": {"token": "e2e-fcm-2", "apns": {"headers": {"apns-priority": "10"}, "payload": {"aps": {"alert": {"title": "t2"}, "badge": 2}, "custom": "y"}}}},
  "Bearer UNREGISTERED": {"message": {"token": "UNREGISTERED", "notification": {"title": "unregistered"}}},
  "Bearer INVALID_ARGUMENT": {"message": {"token": "INVALID_ARGUMENT", "notification": {"title": "invalid"}}}
}`)
		if diff := cmp.Diff(want, any(got)); diff != "" {
			t.Errorf("FCM requests mismatch (-want +got):\n%s", diff)
		}
		if n := len(fcmRec.snapshot()); n != 4 {
			t.Errorf("FCM request count: want 4, got %d", n)
		}
	})

	t.Run("error hook", func(t *testing.T) {
		got := map[string]any{}
		lines := readHooks(t, hookDir)
		for _, l := range lines {
			got[fmt.Sprint(l["token"])] = l
		}
		want := mustJSON(t, `{
  "baddevicetoken": {"provider": "apns", "token": "baddevicetoken", "apns-id": "apns-id", "status": 400, "reason": "BadDeviceToken"},
  "unregistered": {"provider": "apns", "token": "unregistered", "apns-id": "", "status": 410, "reason": "Unregistered"},
  "missingtopic": {"provider": "apns", "token": "missingtopic", "apns-id": "", "status": 400, "reason": "MissingTopic"},
  "UNREGISTERED": {"provider": "fcmv1", "token": "UNREGISTERED", "status": 404, "error": {"status": "UNREGISTERED", "message": "mock error:UNREGISTERED"}},
  "INVALID_ARGUMENT": {"provider": "fcmv1", "token": "INVALID_ARGUMENT", "status": 400, "error": {"status": "INVALID_ARGUMENT", "message": "mock error:INVALID_ARGUMENT"}}
}`)
		if diff := cmp.Diff(want, any(got)); diff != "" {
			t.Errorf("error hook input mismatch (-want +got):\n%s", diff)
		}
		if len(lines) != 5 {
			t.Errorf("error hook invocation count: want 5, got %d", len(lines))
		}
	})

	t.Run("stats profile", func(t *testing.T) {
		v := getJSON(t, base+"/stats/profile")
		if _, ok := v["goroutine_num"]; !ok {
			t.Errorf("unexpected /stats/profile response: %v", v)
		}
	})

	// Graceful shutdown
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-exited:
		stopped = true
		if err != nil {
			t.Errorf("gunfish exited with error: %s", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("gunfish did not stop within 30s after SIGTERM")
	}
}

func buildGunfish(t *testing.T, dir string) string {
	t.Helper()
	bin := filepath.Join(dir, "gunfish")
	out, err := exec.Command("go", "build", "-o", bin, "github.com/kayac/Gunfish/cmd/gunfish").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build gunfish: %s\n%s", err, out)
	}
	return bin
}

func generateCert(t *testing.T, dir string) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "gunfish-e2e"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certFile := filepath.Join(dir, "client.crt")
	keyFile := filepath.Join(dir, "client.key")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := tls.LoadX509KeyPair(certFile, keyFile); err != nil {
		t.Fatal(err)
	}
	return certFile, keyFile
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func waitForReady(t *testing.T, u string, exited <-chan error) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-exited:
			t.Fatalf("gunfish exited before ready: %v", err)
		default:
		}
		res, err := http.Get(u)
		if err == nil {
			res.Body.Close()
			if res.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("gunfish did not become ready")
}

func post(t *testing.T, u, contentType, body string) {
	t.Helper()
	res, err := http.Post(u, contentType, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", res.StatusCode, b)
	}
	var got map[string]string
	if err := json.Unmarshal(b, &got); err != nil || got["result"] != "ok" {
		t.Fatalf("unexpected response body: %s", b)
	}
}

func getJSON(t *testing.T, u string) map[string]any {
	t.Helper()
	res, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var v map[string]any
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func readHooks(t *testing.T, dir string) []map[string]any {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	var hooks []map[string]any
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var v map[string]any
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("invalid hook input %q: %s", b, err)
		}
		hooks = append(hooks, v)
	}
	return hooks
}

func mustJSON(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("invalid JSON %s: %s", s, err)
	}
	return v
}

func pick(m map[string]any, keys map[string]float64) map[string]float64 {
	r := make(map[string]float64, len(keys))
	for k := range keys {
		if v, ok := m[k].(float64); ok {
			r[k] = v
		}
	}
	return r
}
