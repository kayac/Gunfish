package apns

import (
	"testing"
)

func TestResult(t *testing.T) {
	result := Result{
		APNsID:     "xxxx",
		StatusCode: 400,
		Token:      "foo",
		Reason:     "BadDeviceToken",
	}
	b, err := result.MarshalJSON()
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", string(b))
	if string(b) != `{"provider":"apns","apns-id":"xxxx","status":400,"token":"foo","reason":"BadDeviceToken"}` {
		t.Errorf("unexpected encoded json: %s", string(b))
	}
}
