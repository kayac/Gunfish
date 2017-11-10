package fcm

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestUnmarshalPayload(t *testing.T) {
	var p Payload
	if err := json.Unmarshal([]byte(buildPayloadJSON()), &p); err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(p, buildPayload()) {
		t.Errorf("mismatch decoded payload: got=%v, expected=%v", p, buildPayload())
	}
}

func TestMarshalPayload(t *testing.T) {
	p := buildPayload()
	output, err := json.Marshal(p)
	if err != nil {
		t.Error(err)
	}

	expected := `{"registration_ids":["registration_id_1","registration_id_2","registration_id_3"],"data":{"message":"sample message","sample_key":"sample key"},"notification":{"title":"message_title","body":"message_body"}}`

	if string(output) != expected {
		t.Errorf("should be expected json: got=%s, expected=%s", output, expected)
	}
}

func buildPayload() Payload {
	dataMap := &Data{
		"sample_key": "sample key",
		"message":    "sample message",
	}

	return Payload{
		Notification: &Notification{
			Title: "message_title",
			Body:  "message_body",
		},
		Data: dataMap,
		RegistrationIDs: []string{
			"registration_id_1",
			"registration_id_2",
			"registration_id_3",
		},
	}
}

func buildPayloadJSON() string {
	return `{
        "notification": {
            "title": "message_title",
            "body": "message_body"
        },
        "data": {
            "sample_key": "sample key",
            "message":    "sample message"
        },
        "registration_ids": [
            "registration_id_1",
            "registration_id_2",
            "registration_id_3"
        ]
    }`
}
