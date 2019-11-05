package gunfish

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/kayac/Gunfish/apns"
	"github.com/kayac/Gunfish/config"
	"github.com/kayac/Gunfish/fcm"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

// Supervisor monitor mutiple http2 clients.
type Supervisor struct {
	queue   chan *[]Request // supervisor's queue that recieves POST requests.
	retryq  chan Request    // enqueues this retry queue when to failed to send notification on the http layer.
	errq    chan Error      // enqueues this command queue when to get error response from apns.
	exit    chan struct{}   // exit channel is used to stop the supervisor.
	ticker  *time.Ticker    // ticker checks retry queue that has notifications to resend periodically.
	wgrp    *sync.WaitGroup
	workers []*Worker
}

// Worker sends notification to apns.
type Worker struct {
	ac             *apns.Client
	fc             *fcm.Client
	queue          chan Request
	respq          chan SenderResponse
	wgrp           *sync.WaitGroup
	sn             int
	id             int
	errorHandler   func(Request, *http.Response, error)
	successHandler func(Request, *http.Response)
}

// SenderResponse is responses to worker from sender.
type SenderResponse struct {
	Results  []Result `json:"response"`
	RespTime float64  `json:"response_time"`
	Req      Request  `json:"request"`
	Err      error    `json:"error_msg"`
	UID      string   `json:"resp_uid"`
}

// Command has execute command and input stream.
type Error struct {
	input []byte
}

// EnqueueClientRequest enqueues request to supervisor's queue from external application service
func (s *Supervisor) EnqueueClientRequest(reqs *[]Request) error {
	logf := logrus.Fields{
		"type":             "supervisor",
		"request_size":     len(*reqs),
		"queue_size":       len(s.queue),
		"retry_queue_size": len(s.retryq),
	}

	select {
	case s.queue <- reqs:
		LogWithFields(logf).Debugf("Enqueued request from provider.")
	default:
		LogWithFields(logf).Warnf("Supervisor's queue is full.")
		return fmt.Errorf("Supervisor's queue is full")
	}

	return nil
}

// StartSupervisor starts supervisor
func StartSupervisor(conf *config.Config) (Supervisor, error) {
	// Calculates each worker queue size to accept requests with a given parameter of requests per sec as flow rate.
	var wqSize int
	tp := ((conf.Provider.RequestQueueSize * int(AverageResponseTime/time.Millisecond)) / 1000) / SenderNum
	dif := (RequestPerSec - conf.Provider.RequestQueueSize/tp)
	if dif > 0 {
		wqSize = dif * int(FlowRateInterval/time.Second) / conf.Provider.WorkerNum
	} else {
		wqSize = -1 * dif * int(FlowRateInterval/time.Second) / conf.Provider.WorkerNum
	}

	// Initialize Supervisor
	swgrp := &sync.WaitGroup{}
	s := Supervisor{
		queue:  make(chan *[]Request, conf.Provider.QueueSize),
		retryq: make(chan Request, conf.Provider.RequestQueueSize*conf.Provider.WorkerNum),
		errq:   make(chan Error, wqSize*conf.Provider.WorkerNum),
		exit:   make(chan struct{}, 1),
		ticker: time.NewTicker(RetryWaitTime),
		wgrp:   swgrp,
	}
	LogWithFields(logrus.Fields{}).Infof("Retry queue size: %d", cap(s.retryq))
	LogWithFields(logrus.Fields{}).Infof("Queue size: %d", cap(s.queue))

	// Time ticker to retry to send
	go func() {
		for {
			select {
			case <-s.ticker.C:
				// Number of request retry send at once.
				for cnt := 0; cnt < RetryOnceCount; cnt++ {
					select {
					case req := <-s.retryq:
						reqs := &[]Request{req}
						select {
						case s.queue <- reqs:
							LogWithFields(logrus.Fields{"type": "retry", "resend_cnt": req.Tries}).
								Debugf("Enqueue to retry to send notification.")
						default:
							LogWithFields(logrus.Fields{"type": "retry"}).
								Infof("Could not retry to enqueue because the supervisor queue is full.")
						}
					default:
						break
					}
				}
			case <-s.exit:
				s.ticker.Stop()
				return
			}
		}
	}()

	if err := s.startErrorWorkers(conf); err != nil {
		return Supervisor{}, err
	}

	if err := s.startWorkers(conf, wqSize); err != nil {
		return Supervisor{}, err
	}
	return s, nil
}

