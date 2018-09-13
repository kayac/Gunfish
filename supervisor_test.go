package gunfish

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kayac/Gunfish/apns"
	"github.com/kayac/Gunfish/config"
	"github.com/sirupsen/logrus"
)

var (
	conf, _ = config.LoadConfig("./test/gunfish_test.toml")
)

type TestResponseHandler struct {
	scoreboard map[string]*int
	hook       string
	mu         sync.Mutex
}

func (tr *TestResponseHandler) Countup(name string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	*(tr.scoreboard[name])++
}

func (tr TestResponseHandler) OnResponse(result Result) {
	if err := result.Err(); err != nil {
		logrus.Warnf(err.Error())
		tr.Countup(err.Error())
	} else {
		tr.Countup("success")
	}
}

func (tr TestResponseHandler) HookCmd() string {
	return tr.hook
}

func init() {
	logrus.SetLevel(logrus.WarnLevel)
	conf.Apns.Host = MockServer
}

func TestStartAndStopSupervisor(t *testing.T) {
	sup, err := StartSupervisor(&conf)
	if err != nil {
		t.Errorf("cannot start supvisor: %s", err.Error())
	}

	sup.Shutdown()

	if _, ok := <-sup.queue; ok == true {
		t.Errorf("not closed channel: %v", sup.queue)
	}

	if _, ok := <-sup.retryq; ok == true {
		t.Errorf("not closed channel: %v", sup.queue)
	}

	if _, ok := <-sup.cmdq; ok == true {
		t.Errorf("not closed channel: %v", sup.queue)
	}
}

func TestEnqueueRequestToSupervisor(t *testing.T) {
	// Prepare
	score := make(map[string]*int, 5)
	boardList := []string{
		apns.MissingTopic.String(),
		apns.BadDeviceToken.String(),
		apns.Unregistered.String(),
		apns.ExpiredProviderToken.String(),
		"success",
	}
	for _, v := range boardList {
		x := 0
		score[v] = &x
	}

	etr := TestResponseHandler{
		scoreboard: score,
		hook:       conf.Provider.ErrorHook,
		mu:         sync.Mutex{},
	}
	str := TestResponseHandler{
		scoreboard: score,
		mu:         sync.Mutex{},
	}
	InitErrorResponseHandler(etr)
	InitSuccessResponseHandler(str)

	sup, err := StartSupervisor(&conf)
	if err != nil {
		t.Errorf("cannot start supervisor: %s", err.Error())
	}
	defer sup.Shutdown()

	// test success requests
	reqs := repeatRequestData("1122334455667788112233445566778811223344556677881122334455667788", 10)
	for range []int{0, 1, 2, 3, 4, 5, 6} {
		sup.EnqueueClientRequest(&reqs)
	}

	twg := sync.WaitGroup{}
	twg.Add(1)
	expectSuccess := 70
	go func() {
		defer twg.Done()
		endTime := time.Now().Add(time.Second * 10)
		for time.Now().Before(endTime) {
			if *(score["success"]) == expectSuccess {
				return
			}
			time.Sleep(time.Second * 1)
		}
	}()
	twg.Wait()
	if g, w := *(score["success"]), expectSuccess; g != w {
		t.Errorf("not match success count: got %d want %d", g, w)
	}

	// test error requests
	testTable := []struct {
		errToken string
		num      int
		errCode  apns.ErrorResponseCode
		expect   int
	}{
		{
			errToken: "missingtopic",
			num:      1,
			errCode:  apns.MissingTopic,
			expect:   1,
		},
		{
			errToken: "unregistered",
			num:      1,
			errCode:  apns.Unregistered,
			expect:   1,
		},
		{
			errToken: "baddevicetoken",
			num:      1,
			errCode:  apns.BadDeviceToken,
			expect:   1,
		},
		{
			errToken: "expiredprovidertoken",
			num:      1,
			errCode:  apns.ExpiredProviderToken,
			expect:   1 * SendRetryCount,
		},
	}

	for _, tt := range testTable {
		reqs := repeatRequestData(tt.errToken, tt.num)
		sup.EnqueueClientRequest(&reqs)
		errReason := tt.errCode.String()

		twg.Add(1)
		go func() {
			defer twg.Done()
			endTime := time.Now().Add(time.Second * 10)
			for time.Now().Before(endTime) {
				if *(score[errReason]) == tt.expect {
					return
				}
				time.Sleep(time.Second * 1)
			}
		}()
		twg.Wait()

		if g, w := *(score[errReason]), tt.expect; g != w {
			t.Errorf("not match %s count: got %d want %d", errReason, g, w)
		}
	}
}

