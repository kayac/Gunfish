package gcm

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestUnmarshalResponse(t *testing.T) {
	var r Response
	if err := json.Unmarshal([]byte(buildResponseJSON()), &r); err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(r, buildResponse()) {
		t.Errorf("mismatch decoded payload:\ngot=%v\nexpected=%v", r, buildResponse())
	}
}

func TestMarshalResponse(t *testing.T) {
	p := buildResponse()
	output, err := json.Marshal(p)
	if err != nil {
		t.Error(err)
	}

	expected := `{"header":{"status_code":200},"body":{"multicast_id":5302270091026410054,"success":3,"failure":0,"canonical_ids":0,"results":[{"message_id":"0:1487073977923500%caa8591dcaa8591d"},{"message_id":"0:1487073977923148%caa8591dcaa8591d"},{"message_id":"0:1487073977924484%caa8591dcaa8591d"}]}}`

	if string(output) != expected {
		t.Errorf("mismatch decoded response:\ngot=%s\nexpected=%s", output, expected)
	}
}

func buildResponse() Response {
	return Response{
		ResponseHeader{
			StatusCode: 200,
		},
		ResponseBody{
			MulticastID:  5302270091026410054,
			Success:      3,
			Failure:      0,
			CanonicalIDs: 0,
			Results: []Result{
				Result{
					MessageID: "0:1487073977923500%caa8591dcaa8591d",
				},
				Result{
					MessageID: "0:1487073977923148%caa8591dcaa8591d",
				},
				Result{
					MessageID: "0:1487073977924484%caa8591dcaa8591d",
				},
			},
		},
	}
}

func buildResponseJSON() string {
	return `{
        "header": {
            "status_code": 200
        },
        "body":
        {
            "multicast_id":  5302270091026410054,
            "success":       3,
            "failure":       0,
            "canonical_ids": 0,
            "results": [
                { "message_id": "0:1487073977923500%caa8591dcaa8591d" },
                { "message_id": "0:1487073977923148%caa8591dcaa8591d" },
                { "message_id": "0:1487073977924484%caa8591dcaa8591d" }
            ]
        }
    }`
}