func (s *Supervisor) startWorkers(conf *config.Config, wqSize int) error {
	// Spawn workers
	var err error
	for i := 0; i < conf.Provider.WorkerNum; i++ {
		var (
			ac *apns.Client
			fc *fcm.Client
		)
		if conf.Apns.Enabled {
			ac, err = apns.NewClient(conf.Apns)
			if err != nil {
				LogWithFields(logrus.Fields{
					"type": "supervisor",
				}).Errorf("%s", err.Error())
				break
			}
		}
		if conf.FCM.Enabled {
			fc, err = fcm.NewClient(conf.FCM.APIKey, nil, fcm.ClientTimeout)
			if err != nil {
				LogWithFields(logrus.Fields{
					"type": "supervisor",
				}).Errorf("%s", err.Error())
				break
			}
		}
		worker := Worker{
			id:    i,
			queue: make(chan Request, wqSize),
			respq: make(chan SenderResponse, wqSize),
			wgrp:  &sync.WaitGroup{},
			sn:    SenderNum,
			ac:    ac,
			fc:    fc,
		}

		s.workers = append(s.workers, &worker)
		s.wgrp.Add(1)
		go s.spawnWorker(worker, conf)
		LogWithFields(logrus.Fields{
			"type":      "worker",
			"worker_id": i,
		}).Debugf("Spawned worker-%d.", i)
	}
	return err
}

func (s *Supervisor) startErrorWorkers(conf *config.Config) error {
	hookCmd := conf.Provider.ErrorHook
	hookTo := conf.Provider.ErrorHookTo
	logf := logrus.Fields{"type": "error_worker"}

	// stdout / stderr
	if hookTo != "" {
		return s.startErrorHookToWorker(hookTo)
	}

	// cmd
	if hookCmd != "" {
		if conf.Provider.ErrorHookCommandPersistent {
			return s.startErrorCmdPersistentWorker(hookCmd)
		} else {
			return s.startErrorCmdWorker(hookCmd, conf.Provider.WorkerNum)
		}
	}

	// not defined
	LogWithFields(logf).Warnf("Neither of error_hook and error_hook_to are not defined.")
	go func() {
		<-s.errq // dispose simply
	}()
	return nil
}

func (s *Supervisor) startErrorHookToWorker(hookTo string) error {
	logf := func() logrus.Fields {
		return logrus.Fields{"type": "error_worker"}
	}
	var out io.Writer
	switch strings.ToLower(hookTo) {
	case "stdout":
		out = os.Stdout
		LogWithFields(logf()).Info("error_hook_to set output to stdout")
	case "stderr":
		out = os.Stderr
		LogWithFields(logf()).Info("error_hook_to set output to stderr")
	default:
		LogWithFields(logf()).Warn("error_hook_to allows stdout or stderr only. dispose hook payloads to /dev/null")
		out = ioutil.Discard
	}
	s.wgrp.Add(1)
	go func() {
		defer s.wgrp.Done()
		w := bufio.NewWriter(out)
		for e := range s.errq {
			w.Write(e.input)
			io.WriteString(w, "\n")
			if err := w.Flush(); err != nil {
				LogWithFields(logf()).Warnf("failed to write to %s: %s", hookTo, err)
				return
			}
		}
	}()
	return nil
}

func (s *Supervisor) startErrorCmdWorker(hookCmd string, workers int) error {
	logf := func() logrus.Fields {
		return logrus.Fields{"type": "error_worker"}
	}
	for i := 0; i < workers; i++ {
		s.wgrp.Add(1)
		go func() {
			defer s.wgrp.Done()
			for e := range s.errq {
				LogWithFields(logf()).Debugf("invoking command: %s %s", hookCmd, string(e.input))
				src := bytes.NewBuffer(e.input)
				out, err := InvokePipe(hookCmd, src)
				if err != nil {
					LogWithFields(logf()).Errorf("(%s) %s", err.Error(), string(out))
				} else {
					LogWithFields(logf()).Debug("Success to execute command")
				}
			}
		}()
	}
	return nil
}

