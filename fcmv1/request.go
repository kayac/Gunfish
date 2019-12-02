package fcmv1

import (
	"firebase.google.com/go/messaging"
)

// Payload for fcm v1
type Payload struct {
	Message messaging.Message `json:"message"`
}
