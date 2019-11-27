package fcmv1

import (
	"encoding/json"
	"errors"
)

const Provider = "fcmv1"

// ResponseBody fcm response body
type ResponseBody struct {
	Name  string    `json:"name"`
	Error *FCMError `json:"error"`
}

type FCMError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Details []struct {
		Type      string `json:"@type"`
		ErrorCode string `json:"errorCode"`
	}
}

// Result is the status of a processed FCMResponse
type Result struct {
	StatusCode int       `json:"status,omitempty"`
	Token      string    `json:"token,omitempty"`
	Error      *FCMError `json:"error,omitempty"`
}

func (r Result) Err() error {
	if r.Error != nil {
		return errors.New(r.Error.Status)
	}
	return nil
}

func (r Result) Status() int {
	return r.StatusCode
}

func (r Result) RecipientIdentifier() string {
	return r.Token
}

func (r Result) ExtraKeys() []string {
	return []string{"message"}
}

func (r Result) Provider() string {
	return Provider
}

func (r Result) ExtraValue(key string) string {
	switch key {
	case "message":
		if r.Error != nil {
			return r.Error.Message
		}
	}
	return ""
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
