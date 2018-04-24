package apns

// ErrorResponseCode shows error message of responses from apns
type ErrorResponseCode int

// https://developer.apple.com/library/content/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/CommunicatingwithAPNs.html
// HTTP/2 Response from APNs Table 8-6
// ErrorMessage const
const (
	PayloadEmpty ErrorResponseCode = iota
	PayloadTooLarge
	BadTopic
	TopicDisallowed
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
	BadCollapseId
	BadMessageId
	ExpiredProviderToken
	InvalidProviderToken
	MissingProviderToken
	TooManyProviderTokenUpdates
)
