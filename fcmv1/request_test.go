package fcmv1

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"firebase.google.com/go/messaging"
)

func TestUnmarshalPayload(t *testing.T) {
	var p Payload
	if err := json.Unmarshal([]byte(buildPayloadJSON()), &p); err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(p, buildPayload()); diff != "" {
		t.Errorf("mismatch decoded payload: diff: %s", diff)
	}
}

func TestMarshalPayload(t *testing.T) {
	p := buildPayload()
	output, err := json.Marshal(p)
	if err != nil {
		t.Error(err)
	}

	expected := `{"message":{"data":{"message":"sample message","sample_key":"sample key"},"notification":{"title":"message_title","body":"message_body","image":"https://example.com/notification.png"},"token":"testToken"}}`

	if string(output) != expected {
		t.Errorf("should be expected json: got=%s, expected=%s", output, expected)
	}
}

func buildPayload() Payload {
	dataMap := map[string]string{
		"sample_key": "sample key",
		"message":    "sample message",
	}

	return Payload{
		Message: messaging.Message{
			Notification: &messaging.Notification{
				Title:    "message_title",
				Body:     "message_body",
				ImageURL: "https://example.com/notification.png",
			},
			Data:  dataMap,
			Token: "testToken",
		},
	}
}

func buildPayloadJSON() string {
	return `{
  "message": {
    "notification": {
      "title": "message_title",
      "body": "message_body",
      "image": "https://example.com/notification.png"
    },
    "data": {
      "sample_key": "sample key",
      "message": "sample message"
    },
    "token": "testToken"
  }
}`
}