func (s *Supervisor) startErrorCmdPersistentWorker(hookCmd string) error {
	logf := func() logrus.Fields {
		return logrus.Fields{"type": "error_worker"}
	}
	_w, err := InvokePipePersistent(hookCmd)
	if err != nil {
		LogWithFields(logf()).Errorf("failed to invoke command %s", hookCmd)
		return err
	}
	w := bufio.NewWriter(_w)
	s.wgrp.Add(1)
	go func() {
		defer s.wgrp.Done()
		LogWithFields(logf()).Debugf("invoking command: %s", hookCmd)
		for e := range s.errq {
			if w == nil {
				_w, err := InvokePipePersistent(hookCmd)
				if err != nil {
					LogWithFields(logf()).Errorf("failed to invoke command %s", hookCmd)
					LogWithFields(logf()).Warnf("failed to process error hook payload: %s", string(e.input))
					continue
				}
				w = bufio.NewWriter(_w)
			}
			w.Write(e.input)
			w.WriteString("\n")
			if err := w.Flush(); err != nil {
				LogWithFields(logf()).Warnf("failed to write STDIN %s, payload: %s", err, string(e.input))
				w = nil
			}
		}
	}()
	return nil
}

// Shutdown supervisor
func (s *Supervisor) Shutdown() {
	LogWithFields(logrus.Fields{
		"type": "supervisor",
	}).Infoln("Waiting for stopping supervisor...")

	// Waiting for processing notification requests
	zeroCnt := 0
	tryCnt := 0
	for zeroCnt < RestartWaitCount {
		// if 's.counter' is not 0 potentially, here loop should not cancel to wait.
		if len(s.queue)+len(s.errq)+len(s.retryq)+s.workersAllQueueLength() > 0 {
			zeroCnt = 0
			tryCnt++
		} else {
			zeroCnt++
			tryCnt = 0
		}

		// force terminate application waiting for over 2 min.
		// RestartWaitCount: 50
		// ShutdownWaitTime: 10 (msec)
		// 40 * 50 * 6 * 10 (msec) / 1,000 / 60 = 2 (min)
		if tryCnt > RestartWaitCount*40*6 {
			break
		}

		time.Sleep(ShutdownWaitTime)
	}
	close(s.exit)
	close(s.errq)
	s.wgrp.Wait()
	close(s.queue)
	close(s.retryq)

	LogWithFields(logrus.Fields{
		"type": "supervisor",
	}).Infoln("Stoped supervisor.")
}

func (s *Supervisor) spawnWorker(w Worker, conf *config.Config) {
	atomic.AddInt64(&(srvStats.Workers), 1)
	defer func() {
		atomic.AddInt64(&(srvStats.Workers), -1)
		close(w.respq)
		s.wgrp.Done()
	}()

	// Queue of SenderResopnse
	for i := 0; i < w.sn; i++ {
		w.wgrp.Add(1)
		LogWithFields(logrus.Fields{
			"type":      "worker",
			"worker_id": w.id,
		}).Debugf("Spawned a sender-%d-%d.", w.id, i)

		// spawnSender
		go spawnSender(w.queue, w.respq, w.wgrp, w.ac, w.fc)
	}

	func() {
		for {
			select {
			case reqs := <-s.queue:
				w.receiveRequests(reqs)
			case resp := <-w.respq:
				w.receiveResponse(resp, s.retryq, s.errq)
			case <-s.exit:
				return
			}
		}
	}()

	close(w.queue)
	w.wgrp.Wait()
}

func (w *Worker) receiveResponse(resp SenderResponse, retryq chan<- Request, errq chan Error) {
	req := resp.Req

	switch t := req.Notification.(type) {
	case apns.Notification:
		no := req.Notification.(apns.Notification)
		logf := logrus.Fields{
			"type":           "worker",
			"status":         "-",
			"apns_id":        "-",
			"token":          no.Token,
			"payload":        no.Payload,
			"worker_id":      w.id,
			"res_queue_size": len(w.respq),
			"resend_cnt":     req.Tries,
			"response_time":  resp.RespTime,
			"resp_uid":       resp.UID,
		}
		handleAPNsResponse(resp, retryq, errq, logf)
	case fcm.Payload:
		p := req.Notification.(fcm.Payload)
		logf := logrus.Fields{
			"type":           "worker",
			"reg_ids_length": len(p.RegistrationIDs),
			"notification":   p.Notification,
			"data":           p.Data,
			"worker_id":      w.id,
			"res_queue_size": len(w.respq),
			"resend_cnt":     req.Tries,
			"response_time":  resp.RespTime,
			"resp_uid":       resp.UID,
		}
		handleFCMResponse(resp, retryq, errq, logf)
	default:
		LogWithFields(logrus.Fields{"type": "worker"}).Infof("Unknown request type:%s", t)
	}

}

