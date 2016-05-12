package gunfish

// Response from apns
type Response struct {
	ApnsID     string `json:"apns-id"`
	StatusCode int    `json:"status"`
}

// ErrorResponse from apns
type ErrorResponse struct {
	Reason    string `json:"reason"`
	Timestamp int64  `json:"timestamp"`
}

func (e *ErrorResponse) Error() string {
	return e.Reason
}
