package mock

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/kayac/Gunfish/apns"
)

const (
	ApplicationJSON        = "application/json"
	LimitApnsTokenByteSize = 100 // Payload byte size.
)

// StartAPNSMockServer starts HTTP/2 server for mock
func APNsMockServer(verbose bool) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/3/device/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			if verbose {
				log.Printf("reqtime:%f proto:%s method:%s path:%s host:%s", reqtime(start), r.Proto, r.Method, r.URL.Path, r.RemoteAddr)
			}
		}()

		// sets the response time from apns server
		time.Sleep(time.Millisecond*200 + time.Millisecond*(time.Duration(rand.Int63n(200)-100)))

		// only allow path which pattern is '/3/device/:token'
		splitPath := strings.Split(r.URL.Path, "/")
		if len(splitPath) != 4 {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "404 Not found")
			return
		}

		w.Header().Set("Content-Type", ApplicationJSON)

		token := splitPath[len(splitPath)-1]
		if len(([]byte(token))) > LimitApnsTokenByteSize || token == "baddevicetoken" {
			w.Header().Set("apns-id", "apns-id")
			w.WriteHeader(http.StatusBadRequest)
			createErrorResponse(w, apns.BadDeviceToken, http.StatusBadRequest)
		} else if token == "missingtopic" {
			w.WriteHeader(http.StatusBadRequest)
			createErrorResponse(w, apns.MissingTopic, http.StatusBadRequest)
		} else if token == "unregistered" {
			// If the value in the :status header is 410, the value of this key is
			// the last time at which APNs confirmed that the device token was
			// no longer valid for the topic.
			//
			// Stop pushing notifications until the device registers a token with
			// a later timestamp with your provider.
			w.WriteHeader(http.StatusGone)
			createErrorResponse(w, apns.Unregistered, http.StatusGone)
		} else if token == "expiredprovidertoken" {
			w.WriteHeader(http.StatusForbidden)
			createErrorResponse(w, apns.ExpiredProviderToken, http.StatusForbidden)
		} else {
			w.Header().Set("apns-id", "apns-id")
			w.WriteHeader(http.StatusOK)
		}

		return
	})

	return mux
}

func createErrorResponse(w io.Writer, ermsg apns.ErrorResponseCode, status int) error {
	enc := json.NewEncoder(w)
	var er apns.ErrorResponse
	if status == http.StatusGone {
		er = apns.ErrorResponse{
			Reason:    ermsg.String(),
			Timestamp: time.Now().Unix(),
		}
	} else {
		er = apns.ErrorResponse{
			Reason: ermsg.String(),
		}
	}
	return enc.Encode(er)
}

func reqtime(start time.Time) float64 {
	diff := time.Now().Sub(start)
	return diff.Seconds()
}
