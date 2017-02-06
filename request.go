package gunfish

import (
	"github.com/kayac/Gunfish/apns"
)

type Request struct {
	Notification Notification
	Tries        int
}

type Notification interface{}

// PostedData is posted data to this provider server /push/apns.
type PostedData struct {
	Header  apns.Header  `json:"header,omitempty"`
	Token   string       `json:"token"`
	Payload apns.Payload `json:"payload"`
}
