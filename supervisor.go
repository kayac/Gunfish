package gunfish

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/kayac/Gunfish/apns"
	"github.com/kayac/Gunfish/config"
	"github.com/kayac/Gunfish/fcmv1"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

// Supervisor monitor mutiple http2 clients.
type Supervisor struct {
	queue   chan *[]Request // supervisor's queue that recieves POST requests.
	retryq  chan Request    // enqueues this retry queue when to failed to send notification on the http layer.
	cmdq    chan Command    // enqueues this command queue when to get error response from apns.
	exit    chan struct{}   // exit channel is used to stop the supervisor.
	ticker  *time.Ticker    // ticker checks retry queue that has notifications to resend periodically.
	wgrp    *sync.WaitGroup
	workers []*Worker
}

// Worker sends notification to apns.
type Worker struct {
	ac    *apns.Client
	fcv1  *fcmv1.Client
	queue chan Request
	respq chan SenderResponse
	wgrp  *sync.WaitGroup
	sn    int
	id    int
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
type Command struct {
	command string
	input   []byte
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
		cmdq:   make(chan Command, wqSize*conf.Provider.WorkerNum),
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
						var delay time.Duration
						if RetryBackoff {
							delay = time.Duration(math.Pow(float64(req.Tries), 2)) * 100 * time.Millisecond
						}
						time.AfterFunc(delay, func() {
							reqs := &[]Request{req}
							select {
							case s.queue <- reqs:
								LogWithFields(logrus.Fields{"delay": delay, "type": "retry", "resend_cnt": req.Tries}).
									Debugf("Enqueue to retry to send notification.")
							default:
								LogWithFields(logrus.Fields{"delay": delay, "type": "retry"}).
									Infof("Could not retry to enqueue because the supervisor queue is full.")
							}
						})
					default:
					}
				}
			case <-s.exit:
				s.ticker.Stop()
				return
			}
		}
	}()

	// spawn command
	for i := 0; i < conf.Provider.WorkerNum; i++ {
		s.wgrp.Add(1)
		go func() {
			logf := logrus.Fields{"type": "cmd_worker"}
			for c := range s.cmdq {
				LogWithFields(logf).Debugf("invoking command: %s %s", c.command, string(c.input))
				src := bytes.NewBuffer(c.input)
				out, err := InvokePipe(c.command, src)
				if err != nil {
					LogWithFields(logf).Errorf("(%s) %s", err.Error(), string(out))
				} else {
					LogWithFields(logf).Debugf("Success to execute command")
				}
			}
			s.wgrp.Done()
		}()
	}

	// Spawn workers
	var err error
	for i := 0; i < conf.Provider.WorkerNum; i++ {
		var (
			ac   *apns.Client
			fcv1 *fcmv1.Client
		)
		if conf.Apns.Enabled {
			ac, err = apns.NewClient(conf.Apns)
			if err != nil {
				LogWithFields(logrus.Fields{
					"type": "supervisor",
				}).Errorf("faile to new client for apns: %s", err.Error())
				break
			}
		}
		if conf.FCM.Enabled {
			return Supervisor{}, errors.New("FCM legacy is not supported")
		}
		if conf.FCMv1.Enabled {
			fcv1, err = fcmv1.NewClient(conf.FCMv1.TokenSource, conf.FCMv1.ProjectID, conf.FCMv1.Endpoint, fcmv1.ClientTimeout)
			if err != nil {
				LogWithFields(logrus.Fields{
					"type": "supervisor",
				}).Errorf("failed to new client for fcmv1: %s", err.Error())
				break
			}
		}
		worker := Worker{
			id:    i,
			queue: make(chan Request, wqSize),
			respq: make(chan SenderResponse, wqSize*100),
			wgrp:  &sync.WaitGroup{},
			sn:    SenderNum,
			ac:    ac,
			fcv1:  fcv1,
		}

		s.workers = append(s.workers, &worker)
		s.wgrp.Add(1)
		go s.spawnWorker(worker)
		LogWithFields(logrus.Fields{
			"type":      "worker",
			"worker_id": i,
		}).Debugf("Spawned worker-%d.", i)
	}

	if err != nil {
		return Supervisor{}, err
	}
	return s, nil
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
		if len(s.queue)+len(s.cmdq)+len(s.retryq)+s.workersAllQueueLength() > 0 {
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
	close(s.cmdq)
	s.wgrp.Wait()
	close(s.queue)
	close(s.retryq)

	LogWithFields(logrus.Fields{
		"type": "supervisor",
	}).Infoln("Stoped supervisor.")
}

