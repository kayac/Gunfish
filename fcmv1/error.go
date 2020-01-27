package fcmv1

import "fmt"

type FCMErrorResponseCode int

// Error const variables
const (
	InvalidArgument = "INVALID_ARGUMENT"
	Unregistered    = "UNREGISTERED"
	NotFound        = "NOT_FOUND"
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
