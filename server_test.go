package gunfish

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/kayac/Gunfish/apns"
	"github.com/sirupsen/logrus"
)

func init() {
	InitErrorResponseHandler(DefaultResponseHandler{hook: `cat `})
	InitSuccessResponseHandler(DefaultResponseHandler{})
	logrus.SetLevel(logrus.WarnLevel)
	config.Apns.Host = MockServer
}

func TestInvalidCertification(t *testing.T) {
	c, _ := LoadConfig("./test/gunfish_test.toml")
	c.Apns.CertFile = "./test/invalid.crt"
	c.Apns.KeyFile = "./test/invalid.key"
	ss, err := StartSupervisor(&c)
	if err != nil {
		t.Errorf("Expected supervisor cannot start because of using invalid certification files.: %v", ss)
	}
}

func TestSuccessToPostJson(t *testing.T) {
	sup, _ := StartSupervisor(&config)
	prov := &Provider{sup: sup}
	handler := prov.pushAPNsHandler()

	// application/json
	jsons := createJSONPostedData(3)
	w := httptest.NewRecorder()
	r, err := newRequest(jsons, "POST", ApplicationJSON)
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
	r, err = newRequest(data, "POST", ApplicationXW3FormURLEncoded)
	if err != nil {
		t.Errorf("%s", err)
	}

	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code is 200 but got %d", w.Code)
	}

	sup.Shutdown()
}

func TestFailedToPostMalformedJson(t *testing.T) {
	sup, _ := StartSupervisor(&config)
	prov := &Provider{sup: sup}
	handler := prov.pushAPNsHandler()

	jsons := []string{
		`{"test":"test"}`,
		"[{\"payload\": {\"aps\": {\"alert\":\"msg\", \"sound\":\"default\" }}}]",
		"[{\"token\":\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\"}]",
	}
	for _, s := range jsons {
		v := url.Values{}
		v.Add("json", s)

		// application/x-www-form-urlencoded
		r, err := newRequest([]byte(v.Encode()), "POST", ApplicationXW3FormURLEncoded)
		if err != nil {
			t.Errorf("%s", err)
		}

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code == http.StatusOK {
			t.Errorf("Expected status code is NOT 200 but got %d", w.Code)
		}

		// application/json
		r, err = newRequest([]byte(s), "POST", ApplicationJSON)
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
	sup, _ := StartSupervisor(&config)
	prov := &Provider{sup: sup}
	srvStats = NewStats(config)
	handler := prov.pushAPNsHandler()

	// When queue stack is full, return 503
	var manyNum int
	tp := ((config.Provider.RequestQueueSize * int(AverageResponseTime/time.Millisecond)) / 1000) / SenderNum
	dif := (RequestPerSec - config.Provider.RequestQueueSize/tp)
	if dif > 0 {
		manyNum = dif * int(FlowRateInterval/time.Second) * 2
	} else {
		manyNum = -1 * dif * int(FlowRateInterval/time.Second) * 2
	}

	var jsons [][]byte
	for i := 0; i < manyNum; i++ {
		jsons = append(jsons, createJSONPostedData(1)) // Too many requests
	}

	// Test 503 returns
	check503 := false
	w := httptest.NewRecorder() // creates new Recoder because cannot overwrite header.
	var ra string
	for _, json := range jsons {
		r, err := newRequest(json, "POST", ApplicationJSON)
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
	r, err := newRequest(jsons[0], "POST", ApplicationJSON)
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
	sup, _ := StartSupervisor(&config)
	prov := &Provider{sup: sup}
	srvStats = NewStats(config)
	handler := prov.pushAPNsHandler()

	jsons := createJSONPostedData(MaxRequestSize + 1) // Too many requests
	r, err := newRequest(jsons, "POST", ApplicationJSON)
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
	sup, _ := StartSupervisor(&config)
	prov := &Provider{sup: sup}
	handler := prov.pushAPNsHandler()

	jsons := createJSONPostedData(1)
	w := httptest.NewRecorder()
	r, err := newRequest(jsons, "GET", ApplicationJSON)
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
	sup, _ := StartSupervisor(&config)
	prov := &Provider{sup: sup}
	handler := prov.pushAPNsHandler()

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
	sup, _ := StartSupervisor(&config)
	prov := &Provider{sup: sup}
	srvStats = NewStats(config)
	pushh := prov.pushAPNsHandler()
	statsh := prov.statsHandler()

	// Updates stat
	jsons := createJSONPostedData(1)
	w := httptest.NewRecorder()
	r, err := newRequest(jsons, "POST", ApplicationJSON)
	if err != nil {
		t.Errorf("%s", err)
	}
	pushh.ServeHTTP(w, r)
	w = httptest.NewRecorder() // re-creates Recoder because cannot overwrite header after to write body.

	// GET status
	r, err = newRequest([]byte(""), "GET", ApplicationJSON)
	if err != nil {
		t.Errorf("%s", err)
	}
	time.Sleep(time.Second * 1) // for update uptime of stats
	statsh.ServeHTTP(w, r)

	// check stat
	var resStat Stats
	de := json.NewDecoder(w.Body)
	de.Decode(&resStat)

	if !reflect.DeepEqual(srvStats, resStat) {
		t.Errorf("Expected total conenctions \"%v\" but got \"%v\"", srvStats, resStat)
	}

	if resStat.CertificateExpireUntil < 0 {
		t.Errorf("Certificate expired %s %d", resStat.CertificateNotAfter, resStat.CertificateExpireUntil)
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
	pds := make([]PostedData, num)
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

		pds[i] = PostedData{
			Payload: payload,
			Token:   v,
		}
	}

	jsonStr, _ := json.Marshal(pds)

	return jsonStr
}
