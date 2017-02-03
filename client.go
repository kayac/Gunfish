package gunfish

// PushClient is interface to send payload to GCM and APNs.
type PushClient interface {
	Send(Request) (Response, error)
}
