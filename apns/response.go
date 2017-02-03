package apns

// Response from apns
type Response struct {
	APNsID     string `json:"apns-id"`
	StatusCode int    `json:"status"`
}

func (res Response) Response() interface{} {
	return res
}

// ErrorResponse from apns
type ErrorResponse struct {
	Reason    string `json:"reason"`
	Timestamp int64  `json:"timestamp"`
}

func (e *ErrorResponse) Error() string {
	return e.Reason
}