func handleAPNsResponse(resp SenderResponse, retryq chan<- Request, errq chan Error, logf logrus.Fields) {
	req := resp.Req

	// Response handling
	if resp.Err != nil {
		atomic.AddInt64(&(srvStats.ErrCount), 1)
		if len(resp.Results) > 0 {
			result := resp.Results[0]
			for _, key := range result.ExtraKeys() {
				logf[key] = result.ExtraValue(key)
			}
			logf["status"] = result.Status()
			LogWithFields(logf).Errorf("%s", resp.Err)
			// Error handling
			onResponse(result, errq)
		} else {
			// if 'result' is nil, HTTP connection error with APNS.
			retry(retryq, req, errors.New("http connection error between APNs"), logf)
		}
	} else {
		atomic.AddInt64(&(srvStats.SentCount), 1)
		if len(resp.Results) > 0 {
			result := resp.Results[0]
			for _, key := range result.ExtraKeys() {
				logf[key] = result.ExtraValue(key)
			}
			if err := result.Err(); err != nil {
				atomic.AddInt64(&(srvStats.ErrCount), 1)

				// retry when provider auhentication token is expired
				if err.Error() == apns.ExpiredProviderToken.String() {
					retry(retryq, req, err, logf)
				}

				onResponse(result, errq)
				LogWithFields(logf).Errorf("%s", err)
			} else {
				onResponse(result, errq)
				LogWithFields(logf).Info("Succeeded to send a notification")
			}
		}
	}
}

func handleFCMResponse(resp SenderResponse, retryq chan<- Request, errq chan Error, logf logrus.Fields) {
	if resp.Err != nil {
		req := resp.Req
		LogWithFields(logf).Warnf("unexpected response. reason: %s", resp.Err.Error())
		if req.Tries < SendRetryCount {
			req.Tries++
			atomic.AddInt64(&(srvStats.RetryCount), 1)
			logf["resend_cnt"] = req.Tries

			select {
			case retryq <- req:
				LogWithFields(logf).
					Debugf("Retry to enqueue into retryq because of http connection error with FCM.")
			default:
				LogWithFields(logf).
					Warnf("Supervisor retry queue is full.")
			}
		} else {
			LogWithFields(logf).
				Warnf("Retry count is over than %d. Could not deliver notification.", SendRetryCount)
		}
		return
	}

	for _, result := range resp.Results {
		// success when Error is nothing
		err := result.Err()
		if err == nil {
			atomic.AddInt64(&(srvStats.SentCount), 1)
			LogWithFields(logf).Info("Succeeded to send a notification")
			continue
		}
		// handle error response each registration_id
		atomic.AddInt64(&(srvStats.ErrCount), 1)
		switch err.Error() {
		case fcm.InvalidRegistration.String(), fcm.NotRegistered.String():
			onResponse(result, errq)
			LogWithFields(logf).Errorf("%s", err)
		default:
			LogWithFields(logf).Errorf("Unknown error message: %s", err)
		}
	}
}

func (w *Worker) receiveRequests(reqs *[]Request) {
	logf := logrus.Fields{
		"type":              "worker",
		"worker_id":         w.id,
		"worker_queue_size": len(w.queue),
		"request_size":      len(*reqs),
	}

	for _, req := range *reqs {
		select {
		case w.queue <- req:
			LogWithFields(logf).
				Debugf("Enqueue request into worker's queue")
		}
	}
}

