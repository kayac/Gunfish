package gunfish

import (
	"github.com/kayac/Gunfish/apns"
)

type Request interface {
	Request() interface{}
	RetryCount() int
}

// PostedData is posted data to this provider server /push/apns.
type PostedData struct {
	Header  apns.Header  `json:"header,omitempty"`
	Token   string       `json:"token"`
	Payload apns.Payload `json:"payload"`
}
