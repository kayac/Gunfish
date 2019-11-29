package fcmv1

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestUnmarshalResponse(t *testing.T) {
	var r ResponseBody
	if err := json.Unmarshal([]byte(buildResponseBodyJSON()), &r); err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(r, buildResponseBody()); diff != "" {
		t.Errorf("mismatch decoded payload diff: %s", diff)
	}
}

func TestMarshalResponse(t *testing.T) {
	p := buildResponseBody()
	output, err := json.Marshal(p)
	if err != nil {
		t.Error(err)
	}

	expected := `{"error":{"status":"INVALID_ARGUMENT","message":"The registration token is not a valid FCM registration token","details":[{"@type":"type.googleapis.com/google.firebase.fcm.v1.FcmError","errorCode":"INVALID_ARGUMENT"},{"@type":"type.googleapis.com/google.rpc.BadRequest"}]}}`

	if string(output) != expected {
		t.Errorf("mismatch decoded response:\ngot=%s\nexpected=%s", output, expected)
	}
}

func buildResponseBody() ResponseBody {
	return ResponseBody{
		Error: &FCMError{
			Message: "The registration token is not a valid FCM registration token",
			Status:  InvalidArgument,
			Details: []Detail{
				Detail{
					Type:      "type.googleapis.com/google.firebase.fcm.v1.FcmError",
					ErrorCode: InvalidArgument,
				},
				Detail{
					Type: "type.googleapis.com/google.rpc.BadRequest",
				},
			},
		},
	}
}

func buildResponseBodyJSON() string {
	return `{
  "error": {
    "code": 400,
    "message": "The registration token is not a valid FCM registration token",
    "status": "INVALID_ARGUMENT",
    "details": [
      {
        "@type": "type.googleapis.com/google.firebase.fcm.v1.FcmError",
        "errorCode": "INVALID_ARGUMENT"
      },
      {
        "@type": "type.googleapis.com/google.rpc.BadRequest",
        "fieldViolations": [
          {
            "field": "message.token",
            "description": "The registration token is not a valid FCM registration token"
          }
        ]
      }
    ]
  }
}`
}

func TestResult(t *testing.T) {
	result := Result{
		StatusCode: 400,
		Token:      "testToken",
		Error: &FCMError{
			Status:  InvalidArgument,
			Message: "The registration token is not a valid FCM registration token",
		},
	}
	b, err := result.MarshalJSON()
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", string(b))
	if string(b) != `{"provider":"fcmv1","status":400,"token":"testToken","error":{"status":"INVALID_ARGUMENT","message":"The registration token is not a valid FCM registration token"}}` {
		t.Errorf("unexpected encoded json: %s", string(b))
	}
}
