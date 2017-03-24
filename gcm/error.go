package gcm

type GCMErrorResponseCode int

// Error const variables
const (
	// 200 + error
	MissingRegistration GCMErrorResponseCode = iota
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