func repeatRequestData(token string, num int) []Request {
	var reqs []Request
	for i := 0; i < num; i++ {
		// Create request
		aps := &apns.APS{
			Alert: &apns.Alert{
				Title: "test",
				Body:  "message",
			},
			Sound: "default",
		}
		payload := apns.Payload{}
		payload.APS = aps

		req := Request{
			Notification: apns.Notification{
				Token:   token,
				Payload: payload,
			},
			Tries: 0,
		}

		reqs = append(reqs, req)
	}
	return reqs
}

func TestSuccessOrFailureInvoke(t *testing.T) {
	// prepare SenderResponse
	token := "invalid token"
	sre := fmt.Errorf(apns.Unregistered.String())
	aps := &apns.APS{
		Alert: apns.Alert{
			Title: "test",
			Body:  "hoge message",
		},
		Badge: 1,
		Sound: "default",
	}
	payload := apns.Payload{}
	payload.APS = aps
	sr := SenderResponse{
		Req: Request{
			Notification: apns.Notification{
				Token:   token,
				Payload: payload,
			},
			Tries: 0,
		},
		RespTime: 0.0,
		Err:      sre,
	}
	j, err := json.Marshal(sr)
	if err != nil {
		t.Errorf(err.Error())
	}

	// Succeed to invoke
	src := bytes.NewBuffer(j)
	out, err := invokePipe(`cat`, src)
	if err != nil {
		t.Errorf("result: %s, err: %s", string(out), err.Error())
	}

	// checks Unmarshaled result
	if string(out) == `{}` {
		t.Errorf("output of result is empty: %s", string(out))
	}
	if string(out) != string(j) {
		t.Errorf("Expected result %s but got %s", j, string(out))
	}

	// Failure to invoke
	src = bytes.NewBuffer(j)
	out, err = invokePipe(`expr 1 1`, src)
	if err == nil {
		t.Errorf("Expected failure to invoke command: %s", string(out))
	}

	// tests command including Pipe '|'
	src = bytes.NewBuffer(j)
	out, err = invokePipe(`cat | head -n 10 | tail -n 10`, src)
	if err != nil {
		t.Errorf("result: %s, err: %s", string(out), err.Error())
	}
	if string(out) != string(j) {
		t.Errorf("Expected result '%s' but got %s", j, string(out))
	}

	// Must fail
	src = bytes.NewBuffer(j)
	out, err = invokePipe(`echo 'Failure test'; false`, src)
	if err == nil {
		t.Errorf("result: %s, err: %s", string(out), err.Error())
	}
	if fmt.Sprintf("%s", err.Error()) != `exit status 1` {
		t.Errorf("invalid err message: %s", err.Error())
	}

	// stdout be not captured
	OutputHookStdout = true
	src = bytes.NewBuffer(j)
	out, err = invokePipe(`cat; echo 'this is error.' 1>&2`, src)
	if len(out) != 15 {
		t.Errorf("hooks stdout must not be captured: %s", out)
	}

	// stderr
	OutputHookStderr = true
	src = bytes.NewBuffer(j)
	out, err = invokePipe(`cat; echo 'this is error.' 1>&2`, src)
	if len(out) != 0 {
		t.Errorf("hooks stderr must not be captured: %s", out)
	}
}