func (s *Supervisor) spawnWorker(w Worker) {
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
		go spawnSender(w.queue, w.respq, w.wgrp, w.ac, w.fcv1)
	}

	func() {
		for {
			select {
			case reqs := <-s.queue:
				w.receiveRequests(reqs)
			case resp := <-w.respq:
				w.receiveResponse(resp, s.retryq, s.cmdq)
			case <-s.exit:
				return
			}
		}
	}()

	close(w.queue)
	w.wgrp.Wait()
}

func (w *Worker) receiveResponse(resp SenderResponse, retryq chan<- Request, cmdq chan Command) {
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
		handleAPNsResponse(resp, retryq, cmdq, logf)
	case fcmv1.Payload:
		p := req.Notification.(fcmv1.Payload)
		logf := logrus.Fields{
			"type":           "worker",
			"token":          p.Message.Token,
			"worker_id":      w.id,
			"res_queue_size": len(w.respq),
			"resend_cnt":     req.Tries,
			"response_time":  resp.RespTime,
			"resp_uid":       resp.UID,
		}
		handleFCMResponse(resp, retryq, cmdq, logf)
	default:
		LogWithFields(logrus.Fields{"type": "worker"}).Infof("Unknown response type:%s", t)
	}

}

func handleAPNsResponse(resp SenderResponse, retryq chan<- Request, cmdq chan Command, logf logrus.Fields) {
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
			onResponse(result, errorResponseHandler.HookCmd(), cmdq)
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

				onResponse(result, errorResponseHandler.HookCmd(), cmdq)
				LogWithFields(logf).Errorf("%s", err)
			} else {
				onResponse(result, "", cmdq)
				LogWithFields(logf).Info("Succeeded to send a notification")
			}
		}
	}
}

func handleFCMResponse(resp SenderResponse, retryq chan<- Request, cmdq chan Command, logf logrus.Fields) {
	if resp.Err != nil {
		req := resp.Req
		LogWithFields(logf).Warnf("response is nil. reason: %s", resp.Err.Error())
		retry(retryq, req, resp.Err, logf)
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
		switch err.Error() {
		case fcmv1.Internal, fcmv1.Unavailable:
			LogWithFields(logf).Warn("retrying:", err)
			retry(retryq, resp.Req, err, logf)
		case fcmv1.QuotaExceeded:
			LogWithFields(logf).Warn("retrying after 1 min:", err)
			time.AfterFunc(time.Minute, func() { retry(retryq, resp.Req, err, logf) })
		case fcmv1.Unregistered, fcmv1.InvalidArgument, fcmv1.NotFound:
			LogWithFields(logf).Errorf("calling error hook: %s", err)
			atomic.AddInt64(&(srvStats.ErrCount), 1)
			onResponse(result, errorResponseHandler.HookCmd(), cmdq)
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

func spawnSender(wq <-chan Request, respq chan<- SenderResponse, wgrp *sync.WaitGroup, ac *apns.Client, fcv1 *fcmv1.Client) {
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
			respTime := time.Since(start).Seconds()
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
		case fcmv1.Payload:
			if fcv1 == nil {
				LogWithFields(logrus.Fields{"type": "sender"}).
					Errorf("fcmv1 client is not present")
				continue
			}
			p := req.Notification.(fcmv1.Payload)
			start := time.Now()
			results, err := fcv1.Send(p)
			respTime := time.Since(start).Seconds()
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

func onResponse(result Result, cmd string, cmdq chan<- Command) {
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

	if cmd == "" {
		return
	}

	b, _ := result.MarshalJSON()
	command := Command{
		command: cmd,
		input:   b,
	}
	select {
	case cmdq <- command:
		LogWithFields(logf).Debugf("Enqueue command: %s < %s", command.command, string(b))
	default:
		LogWithFields(logf).Warnf("Command queue is full, so could not execute commnad: %v", command)
	}
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
		LogWithFields(logf).Errorf("failed to write STDIN: cmd( %s ), error( %s )", hook, err.Error())
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
