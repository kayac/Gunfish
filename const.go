package gunfish

import (
	"time"
)

// Version
const (
	Version = "v0.1.0"
)

// Limit values
const (
	MaxWorkerNum           = 119   // Maximum of worker number
	MinWorkerNum           = 1     // Minimum of worker number
	MaxSenderNum           = 150   // Maximum of sender number
	MinSenderNum           = 1     // Minimum of sender number
	MaxQueueSize           = 40960 // Maximum queue size.
	MinQueueSize           = 128   // Minimum Queue size.
	MaxRequestSize         = 5000  // Maximum of requset count.
	MinRequestSize         = 1     // Minimum of request size.
	LimitApnsTokenByteSize = 100   // Payload byte size.
)

// Default values
const (
	// HTTP2 client timeout
	HTTP2ClientTimeout = time.Second * 10
	// SendRetryCount is the threashold which is resend count.
	SendRetryCount = 10
	// RetryWaitTime is periodical time to retrieve notifications from retry queue to resend
	RetryWaitTime = time.Millisecond * 5000
	// RetryOnceCount is the number of sending notification at once.
	RetryOnceCount = 1000
	// Default multiplicity of sending notifications to apns. If not configures
	// at file, this value is set.
	DefaultApnsSenderNum = 20
	// Default array size of posted data. If not configures at file, this value is set.
	DefaultRequestQueueSize = 2000
	// Default port number of provider server
	DefaultPort = 38003
	// Default supervisor's queue size. If not configures at file, this value is set.
	DefaultQueueSize = 1000
	// About the average time of response from apns. That value is not accurate
	// because that is defined heuristically in Japan.
	AverageResponseTime = time.Millisecond * 150
	// Minimum RetryAfter time (seconds).
	RetryAfterSecond = time.Second * 10
	// Retry wait time incremental rate for exponential backoff
	RetryWaitIncrRate = 1.1
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
