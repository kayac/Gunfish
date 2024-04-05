package mock

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/kayac/Gunfish/fcmv1"
)

func FCMv1MockServer(projectID string, verbose bool) *http.ServeMux {
	mux := http.NewServeMux()
	p := fmt.Sprintf("/v1/projects/%s/messages:send", projectID)
	log.Println("fcmv1 mock server path:", p)
	mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			if verbose {
				log.Printf("reqtime:%f proto:%s method:%s path:%s host:%s", reqtime(start), r.Proto, r.Method, r.URL.Path, r.RemoteAddr)
			}
		}()

		// sets the response time from FCM server
		time.Sleep(time.Millisecond*200 + time.Millisecond*(time.Duration(rand.Int63n(200)-100)))
		token := r.Header.Get("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")

		w.Header().Set("Content-Type", ApplicationJSON)
		switch token {
		case fcmv1.InvalidArgument:
			createFCMv1ErrorResponse(w, http.StatusBadRequest, fcmv1.InvalidArgument)
		case fcmv1.Unregistered:
			createFCMv1ErrorResponse(w, http.StatusNotFound, fcmv1.Unregistered)
		case fcmv1.Unavailable:
			createFCMv1ErrorResponse(w, http.StatusServiceUnavailable, fcmv1.Unavailable)
		case fcmv1.Internal:
			createFCMv1ErrorResponse(w, http.StatusInternalServerError, fcmv1.Internal)
		case fcmv1.QuotaExceeded:
			createFCMv1ErrorResponse(w, http.StatusTooManyRequests, fcmv1.QuotaExceeded)
		default:
			enc := json.NewEncoder(w)
			enc.Encode(fcmv1.ResponseBody{
				Name: "ok",
			})
		}
	})

	return mux
}

func createFCMv1ErrorResponse(w http.ResponseWriter, code int, status string) error {
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	return enc.Encode(fcmv1.ResponseBody{
		Error: &fcmv1.FCMError{
			Status:  status,
			Message: "mock error:" + status,
		},
	})
}
