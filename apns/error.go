package apns

// https://developer.apple.com/library/ios/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/Chapters/APNsProviderAPI.html
//
// Reason                    | Description
// -------------------------------------------------------------------------------------------------
// PayloadEmpty              | The message payload was empty.
// --------------------------|----------------------------------------------------------------------
// PayloadTooLarge           | The message payload was too large. The maximum payload size is 4096
//                           | bytes.
// --------------------------|----------------------------------------------------------------------
// BadTopic                  | The apns-topic was invalid.
// --------------------------|----------------------------------------------------------------------
// TopicDisallowed           | Pushing to this topic is not allowed.
// --------------------------|----------------------------------------------------------------------
// BadMessageId              | The apns-id value is bad.
// --------------------------|----------------------------------------------------------------------
// BadExpirationDate         | The apns-expiration value is bad.
// --------------------------|----------------------------------------------------------------------
// BadPriority               | The apns-priority value is bad.
// --------------------------|----------------------------------------------------------------------
// MissingDeviceToken        | The device token is not specified in the request :path. Verify that
//                           | the :path header contains the device token.
// --------------------------|----------------------------------------------------------------------
// BadDeviceToken            | The specified device token was bad. Verify that the request contains
//                           | a valid token and that the token matches the environment.
// --------------------------|----------------------------------------------------------------------
// DeviceTokenNotForTopic    | The device token does not match the specified topic.
// --------------------------|----------------------------------------------------------------------
// Unregistered              | The device token is inactive for the specified topic.
// --------------------------|----------------------------------------------------------------------
// DuplicateHeaders          | One or more headers were repeated.
// --------------------------|----------------------------------------------------------------------
// BadCertificateEnvironment | The client certificate was for the wrong environment.
// --------------------------|----------------------------------------------------------------------
// BadCertificate            | The certificate was bad.
// --------------------------|----------------------------------------------------------------------
// Forbidden                 | The specified action is not allowed.
// --------------------------|----------------------------------------------------------------------
// BadPath                   | The request contained a bad :path value.
// --------------------------|----------------------------------------------------------------------
// MethodNotAllowed          | The specified :method was not POST.
// --------------------------|----------------------------------------------------------------------
// TooManyRequests           | Too many requests were made consecutively to the same device token.
// --------------------------|----------------------------------------------------------------------
// IdleTimeout               | Idle time out.
// --------------------------|----------------------------------------------------------------------
// Shutdown                  | The server is shutting down.
// --------------------------|----------------------------------------------------------------------
// InternalServerError       | An internal server error occurred.
// --------------------------|----------------------------------------------------------------------
// ServiceUnavailable        | The service is unavailable.
// --------------------------|----------------------------------------------------------------------
// MissingTopic              | The apns-topic header of the request was not specified and was
//                           | required. The apns-topic header is mandatory when the client is
//                           | connected using a certificate that supports multiple topics.
// --------------------------|----------------------------------------------------------------------

// ErrorResponseCode shows error message of responses from apns
type ErrorResponseCode int

// ErrorMessage const
const (
	PayloadEmpty ErrorResponseCode = iota
	PayloadTooLarge
	BadTopic
	TopicDisallowed
	BadMessageID
	BadExpirationDate
	BadPriority
	MissingDeviceToken
	BadDeviceToken
	DeviceTokenNotForTopic
	Unregistered
	DuplicateHeaders
	BadCertificateEnvironment
	BadCertificate
	Forbidden
	BadPath
	MethodNotAllowed
	TooManyRequests
	IdleTimeout
	Shutdown
	InternalServerError
	ServiceUnavailable
	MissingTopic
)
