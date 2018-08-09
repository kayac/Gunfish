package gunfish

import (
	"time"
)

// Default values
const (
	// SendRetryCount is the threashold which is resend count.
	SendRetryCount = 10
	// RetryWaitTime is periodical time to retrieve notifications from retry queue to resend
	RetryWaitTime = time.Millisecond * 500
	// RetryOnceCount is the number of sending notification at once.
	RetryOnceCount = 1000
	// multiplicity of sending notifications.
	SenderNum     = 20
	RequestPerSec = 2000

	// About the average time of response from apns. That value is not accurate
	// because that is defined heuristically in Japan.
	AverageResponseTime = time.Millisecond * 150
	// Minimum RetryAfter time (seconds).
	RetryAfterSecond = time.Second * 10
	// Gunfish returns RetryAfter header based on 'Exponential Backoff'. Therefore,
	// that defines the wait time threshold so as not to wait too long.
	ResetRetryAfterSecond = time.Second * 60
	// FlowRateInterval is the designed value to enable to delivery notifications
	// for that value seconds. Gunfish is designed as to ensure to delivery
	// notifications for 10 seconds.
	FlowRateInterval = time.Second * 10
	// Default flow rate as notification requests per sec (req/sec).
	DefaultFlowRatePerSec = 2000
	// Wait millisecond interval when to shutdown.
	ShutdownWaitTime = time.Millisecond * 10
	// That is the count while request counter is 0 in the 'ShutdownWaitTime' period.
	RestartWaitCount = 50
)

// Apns endpoints
const (
	DevServer  = "https://api.development.push.apple.com"
	ProdServer = "https://api.push.apple.com"
	MockServer = "https://localhost:2195"
)

// Supports Content-Type
const (
	ApplicationJSON              = "application/json"
	ApplicationXW3FormURLEncoded = "application/x-www-form-urlencoded"
)

// Environment struct
type Environment int

// Executed environment
const (
	Production Environment = iota
	Development
	Test
	Disable
)

// Alert fields mapping
var (
	AlertKeyToField = map[string]string{
		"title":          "Title",
		"body":           "Body",
		"title-loc-key":  "TitleLocKey",
		"title-loc-args": "TitleLocArgs",
		"action-loc-key": "ActionLocKey",
		"loc-key":        "LocKey",
		"loc-args":       "LocArgs",
		"launch-image":   "LaunchImage",
	}
)

var (
	OutputHookStdout bool
	OutputHookStderr bool
)
