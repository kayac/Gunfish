package apns

import (
	"encoding/json"
)

// Request for a http2 client
type Request struct {
	Header  Header  `json:"header,omitempty"`
	Token   string  `json:"token"`
	Payload Payload `json:"payload"`
	Tries   int     `json:"tries,int"`
}

func (r Request) Request() interface{} {
	return r
}

func (r Request) RetryCount() int {
	return r.Tries
}

// Header for apns request
type Header struct {
	ApnsID         string `json:"apns-id,omitempty"`
	ApnsExpiration string `json:"apns-expiration,omitempty"`
	ApnsPriority   string `json:"apns-priority,omitempty"`
	ApnsTopic      string `json:"apns-topic,omitempty"`
}

// Payload is Notification Payload
type Payload struct {
	*APS     `json:"aps"`
	Optional map[string]interface{}
}

// APS is a part of Payload
type APS struct {
	Alert            interface{} `json:"alert,omitempty"`
	Badge            int         `json:"badge,omitempty"`
	Sound            string      `json:"sound,omitempty"`
	ContentAvailable int         `json:"content-available,omitempty"`
	Category         string      `json:"category,omitempty"`
}

// Alert is a part of APS
type Alert struct {
	Title        string   `json:"title,omitempty"`
	Body         string   `json:"body,omitempty"`
	TitleLocKey  string   `json:"title-loc-key,omitempty"`
	TitleLocArgs []string `json:"title-loc-args,omitempty"`
	ActionLocKey string   `json:"action-loc-key,omitempty"`
	LocKey       string   `json:"loc-key,omitempty"`
	LocArgs      []string `json:"loc-args,omitempty"`
	LaunchImage  string   `json:"launch-image,omitempty"`
}

// MarshalJSON for Payload struct.
func (p Payload) MarshalJSON() ([]byte, error) {
	payloadMap := make(map[string]interface{})

	payloadMap["aps"] = p.APS
	for k, v := range p.Optional {
		payloadMap[k] = v
	}

	return json.Marshal(payloadMap)
}

// UnmarshalJSON for Payload struct.
func (p *Payload) UnmarshalJSON(data []byte) error {
	var payloadMap map[string]interface{}
	p.APS = &APS{}
	p.Optional = make(map[string]interface{})

	if err := json.Unmarshal(data, &payloadMap); err != nil {
		return err
	}

	apsMap := payloadMap["aps"].(map[string]interface{})

	for k, v := range apsMap {
		switch k {
		case "alert":
			p.APS.Alert = v
		case "badge":
			p.APS.Badge = int(v.(float64))
		case "sound":
			p.APS.Sound = v.(string)
		case "category":
			p.APS.Category = v.(string)
		case "content-available":
			p.APS.ContentAvailable = int(v.(float64))
		}
	}

	for k, v := range payloadMap {
		if k != "aps" {
			p.Optional[k] = v
		}
	}

	return nil
}
