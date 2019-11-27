package fcmv1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// fcm v1 Client const variables
const (
	DefaultFCMEndpointFmt = "https://fcm.googleapis.com/v1/projects/%s/messages:send"
	Scope                 = "https://www.googleapis.com/auth/firebase.messaging"
	ClientTimeout         = time.Second * 10
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
func NewClient(serviceAccountFilepath string, endpoint *url.URL, timeout time.Duration) (*Client, error) {
	b, err := ioutil.ReadFile(serviceAccountFilepath)
	if err != nil {
		return nil, err
	}
	serviceAccount := make(map[string]string)
	if err := json.Unmarshal(b, &serviceAccount); err != nil {
		return nil, err
	}
	projectID := serviceAccount["project_id"]
	if projectID == "" {
		return nil, fmt.Errorf("invalid service account json: %s project_id is not defined", serviceAccountFilepath)
	}

	conf, err := google.JWTConfigFromJSON(b, Scope)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: timeout,
	}

	c := &Client{
		Client:      client,
		tokenSource: conf.TokenSource(context.Background()),
	}

	if endpoint != nil {
		c.endpoint = endpoint
	} else {
		if ep, err := url.Parse(fmt.Sprintf(DefaultFCMEndpointFmt, projectID)); err != nil {
			return nil, err
		} else {
			c.endpoint = ep
		}
	}

	return c, nil
}
