package gunfish_test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	gunfish "github.com/kayac/Gunfish"
	"github.com/kayac/Gunfish/apns"
	"github.com/kayac/Gunfish/config"
	"github.com/kayac/Gunfish/mock"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
)

func TestMain(m *testing.M) {
	runner := func() int {
		apns.ClientTransport = func(cert tls.Certificate) *http.Transport {
			return &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
					Certificates:       []tls.Certificate{cert},
				},
			}
		}
		gunfish.InitErrorResponseHandler(gunfish.DefaultResponseHandler{Hook: `cat `})
		gunfish.InitSuccessResponseHandler(gunfish.DefaultResponseHandler{})
		logrus.SetLevel(logrus.WarnLevel)

		ts := httptest.NewUnstartedServer(mock.APNsMockServer(false))
		if err := http2.ConfigureServer(ts.Config, nil); err != nil {
			return 1
		}
		ts.TLS = ts.Config.TLSConfig
		ts.StartTLS()
		conf.Apns.Host = ts.URL

		code := m.Run()

		return code
	}

	os.Exit(runner())
}

func TestInvalidCertification(t *testing.T) {
	c, _ := config.LoadConfig("./test/gunfish_test.toml")
	c.Apns.CertFile = "./test/invalid.crt"
	c.Apns.KeyFile = "./test/invalid.key"
	ss, err := gunfish.StartSupervisor(&c)
	if err != nil {
		t.Errorf("Expected supervisor cannot start because of using invalid certification files.: %v", ss)
	}
}

func TestSuccessToPostJson(t *testing.T) {
	sup, _ := gunfish.StartSupervisor(&conf)
	prov := &gunfish.Provider{Sup: sup}
	handler := prov.PushAPNsHandler()

	// application/json
	jsons := createJSONPostedData(3)
	w := httptest.NewRecorder()
	r, err := newRequest(jsons, "POST", gunfish.ApplicationJSON)
	if err != nil {
		t.Errorf("%s", err)
	}

	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code is 200 but got %d", w.Code)
	}

	// application/x-www-form-urlencoded
	data := createFormPostedData(3)
	w = httptest.NewRecorder() // re-creates Recoder because cannot overwrite header after to write body.
	r, err = newRequest(data, "POST", gunfish.ApplicationXW3FormURLEncoded)
	if err != nil {
		t.Errorf("%s", err)
	}

	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code is 200 but got %d", w.Code)
	}

	sup.Shutdown()
}

func TestFailedToPostInvalidJson(t *testing.T) {
	sup, _ := gunfish.StartSupervisor(&conf)
	prov := &gunfish.Provider{Sup: sup}
	handler := prov.PushFCMHandler()

	// missing `}`
	invalidJson := []byte(`{"registration_ids": ["xxxxxxxxx"], "data": {"message":"test"`)

	w := httptest.NewRecorder()
	r, err := newRequest(invalidJson, "POST", gunfish.ApplicationJSON)
	if err != nil {
		t.Errorf("%s", err)
	}

	handler.ServeHTTP(w, r)

	invalidResponse := bytes.NewBufferString("{\"reason\":\"unexpected EOF\"}{\"result\": \"ok\"}").String()
	if w.Body.String() == invalidResponse {
		t.Errorf("Invalid Json responce: '%s'", w.Body)
	}

	sup.Shutdown()
}

func TestFailedToPostMalformedJson(t *testing.T) {
	sup, _ := gunfish.StartSupervisor(&conf)
	prov := &gunfish.Provider{Sup: sup}
	handler := prov.PushAPNsHandler()

	jsons := []string{
		`{"test":"test"}`,
		"[{\"payload\": {\"aps\": {\"alert\":\"msg\", \"sound\":\"default\" }}}]",
		"[{\"token\":\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\"}]",
	}
	for _, s := range jsons {
		v := url.Values{}
		v.Add("json", s)

		// application/x-www-form-urlencoded
		r, err := newRequest([]byte(v.Encode()), "POST", gunfish.ApplicationXW3FormURLEncoded)
		if err != nil {
			t.Errorf("%s", err)
		}

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code == http.StatusOK {
			t.Errorf("Expected status code is NOT 200 but got %d", w.Code)
		}

		// application/json
		r, err = newRequest([]byte(s), "POST", gunfish.ApplicationJSON)
		if err != nil {
			t.Errorf("%s", err)
		}

		w = httptest.NewRecorder() // re-creates Recoder because cannot overwrite header after to write body.
		handler.ServeHTTP(w, r)
		if w.Code == http.StatusOK {
			t.Errorf("Expected status code is NOT 200 but got %d", w.Code)
		}
	}

	sup.Shutdown()
}

