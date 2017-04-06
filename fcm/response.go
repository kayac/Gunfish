package fcm

import (
	"encoding/json"
	"errors"
)

const Provider = "fcm"

// ResponseBody fcm response body
type ResponseBody struct {
	MulticastID  int      `json:"multicast_id"`
	Success      int      `json:"success"`
	Failure      int      `json:"failure"`
	CanonicalIDs int      `json:"canonical_ids"`
	Results      []Result `json:"results,omitempty"`
	MessageID    int      `json:"message_id,omitempty"`
}

// Result is the status of a processed FCMResponse
type Result struct {
	StatusCode     int    `json:"status",omitempty`
	MessageID      string `json:"message_id,omitempty"`
	To             string `json:"to,omitempty"`
	RegistrationID string `json:"registration_id,omitempty"`
	Error          string `json:"error,omitempty"`
}

func (r Result) Err() error {
	if r.Error != "" {
		return errors.New(r.Error)
	}
	return nil
}

func (r Result) Status() int {
	return r.StatusCode
}

func (r Result) RecipientIdentifier() string {
	if r.To != "" {
		return r.To
	}
	return r.RegistrationID
}

func (r Result) ExtraKeys() []string {
	return []string{"message_id", "error"}
}

func (r Result) Provider() string {
	return Provider
}

func (r Result) ExtraValue(key string) string {
	switch key {
	case "message_id":
		return r.MessageID
	case "error":
		return r.Error
	}
	return ""
}

func (r *Result) MarshalJSON() ([]byte, error) {
	type Alias Result
	return json.Marshal(&struct {
		Provider string `json:"provider"`
		*Alias
	}{
		Provider: Provider,
		Alias:    (*Alias)(r),
	})
}
