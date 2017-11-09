package apns

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/http2"
)

const (
	ApplicationJSON        = "application/json"
	LimitApnsTokenByteSize = 100 // Payload byte size.
)

// StartAPNSMockServer starts HTTP/2 server for mock
func StartAPNSMockServer(cert, key string) {
	// Create TLSlistener
	s := http.Server{}
	s.Addr = ":2195"
	http2.VerboseLogs = false
	http2.ConfigureServer(&s, nil)
	tlsConf := &tls.Config{}
	if s.TLSConfig != nil {
		tlsConf = s.TLSConfig.Clone()
	}
	if tlsConf.NextProtos == nil {
		tlsConf.NextProtos = []string{"http/2.0"}
	}

	var err error
	tlsConf.Certificates = make([]tls.Certificate, 1)
	tlsConf.Certificates[0], err = tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return
	}

	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return
	}

	tlsListener := tls.NewListener(ln, tlsConf)

	http.HandleFunc("/3/device/", func(w http.ResponseWriter, r *http.Request) {
		// sets the response time from apns server
		time.Sleep(time.Millisecond*200 + time.Millisecond*(time.Duration(rand.Int63n(90))-45))

		// only allow path which pattern is '/3/device/:token'
		splitPath := strings.Split(r.URL.Path, "/")
		if len(splitPath) != 4 {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "404 Not found")
			return
		}

		w.Header().Set("Content-Type", ApplicationJSON)

		token := splitPath[len(splitPath)-1]
		if len(([]byte(token))) > LimitApnsTokenByteSize {
			w.Header().Set("apns-id", "apns-id")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, createErrorResponse(BadDeviceToken, http.StatusBadRequest))
		} else if token == "missingtopic" {
			// MissingDeviceToken
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, createErrorResponse(MissingTopic, http.StatusBadRequest))
		} else if token == "status410" {
			// If the value in the :status header is 410, the value of this key is
			// the last time at which APNs confirmed that the device token was
			// no longer valid for the topic.
			//
			// Stop pushing notifications until the device registers a token with
			// a later timestamp with your provider.
			w.WriteHeader(http.StatusGone)
			fmt.Fprint(w, createErrorResponse(TopicDisallowed, http.StatusGone))
		} else {
			w.Header().Set("apns-id", "apns-id")
			w.WriteHeader(http.StatusOK)
		}

		return
	})

	http.HandleFunc("/stop/", func(w http.ResponseWriter, r *http.Request) {
		tlsListener.Close()
		return
	})

	log.Fatal(s.Serve(tlsListener))
}

// StopAPNSServer stops APNS Mock server
func StopAPNSServer(cert, key string, insecure bool) error {
	client, err := NewConnection(cert, key, insecure)
	if err != nil {
		return err
	}

	client.Get("/stop")

	return nil
}

func createErrorResponse(ermsg ErrorResponseCode, status int) string {
	var er ErrorResponse
	if status == http.StatusGone {
		er = ErrorResponse{
			Reason:    ermsg.String(),
			Timestamp: time.Now().Unix(),
		}
	} else {
		er = ErrorResponse{
			Reason: ermsg.String(),
		}
	}
	der, _ := json.Marshal(er)
	return string(der)
}
