package gunfish

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/kayac/Gunfish/apns"
)

var (
	config, _ = LoadConfig("./test/gunfish_test.toml")
)

type TestResponseHandler struct {
	scoreboard map[string]*int
	wg         *sync.WaitGroup
	hook       string
}

func (tr *TestResponseHandler) Done(token string) {
	tr.wg.Done()
}

func (tr *TestResponseHandler) Countup(name string) {
	*(tr.scoreboard[name])++
}

func (tr TestResponseHandler) OnResponse(result Result) {
	tr.wg.Add(1)
	if err := result.Err(); err != nil {
		logrus.Warnf(err.Error())
		if err.Error() == apns.MissingTopic.String() {
			tr.Countup(apns.MissingTopic.String())
		}
		if err.Error() == apns.BadDeviceToken.String() {
			tr.Countup(apns.BadDeviceToken.String())
		}
		if err.Error() == apns.Unregistered.String() {
			tr.Countup(apns.Unregistered.String())
		}
	} else {
		tr.Countup("success")
	}
	tr.Done(result.RecipientIdentifier())
}

func (tr TestResponseHandler) HookCmd() string {
	return tr.hook
}

func init() {
	logrus.SetLevel(logrus.WarnLevel)
	config.Apns.Host = MockServer
}

func TestStartAndStopSupervisor(t *testing.T) {
	sup, err := StartSupervisor(&config)
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

func TestEnqueuRequestToSupervisor(t *testing.T) {
	// Prepare
	wg := sync.WaitGroup{}
	score := make(map[string]*int, 4)
	for _, v := range []string{apns.MissingTopic.String(), apns.BadDeviceToken.String(), apns.Unregistered.String(), "success"} {
		x := 0
		score[v] = &x
	}

	etr := TestResponseHandler{
		wg:         &wg,
		scoreboard: score,
		hook:       config.Provider.ErrorHook,
	}
	str := TestResponseHandler{
		wg:         &wg,
		scoreboard: score,
	}
	InitErrorResponseHandler(etr)
	InitSuccessResponseHandler(str)

	sup, err := StartSupervisor(&config)
	if err != nil {
		t.Errorf("cannot start supervisor: %s", err.Error())
	}

	// test success requests
	reqs := repeatRequestData("1122334455667788112233445566778811223344556677881122334455667788", 10)
	for range []int{0, 1, 2, 3, 4, 5, 6} {
		sup.EnqueueClientRequest(&reqs)
	}

	// test error requests
	mreqs := repeatRequestData("missingtopic", 1)
	sup.EnqueueClientRequest(&mreqs)

	ureqs := repeatRequestData("unregistered", 1)
	sup.EnqueueClientRequest(&ureqs)

	breqs := repeatRequestData("baddevicetoken", 1)
	sup.EnqueueClientRequest(&breqs)

	time.Sleep(time.Second * 1)
	wg.Wait()
	sup.Shutdown()

	if *(score[apns.MissingTopic.String()]) != 1 {
		t.Errorf("Expected MissingTopic count is 1 but got %d", *(score[apns.MissingTopic.String()]))
	}
	if *(score[apns.Unregistered.String()]) != 1 {
		t.Errorf("Expected Unregistered count is 1 but got %d", *(score[apns.Unregistered.String()]))
	}
	if *(score[apns.BadDeviceToken.String()]) != 1 {
		t.Errorf("Expected BadDeviceToken count is 1 but got %d", *(score[apns.BadDeviceToken.String()]))
	}
	if *(score["success"]) != 70 {
		t.Errorf("Expected success count is 70 but got %d", *(score["success"]))
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
}
