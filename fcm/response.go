package fcm

import (
	"fmt"
)

// ErrorResponse is fcm error response
type ErrorResponse struct {
	ErrCodes   []string
	StatusCode int
}

func (er ErrorResponse) Error() string {
	return fmt.Sprintf("%v:[http_status:%d]", er.ErrCodes, er.StatusCode)
}

// Response is the fcm connection server response
type Response struct {
	Header ResponseHeader `json:"header"`
	Body   ResponseBody   `json:"body"`
}

// ResponseHeader fcm response header
type ResponseHeader struct {
	StatusCode int `json:"status_code"`
}

// ResponseBody fcm response body
type ResponseBody struct {
	MulticastID  int      `json:"multicast_id"`
	Success      int      `json:"success"`
	Failure      int      `json:"failure"`
	CanonicalIDs int      `json:"canonical_ids"`
	Results      []Result `json:"results,omitempty"`
	MessageID    int      `json:"message_id,omitempty"`
	Error        string   `json:"error,omitempty"`
}

// Result is the status of a processed FCMResponse
type Result struct {
	MessageID      string `json:"message_id,omitempty"`
	RegistrationID string `json:"registration_id,omitempty"`
	Error          string `json:"error,omitempty"`
}
