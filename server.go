package gunfish

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	stats_api "github.com/fukata/golang-stats-api-handler"
	"github.com/kayac/Gunfish/apns"
	"github.com/kayac/Gunfish/config"
	"github.com/kayac/Gunfish/fcm"
	"github.com/kayac/Gunfish/fcmv1"
	"github.com/lestrrat-go/server-starter/listener"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/netutil"
)

// Provider defines Gunfish httpHandler and has a state
// of queue which is shared by the supervisor.
type Provider struct {
	Sup Supervisor
}

// ResponseHandler provides you to implement handling on success or on error response from apns.
// Therefore, you can specifies hook command which is set at toml file.
type ResponseHandler interface {
	OnResponse(Result)
	HookCmd() string
}

// DefaultResponseHandler is the default ResponseHandler if not specified.
type DefaultResponseHandler struct {
	Hook string
}

// OnResponse is performed when to receive result from APNs or FCM.
func (rh DefaultResponseHandler) OnResponse(result Result) {
}

// HookCmd returns hook command to execute after getting response from APNS
// only when to get error response.
func (rh DefaultResponseHandler) HookCmd() string {
	return rh.Hook
}

// StartServer starts an apns provider server on http.
func StartServer(conf config.Config, env Environment) {
	// Initialize DefaultResponseHandler if response handlers are not defined.
	if successResponseHandler == nil {
		InitSuccessResponseHandler(DefaultResponseHandler{})
	}

	if errorResponseHandler == nil {
		InitErrorResponseHandler(DefaultResponseHandler{Hook: conf.Provider.ErrorHook})
	}

	// Init Provider
	srvStats = NewStats(conf)
	prov := &Provider{}

	srvStats.DebugPort = conf.Provider.DebugPort
	LogWithFields(logrus.Fields{
		"type": "provider",
	}).Infof("Size of POST request queue is %d", conf.Provider.QueueSize)

	// Set APNS host addr according of environment
	if env == Production {
		conf.Apns.Host = ProdServer
	} else if env == Development {
		conf.Apns.Host = DevServer
	} else if env == Test {
		conf.Apns.Host = MockServer
	}

	// start supervisor
	sup, err := StartSupervisor(&conf)
	if err != nil {
		LogWithFields(logrus.Fields{
			"type": "provider",
		}).Fatalf("Failed to start Gunfish: %s", err.Error())
	}
	prov.Sup = sup

	LogWithFields(logrus.Fields{
		"type": "supervisor",
	}).Infof("Starts supervisor at %s", Production.String())

	// StartServer listener
	listeners, err := listener.ListenAll()
	if err != nil {
		LogWithFields(logrus.Fields{
			"type": "provider",
		}).Infof("%s. If you want graceful to restart Gunfish, you should use 'starter_server' (github.com/lestrrat/go-server-starter).", err)
	}

	// Start gunfish provider server
	var lis net.Listener
	if err == listener.ErrNoListeningTarget {
		// Fallback if not running under ServerStarter
		service := fmt.Sprintf(":%d", conf.Provider.Port)
		lis, err = net.Listen("tcp", service)
		if err != nil {
			LogWithFields(logrus.Fields{
				"type": "provider",
			}).Error(err)
			sup.Shutdown()
			return
		}
	} else {
		if l, ok := listeners[0].Addr().(*net.TCPAddr); ok && l.Port != conf.Provider.Port {
			LogWithFields(logrus.Fields{
				"type": "provider",
			}).Infof("'start_server' starts on :%d", l.Port)
		}
		// Starts Gunfish under ServerStarter.
		conf.Provider.Port = listeners[0].Addr().(*net.TCPAddr).Port
		lis = listeners[0]
	}

	// If many connections establishs between Gunfish provider and your application,
	// Gunfish provider would be overload, and decrease performance.
	llis := netutil.LimitListener(lis, conf.Provider.MaxConnections)

	// Start Gunfish provider
	LogWithFields(logrus.Fields{
		"type": "provider",
	}).Infof("Starts provider on :%d ...", conf.Provider.Port)

	mux := http.NewServeMux()
	if conf.Apns.Enabled {
		LogWithFields(logrus.Fields{
			"type": "provider",
		}).Infof("Enable endpoint /push/apns")
		mux.HandleFunc("/push/apns", prov.PushAPNsHandler())
	}
	if conf.FCM.Enabled {
		LogWithFields(logrus.Fields{
			"type": "provider",
		}).Infof("Enable endpoint /push/fcm")
		mux.HandleFunc("/push/fcm", prov.PushFCMHandler(false))
	}
	if conf.FCMv1.Enabled {
		LogWithFields(logrus.Fields{
			"type": "provider",
		}).Infof("Enable endpoint /push/fcm/v1")
		mux.HandleFunc("/push/fcm/v1", prov.PushFCMHandler(true))
	}
	mux.HandleFunc("/stats/app", prov.StatsHandler())
	mux.HandleFunc("/stats/profile", stats_api.Handler)

	srv := &http.Server{Handler: mux}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if err := srv.Serve(llis); err != nil {
			LogWithFields(logrus.Fields{}).Error(err)
		}
		wg.Done()
	}()

	// signal handling
	wg.Add(1)
	go startSignalReciever(&wg, srv)

	// wait for server shutdown complete
	wg.Wait()

	// if Gunfish server stop, Close queue
	LogWithFields(logrus.Fields{
		"type": "provider",
	}).Info("Stopping server")

	// if Gunfish server stop, Close queue
	sup.Shutdown()
}

