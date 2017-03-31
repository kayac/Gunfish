package fcm

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestUnmarshalResponse(t *testing.T) {
	var r ResponseBody
	if err := json.Unmarshal([]byte(buildResponseBodyJSON()), &r); err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(r, buildResponseBody()) {
		t.Errorf("mismatch decoded payload:\ngot=%v\nexpected=%v", r, buildResponseBody())
	}
}

func TestMarshalResponse(t *testing.T) {
	p := buildResponseBody()
	output, err := json.Marshal(p)
	if err != nil {
		t.Error(err)
	}

	expected := `{"multicast_id":5302270091026410054,"success":3,"failure":0,"canonical_ids":0,"results":[{"provider":"fcm","status":200,"message_id":"0:1487073977923500%caa8591dcaa8591d"},{"provider":"fcm","status":200,"message_id":"0:1487073977923148%caa8591dcaa8591d"},{"provider":"fcm","status":200,"message_id":"0:1487073977924484%caa8591dcaa8591d"}]}`

	if string(output) != expected {
		t.Errorf("mismatch decoded response:\ngot=%s\nexpected=%s", output, expected)
	}
}

func buildResponseBody() ResponseBody {
	return ResponseBody{
		MulticastID:  5302270091026410054,
		Success:      3,
		Failure:      0,
		CanonicalIDs: 0,
		Results: []Result{
			Result{
				StatusCode: 200,
				MessageID:  "0:1487073977923500%caa8591dcaa8591d",
			},
			Result{
				StatusCode: 200,
				MessageID:  "0:1487073977923148%caa8591dcaa8591d",
			},
			Result{
				StatusCode: 200,
				MessageID:  "0:1487073977924484%caa8591dcaa8591d",
			},
		},
	}
}

func buildResponseBodyJSON() string {
	return `{
            "multicast_id":  5302270091026410054,
            "success":       3,
            "failure":       0,
            "canonical_ids": 0,
            "results": [
                { "status":200, "message_id": "0:1487073977923500%caa8591dcaa8591d" },
                { "status":200, "message_id": "0:1487073977923148%caa8591dcaa8591d" },
                { "status":200, "message_id": "0:1487073977924484%caa8591dcaa8591d" }
            ]
        }`
}
