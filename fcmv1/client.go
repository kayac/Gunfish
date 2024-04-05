package fcmv1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"time"

	"golang.org/x/oauth2"
)

// fcm v1 Client const variables
const (
	DefaultFCMEndpoint = "https://fcm.googleapis.com/v1/projects"
	Scope              = "https://www.googleapis.com/auth/firebase.messaging"
	ClientTimeout      = time.Second * 10
)

// Client is FCM v1 client
type Client struct {
	endpoint    *url.URL
	Client      *http.Client
	tokenSource oauth2.TokenSource
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

	if body.Error == nil && body.Name != "" {
		return []Result{
			Result{
				StatusCode: res.StatusCode,
				Token:      p.Message.Token,
			},
		}, nil
	} else if body.Error != nil {
		return []Result{
			Result{
				StatusCode: res.StatusCode,
				Token:      p.Message.Token,
				Error:      body.Error,
			},
		}, nil
	}

	return nil, NewError(res.StatusCode, "unexpected response")
}

// NewRequest creates request for fcm
func (c *Client) NewRequest(p Payload) (*http.Request, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	token, err := c.tokenSource.Token()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", c.endpoint.String(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// NewClient establishes a http connection with fcm v1
func NewClient(tokenSource oauth2.TokenSource, projectID string, endpoint string, timeout time.Duration) (*Client, error) {
	client := &http.Client{
		Timeout: timeout,
	}
	c := &Client{
		Client:      client,
		tokenSource: tokenSource,
	}

	if endpoint == "" {
		endpoint = DefaultFCMEndpoint
	}
	ep, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	ep.Path = path.Join(ep.Path, projectID, "messages:send")
	c.endpoint = ep

	return c, nil
}