func TestEnqueueTooManyRequest(t *testing.T) {
	sup, _ := gunfish.StartSupervisor(&conf)
	prov := &gunfish.Provider{Sup: sup}
	handler := prov.PushAPNsHandler()

	var jsons [][]byte
	for i := 0; i < 1000; i++ {
		jsons = append(jsons, createJSONPostedData(1)) // Too many requests
	}

	// Test 503 returns
	check503 := false
	w := httptest.NewRecorder() // creates new Recoder because cannot overwrite header.
	var ra string
	for _, json := range jsons {
		r, err := newRequest(json, "POST", gunfish.ApplicationJSON)
		if err != nil {
			t.Errorf("%s", err)
		}

		handler.ServeHTTP(w, r)
		if w.Code == http.StatusServiceUnavailable {
			check503 = true
			ra = w.Header().Get("Retry-After")
			break
		} else {
			w = httptest.NewRecorder()
		}
	}
	if check503 == false {
		t.Errorf("Expected status code is 503 but got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("Not set Retry-After correctlly")
	}

	// Test Retry-After value increases
	w = httptest.NewRecorder() // re-creates Recoder because cannot overwrite header after to write body.
	r, err := newRequest(jsons[0], "POST", gunfish.ApplicationJSON)
	if err != nil {
		t.Errorf("%s", err)
	}

	handler.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status code is 503 but got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == ra {
		t.Errorf("Retry-After should become different: old[ %s ], new[ %s ]", ra, w.Header().Get("Retry-After"))
	}

	sup.Shutdown()
}

func TestTooLargeRequest(t *testing.T) {
	sup, _ := gunfish.StartSupervisor(&conf)
	prov := &gunfish.Provider{Sup: sup}
	handler := prov.PushAPNsHandler()

	jsons := createJSONPostedData(config.MaxRequestSize + 1) // Too many requests
	r, err := newRequest(jsons, "POST", gunfish.ApplicationJSON)
	if err != nil {
		t.Errorf("%s", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code is %d but got %d", http.StatusBadRequest, w.Code)
	}

	sup.Shutdown()
}

func TestMethodNotAllowed(t *testing.T) {
	sup, _ := gunfish.StartSupervisor(&conf)
	prov := &gunfish.Provider{Sup: sup}
	handler := prov.PushAPNsHandler()

	jsons := createJSONPostedData(1)
	w := httptest.NewRecorder()
	r, err := newRequest(jsons, "GET", gunfish.ApplicationJSON)
	if err != nil {
		t.Errorf("%s", err)
	}

	handler.ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code is 405 but got %d", w.Code)
	}

	sup.Shutdown()
}

func TestUnsupportedMediaType(t *testing.T) {
	sup, _ := gunfish.StartSupervisor(&conf)
	prov := &gunfish.Provider{Sup: sup}
	handler := prov.PushAPNsHandler()

	jsons := createPostedData(1)
	r, err := http.NewRequest(
		"POST",
		"",
		bytes.NewBuffer([]byte(jsons)),
	)
	r.Header.Set("Content-Type", "plain/text")
	w := httptest.NewRecorder()
	if err != nil {
		t.Errorf("%s", err)
	}

	handler.ServeHTTP(w, r)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("Expected status code is 415 but got %d", w.Code)
	}

	sup.Shutdown()
}

func TestStats(t *testing.T) {
	sup, _ := gunfish.StartSupervisor(&conf)
	prov := &gunfish.Provider{Sup: sup}
	pushh := prov.PushAPNsHandler()
	statsh := prov.StatsHandler()

	var statsBefore, statsAfter gunfish.Stats
	// GET stats
	{
		r, err := newRequest([]byte(""), "GET", gunfish.ApplicationJSON)
		if err != nil {
			t.Errorf("%s", err)
		}
		w := httptest.NewRecorder()
		statsh.ServeHTTP(w, r)
		de := json.NewDecoder(w.Body)
		de.Decode(&statsBefore)
	}

	// Updates stat
	{
		jsons := createJSONPostedData(1)
		r, err := newRequest(jsons, "POST", gunfish.ApplicationJSON)
		if err != nil {
			t.Errorf("%s", err)
		}
		w := httptest.NewRecorder()
		pushh.ServeHTTP(w, r)
	}

	// GET stats
	{
		r, err := newRequest([]byte(""), "GET", gunfish.ApplicationJSON)
		if err != nil {
			t.Errorf("%s", err)
		}
		w := httptest.NewRecorder()
		statsh.ServeHTTP(w, r)
		de := json.NewDecoder(w.Body)
		de.Decode(&statsAfter)
	}

	if statsAfter.RequestCount != statsBefore.RequestCount+1 {
		t.Errorf("Unexpected stats request count: %#v %#v", statsBefore, statsAfter)
	}

	if statsAfter.CertificateExpireUntil < 0 {
		t.Errorf("Certificate expired %s %d", statsAfter.CertificateNotAfter, statsAfter.CertificateExpireUntil)
	}

	sup.Shutdown()
}

func newRequest(data []byte, method string, c string) (*http.Request, error) {
	req, err := http.NewRequest(
		method,
		"",
		bytes.NewBuffer([]byte(data)),
	)
	req.Header.Set("Content-Type", c)
	return req, err
}

func createJSONPostedData(num int) []byte {
	return createPostedData(num)
}

func createFormPostedData(num int) []byte {
	jsonStr := createPostedData(num)
	v := url.Values{}
	v.Add("json", string(jsonStr))
	return []byte(v.Encode())
}

func createPostedData(num int) []byte {
	pds := make([]gunfish.PostedData, num)
	tokens := make([]string, num)
	for i := 0; i < num; i++ {
		tokens[i] = fmt.Sprintf("%032d", i)
	}
	for i, v := range tokens {
		payload := apns.Payload{}

		payload.APS = &apns.APS{
			Alert: apns.Alert{
				Title: "test",
				Body:  "message",
			},
			Sound: "default",
		}

		pds[i] = gunfish.PostedData{
			Payload: payload,
			Token:   v,
		}
	}

	jsonStr, _ := json.Marshal(pds)

	return jsonStr
}