func (prov *Provider) PushAPNsHandler() http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		atomic.AddInt64(&(srvStats.RequestCount), 1)

		// Method Not Alllowed
		if err := validateMethod(res, req); err != nil {
			logrus.Warn(err)
			return
		}

		// Parse request body
		c := req.Header.Get("Content-Type")
		var ps []PostedData
		switch c {
		case ApplicationXW3FormURLEncoded:
			body := req.FormValue("json")
			if err := json.Unmarshal([]byte(body), &ps); err != nil {
				LogWithFields(logrus.Fields{}).Warnf("%s: %s", err, body)
				res.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(res, `{"reason": "%s"}`, err.Error())
				return
			}
		case ApplicationJSON:
			decoder := json.NewDecoder(req.Body)
			if err := decoder.Decode(&ps); err != nil {
				LogWithFields(logrus.Fields{}).Warnf("%s: %v", err, ps)
				res.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(res, `{"reason": "%s"}`, err.Error())
				return
			}
		default:
			// Unsupported Media Type
			logrus.Warnf("Unsupported Media Type: %s", c)
			res.WriteHeader(http.StatusUnsupportedMediaType)
			fmt.Fprintf(res, `{"reason":"Unsupported Media Type"}`)
			return
		}

		// Validates posted data
		if err := validatePostedData(ps); err != nil {
			res.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(res, `{"reason":"%s"}`, err.Error())
			return
		}

		// Create requests
		reqs := make([]Request, len(ps))
		for i, p := range ps {
			switch t := p.Payload.Alert.(type) {
			case map[string]interface{}:
				var alert apns.Alert
				mapToAlert(t, &alert)
				p.Payload.Alert = alert
			}

			req := Request{
				Notification: apns.Notification{
					Header:  p.Header,
					Token:   p.Token,
					Payload: p.Payload,
				},
				Tries: 0,
			}

			reqs[i] = req
		}

		// enqueues one request into supervisor's queue.
		if err := prov.Sup.EnqueueClientRequest(&reqs); err != nil {
			setRetryAfter(res, req, err.Error())
			return
		}

		// success
		res.WriteHeader(http.StatusOK)
		fmt.Fprint(res, "{\"result\": \"ok\"}")
	})
}

func (prov *Provider) PushFCMHandler(v1 bool) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		atomic.AddInt64(&(srvStats.RequestCount), 1)

		// Method Not Alllowed
		if err := validateMethod(res, req); err != nil {
			logrus.Warn(err)
			return
		}

		// only Content-Type application/json
		c := req.Header.Get("Content-Type")
		if c != ApplicationJSON {
			// Unsupported Media Type
			logrus.Warnf("Unsupported Media Type: %s", c)
			res.WriteHeader(http.StatusUnsupportedMediaType)
			fmt.Fprintf(res, `{"reason":"Unsupported Media Type"}`)
			return
		}

		// create request for fcm
		grs, err := newFCMRequests(req.Body, v1)
		if err != nil {
			logrus.Warnf("bad request: %s", err)
			res.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(res, "{\"reason\":\"%s\"}", err.Error())
			return
		}

		// enqueues one request into supervisor's queue.
		if err := prov.Sup.EnqueueClientRequest(&grs); err != nil {
			setRetryAfter(res, req, err.Error())
			return
		}

		// success
		res.WriteHeader(http.StatusOK)
		fmt.Fprint(res, "{\"result\": \"ok\"}")
	})
}

func newFCMRequests(src io.Reader, v1 bool) ([]Request, error) {
	dec := json.NewDecoder(src)
	reqs := []Request{
		Request{
			Tries: 0,
		},
	}
	if v1 {
		var payload fcmv1.Payload
		if err := dec.Decode(&payload); err != nil {
			return nil, err
		}
		reqs[0].Notification = payload
		return reqs, nil
	} else {
		var payload fcm.Payload
		if err := dec.Decode(&payload); err != nil {
			return nil, err
		}
		reqs[0].Notification = payload
		return reqs, nil
	}
}

