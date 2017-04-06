package gunfish

// Client interface for fcm and apns client
type Client interface {
	Send(Notification) ([]Result, error)
}
