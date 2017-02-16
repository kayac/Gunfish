package gcm

import (
	"fmt"
)

// ErrorResponse is gcm error response
type ErrorResponse struct {
	ErrCodes   []string
	StatusCode int
}

func (er ErrorResponse) Error() string {
	return fmt.Sprintf("%v:[http_status:%d]", er.ErrCodes, er.StatusCode)
}

// Response is the gcm connection server response
type Response struct {
	Header ResponseHeader `json:"header"`
	Body   ResponseBody   `json:"body"`
}

// ResponseHeader gcm response header
type ResponseHeader struct {
	StatusCode int `json:"status_code"`
}

// ResponseBody gcm response body
type ResponseBody struct {
	MulticastID  int      `json:"multicast_id"`
	Success      int      `json:"success"`
	Failure      int      `json:"failure"`
	CanonicalIDs int      `json:"canonical_ids"`
	Results      []Result `json:"results,omitempty"`
	MessageID    int      `json:"message_id,omitempty"`
	Error        string   `json:"error,omitempty"`
}

// Result is the status of a processed GCMResponse
type Result struct {
	MessageID      string `json:"message_id,omitempty"`
	RegistrationID string `json:"registration_id,omitempty"`
	Error          string `json:"error,omitempty"`
}