func validateMethod(res http.ResponseWriter, req *http.Request) error {
	if req.Method != "POST" {
		res.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(res, "{\"reason\":\"Method Not Allowed.\"}")
		return fmt.Errorf("Method Not Allowed: %s", req.Method)
	}
	return nil
}

func setRetryAfter(res http.ResponseWriter, req *http.Request, reason string) {
	now := time.Now().Unix()
	atomic.StoreInt64(&(srvStats.ServiceUnavailableAt), now)
	updateRetryAfterStat(now - srvStats.ServiceUnavailableAt)
	// Retry-After is set seconds
	res.Header().Set("Retry-After", fmt.Sprintf("%d", srvStats.RetryAfter))
	res.WriteHeader(http.StatusServiceUnavailable)
	fmt.Fprintf(res, fmt.Sprintf(`{"reason":"%s"}`, reason))
}

func (prov *Provider) StatsHandler() http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if ok := validateStatsHandler(res, req); ok != true {
			return
		}

		wqs := 0
		for _, w := range prov.Sup.workers {
			wqs += len(w.queue)
		}

		atomic.StoreInt64(&(srvStats.QueueSize), int64(len(prov.Sup.queue)))
		atomic.StoreInt64(&(srvStats.RetryQueueSize), int64(len(prov.Sup.retryq)))
		atomic.StoreInt64(&(srvStats.WorkersQueueSize), int64(wqs))
		atomic.StoreInt64(&(srvStats.CommandQueueSize), int64(len(prov.Sup.cmdq)))
		res.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(res)
		err := encoder.Encode(srvStats.GetStats())
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(res, `{"reason":"Internal Server Error"}`)
			return
		}
	})
}

func validatePostedData(ps []PostedData) error {
	if len(ps) == 0 {
		return fmt.Errorf("PostedData must not be empty: %v", ps)
	}

	if len(ps) > config.MaxRequestSize {
		return fmt.Errorf("PostedData was too long. Be less than %d: %v", config.MaxRequestSize, len(ps))
	}

	for _, p := range ps {
		if p.Payload.APS == nil || p.Token == "" {
			return fmt.Errorf("Payload format was malformed: %v", p.Payload)
		}
	}
	return nil
}

func validateStatsHandler(res http.ResponseWriter, req *http.Request) bool {
	// Method Not Alllowed
	if req.Method != "GET" {
		res.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(res, `{"reason":"Method Not Allowed."}`)
		logrus.Warnf("Method Not Allowed: %s", req.Method)
		return false
	}

	return true
}

func mapToAlert(mapVal map[string]interface{}, alert *apns.Alert) {
	a := reflect.ValueOf(alert).Elem()
	for k, v := range mapVal {
		newk, ok := AlertKeyToField[k]
		if ok == true {
			a.FieldByName(newk).Set(reflect.ValueOf(v))
		} else {
			logrus.Warnf("\"%s\" is not supported key for Alert struct.", k)
		}
	}
}

func startSignalReciever(wg *sync.WaitGroup, srv *http.Server) {
	defer wg.Done()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT)
	s := <-sigChan
	switch s {
	case syscall.SIGHUP:
		LogWithFields(logrus.Fields{
			"type": "provider",
		}).Info("Gunfish recieved SIGHUP signal.")
		srv.Shutdown(context.Background())
	case syscall.SIGTERM:
		LogWithFields(logrus.Fields{
			"type": "provider",
		}).Info("Gunfish recieved SIGTERM signal.")
		srv.Shutdown(context.Background())
	case syscall.SIGINT:
		LogWithFields(logrus.Fields{
			"type": "provider",
		}).Info("Gunfish recieved SIGINT signal. Stopping server now...")
		srv.Shutdown(context.Background())
	}
}

func updateRetryAfterStat(x int64) {
	var nxtRA int64
	if x > int64(ResetRetryAfterSecond/time.Second) {
		nxtRA = int64(RetryAfterSecond / time.Second)
	} else {
		a := int64(math.Log(float64(10/(x+1) + 1)))
		if srvStats.RetryAfter+2*a < int64(ResetRetryAfterSecond/time.Second) {
			nxtRA = srvStats.RetryAfter + 2*a
		} else {
			nxtRA = int64(ResetRetryAfterSecond / time.Second)
		}
	}

	atomic.StoreInt64(&(srvStats.RetryAfter), nxtRA)
}
