package fcm

import "fmt"

type FCMErrorResponseCode int

// Error const variables
const (
	// 200 + error
	MissingRegistration FCMErrorResponseCode = iota
	InvalidRegistration
	NotRegistered
	InvalidPackageName
	MismatchSenderId
	MessageTooBig
	InvalidDataKey
	InvalidTtl
	DeviceMessageRateExceeded
	TopicsMessageRateExceeded
	InvalidApnsCredentials

	// NOTES: 40x response has no error message
	AuthenticationError // 401
	InvalidJSON         // 400

	// 5xx or 200
	Unavailable

	// 500 or 200
	InternalServerError

	// UnknownError
	UnknownError
)

type Error struct {
	StatusCode int
	Reason     string
}

func (e Error) Error() string {
	return fmt.Sprintf("status:%d reason:%s", e.StatusCode, e.Reason)
}

func NewError(s int, r string) Error {
	return Error{
		StatusCode: s,
		Reason:     r,
	}
}
