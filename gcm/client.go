package gcm

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/http2"
)

// gcm Client const variables
const (
	GCMClientTimeout = time.Second * 10
	GCMEndpoint      = "https://fcm.googleapis.com/fcm/send"
)

// Client is GCM client
type Client struct {
	apiKey string
	Client *http.Client
}

// Send sends notifications to gcm (TODO: send retry)
func (gc *Client) Send(p Payload) (*Response, error) {
	req, err := gc.NewRequest(p)
	if err != nil {
		return nil, err
	}

	res, err := gc.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resp := &Response{}
	dec := json.NewDecoder(res.Body)
	if err = dec.Decode(&resp.Body); err != nil {
		return nil, err
	}
	resp.Header.StatusCode = res.StatusCode

	if res.StatusCode != http.StatusOK {
		errCode := resp.Body.Error
		eres := ErrorResponse{
			StatusCode: resp.Header.StatusCode,
			ErrCode:    errCode,
		}
		return nil, eres
	}

	return resp, err
}

// NewRequest creates request for gcm
func (gc *Client) NewRequest(p Payload) (*http.Request, error) {
	u, err := url.Parse(GCMEndpoint)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", u.String(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("key=%s", gc.apiKey))
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// NewClient establishes a http connection with gcm
func NewClient(apikey string, insecure bool) (*Client, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
		},
	}

	if err := http2.ConfigureTransport(tr); err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   GCMClientTimeout,
	}

	return &Client{
		apiKey: apikey,
		Client: client,
	}, nil
}