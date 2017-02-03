package gunfish

import (
	"encoding/json"

	"github.com/kayac/Gunfish/apns"
)

type Request interface {
	Request() interface{}
}

// PostedData is posted data to this provider server /push/apns.
type PostedData struct {
	Header  apns.Header  `json:"header,omitempty"`
	Token   string       `json:"token"`
	Payload apns.Payload `json:"payload"`
}
