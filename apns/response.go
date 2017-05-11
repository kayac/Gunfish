package apns

import (
	"encoding/json"
	"errors"
)

const Provider = "apns"

// Response from apns
type Result struct {
	APNsID     string `json:"apns-id"`
	StatusCode int    `json:"status"`
	Token      string `json:"token"`
	Reason     string `json:"reason"`
}

func (r Result) Err() error {
	if r.StatusCode == 200 && r.Reason == "" {
		return nil
	}
	return errors.New(r.Reason)
}

func (r Result) RecipientIdentifier() string {
	return r.Token
}

func (r Result) ExtraKeys() []string {
	return []string{"apns-id", "reason"}
}

func (r Result) ExtraValue(key string) string {
	switch key {
	case "apns-id":
		return r.APNsID
	case "reason":
		return r.Reason
	}
	return ""
}

func (r Result) Status() int {
	return r.StatusCode
}

func (r Result) Provider() string {
	return Provider
}

func (r Result) MarshalJSON() ([]byte, error) {
	type Alias Result
	return json.Marshal(struct {
		Provider string `json:"provider"`
		Alias
	}{
		Provider: Provider,
		Alias:    (Alias)(r),
	})
}

type ErrorResponse struct {
	Reason    string `json:"reason"`
	Timestamp int64  `json:"timestamp"`
}
