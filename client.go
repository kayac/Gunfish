package gunfish

// Client interface for gcm and apns client
type Client interface {
	Send(Notification) (*Response, error)
}