func spawnSender(wq <-chan Request, respq chan<- SenderResponse, wgrp *sync.WaitGroup, ac *apns.Client, fc *fcm.Client) {
	defer wgrp.Done()
	for req := range wq {
		var sres SenderResponse
		switch t := req.Notification.(type) {
		case apns.Notification:
			if ac == nil {
				LogWithFields(logrus.Fields{"type": "sender"}).
					Errorf("apns client is not present")
				continue
			}
			no := req.Notification.(apns.Notification)
			start := time.Now()
			results, err := ac.Send(no)
			respTime := time.Now().Sub(start).Seconds()
			rs := make([]Result, 0, len(results))
			for _, v := range results {
				rs = append(rs, v)
			}
			sres = SenderResponse{
				Results:  rs,
				RespTime: respTime,
				Req:      req, // Must copy
				Err:      err,
				UID:      uuid.NewV4().String(),
			}
		case fcm.Payload:
			if fc == nil {
				LogWithFields(logrus.Fields{"type": "sender"}).
					Errorf("fcm client is not present")
				continue
			}
			p := req.Notification.(fcm.Payload)
			start := time.Now()
			results, err := fc.Send(p)
			respTime := time.Now().Sub(start).Seconds()
			rs := make([]Result, 0, len(results))
			for _, v := range results {
				rs = append(rs, v)
			}
			sres = SenderResponse{
				Results:  rs,
				RespTime: respTime,
				Req:      req,
				Err:      err,
				UID:      uuid.NewV4().String(),
			}
		default:
			LogWithFields(logrus.Fields{"type": "sender"}).
				Errorf("Unknown request data type: %s", t)
			continue
		}

		select {
		case respq <- sres:
			LogWithFields(logrus.Fields{"type": "sender", "resp_queue_size": len(respq)}).
				Debugf("Enqueue response into respq.")
		default:
			LogWithFields(logrus.Fields{"type": "sender", "resp_queue_size": len(respq)}).
				Warnf("Response queue is full.")
		}
	}
}

func (s Supervisor) workersAllQueueLength() int {
	sum := 0
	for _, w := range s.workers {
		sum += len(w.queue) + len(w.respq)
	}
	return sum
}

func onResponse(result Result, errq chan<- Error) {
	logf := logrus.Fields{
		"provider": result.Provider(),
		"type":     "on_response",
		"token":    result.RecipientIdentifier(),
	}
	for _, key := range result.ExtraKeys() {
		logf[key] = result.ExtraValue(key)
	}
	// on error handler
	if err := result.Err(); err != nil {
		errorResponseHandler.OnResponse(result)
	} else {
		successResponseHandler.OnResponse(result)
	}

	b, _ := result.MarshalJSON()
	error := Error{
		input: b,
	}
	select {
	case errq <- error:
		LogWithFields(logf).Debugf("Enqueue error: %v", error)
	default:
		LogWithFields(logf).Warnf("Error queue is full. dropping error: %v", error)
	}
}

func InvokePipePersistent(hook string) (io.WriteCloser, error) {
	cmd := exec.Command("sh", "-c", hook)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed: %v %s", cmd, err.Error())
	}

	// merge std(out|err) of command to gunfish
	if OutputHookStdout {
		cmd.Stdout = os.Stdout
	}
	if OutputHookStderr {
		cmd.Stderr = os.Stderr
	}
	return stdin, cmd.Start()
}

func InvokePipe(hook string, src io.Reader) ([]byte, error) {
	logf := logrus.Fields{"type": "invoke_pipe"}
	cmd := exec.Command("sh", "-c", hook)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed: %v %s", cmd, err.Error())
	}

	var b bytes.Buffer
	// merge std(out|err) of command to gunfish
	if OutputHookStdout {
		cmd.Stdout = os.Stdout
	} else {
		cmd.Stdout = &b
	}
	if OutputHookStderr {
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stderr = &b
	}

	// src copy to cmd.stdin
	_, err = io.Copy(stdin, src)
	if e, ok := err.(*os.PathError); ok && e.Err == syscall.EPIPE {
		LogWithFields(logf).Errorf(e.Error())
	} else if err != nil {
		LogWithFields(logf).Errorf("failed to write STDIN of %s. %s", hook, err.Error())
	}
	stdin.Close()

	err = cmd.Run()
	return b.Bytes(), err
}

func retry(retryq chan<- Request, req Request, err error, logf logrus.Fields) {
	if req.Tries < SendRetryCount {
		req.Tries++
		atomic.AddInt64(&(srvStats.RetryCount), 1)
		logf["resend_cnt"] = req.Tries

		select {
		case retryq <- req:
			LogWithFields(logf).
				Debugf("%s: Retry to enqueue into retryq.", err.Error())
		default:
			LogWithFields(logf).
				Warnf("Supervisor retry queue is full.")
		}
	} else {
		LogWithFields(logf).
			Warnf("Retry count is over than %d. Could not deliver notification.", SendRetryCount)
	}
}
