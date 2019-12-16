package fcmv1

import (
	"firebase.google.com/go/messaging"
)

// Payload for fcm v1
type Payload struct {
	Message messaging.Message `json:"message"`
}

// MaxBulkRequests represens max count of request payloads in a request body.
const MaxBulkRequests = 500
