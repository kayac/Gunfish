package fcm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// fcm Client const variables
const (
	DefaultFCMEndpoint = "https://fcm.googleapis.com/fcm/send"
	ClientTimeout      = time.Second * 10
)

// Client is FCM client
type Client struct {
	endpoint *url.URL
	apiKey   string
	Client   *http.Client
}

// Send sends notifications to fcm
func (c *Client) Send(p Payload) ([]Result, error) {
	req, err := c.NewRequest(p)
	if err != nil {
		return nil, err
	}

	res, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var body ResponseBody
	dec := json.NewDecoder(res.Body)
	if err = dec.Decode(&body); err != nil {
		return nil, NewError(res.StatusCode, err.Error())
	}

	if res.StatusCode != http.StatusOK {
		return nil, NewError(res.StatusCode, "status is not OK")
	}
	if len(p.RegistrationIDs) == 0 && len(body.Results) == 1 {
		r := body.Results[0]
		r.To = p.To
		r.StatusCode = res.StatusCode
		return []Result{r}, nil
	} else if len(p.RegistrationIDs) > 0 && len(p.RegistrationIDs) == len(body.Results) {
		ret := make([]Result, 0, len(body.Results))
		for i, r := range body.Results {
			r.RegistrationID = p.RegistrationIDs[i]
			r.StatusCode = res.StatusCode
			ret = append(ret, r)
		}
		return ret, nil
	}

	return nil, NewError(res.StatusCode, "unexpected response")
}

// NewRequest creates request for fcm
func (c *Client) NewRequest(p Payload) (*http.Request, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.endpoint.String(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("key=%s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// NewClient establishes a http connection with fcm
func NewClient(apikey string, endpoint *url.URL, timeout time.Duration) *Client {
	client := &http.Client{
		Timeout: timeout,
	}

	c := &Client{
		apiKey: apikey,
		Client: client,
	}

	if endpoint != nil {
		c.endpoint = endpoint

	} else {
		c.endpoint, _ = url.Parse(DefaultFCMEndpoint)
	}

	return c
}
