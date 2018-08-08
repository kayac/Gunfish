package apns

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/kayac/Gunfish/config"
	"golang.org/x/net/http2"
)

const (
	// HTTP2 client timeout
	HTTP2ClientTimeout = time.Second * 10
)

// Client is apns client
type Client struct {
	Host         string
	client       *http.Client
	authToken    string
	kid          string
	teamID       string
	keyFile      string
	issuedAt     time.Time
	useAuthToken bool
}

// Send sends notifications to apns
func (ac *Client) Send(n Notification) ([]Result, error) {
	req, err := ac.NewRequest(n.Token, &n.Header, n.Payload)
	if err != nil {
		return nil, err
	}

	res, err := ac.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	ret := []Result{
		Result{
			APNsID:     res.Header.Get("apns-id"),
			StatusCode: res.StatusCode,
			Token:      n.Token,
		},
	}

	if res.StatusCode != http.StatusOK {
		var er ErrorResponse
		err := json.NewDecoder(res.Body).Decode(&er)
		if err != nil {
			ret[0].Reason = err.Error()
		} else {
			ret[0].Reason = er.Reason
		}
	}

	return ret, nil
}

// NewRequest creates request for apns
func (ac *Client) NewRequest(token string, h *Header, payload Payload) (*http.Request, error) {
	u, err := url.Parse(fmt.Sprintf("%s/3/device/%s", ac.Host, token))
	if err != nil {
		return nil, err
	}

	data, err := payload.MarshalJSON()
	if err != nil {
		return nil, err
	}

	nreq, err := http.NewRequest("POST", u.String(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	if h != nil {
		if h.ApnsID != "" {
			nreq.Header.Set("apns-id", h.ApnsID)
		}
		if h.ApnsExpiration != "" {
			nreq.Header.Set("apns-expiration", h.ApnsExpiration)
		}
		if h.ApnsPriority != "" {
			nreq.Header.Set("apns-priority", h.ApnsPriority)
		}
		if h.ApnsTopic != "" {
			nreq.Header.Set("apns-topic", h.ApnsTopic)
		}
	}

	// APNs provider token authenticaton
	if ac.useAuthToken {
		// If iat of jwt is more than 1 hour ago, returns 403 InvalidProviderToken.
		// So, recreate jwt earlier than 1 hour.
		if ac.issuedAt.Add(time.Hour - time.Minute).Before(time.Now()) {
			if err := ac.issueToken(); err != nil {
				return nil, err
			}
		}
		nreq.Header.Set("Authorization", "bearer "+ac.authToken)
	}

	return nreq, err
}

func (ac *Client) issueToken() error {
	var err error
	now := time.Now()
	ac.authToken, err = CreateJWT(ac.keyFile, ac.kid, ac.teamID, now)
	if err != nil {
		return err
	}
	ac.issuedAt = now
	return nil
}

// NewConnection establishes a http2 connection
func NewConnection(certFile, keyFile string, secureSkip, useAuthToken bool) (*http.Client, error) {
	// Provider authentication token
	if useAuthToken {
		tr := &http.Transport{}
		if err := http2.ConfigureTransport(tr); err != nil {
			return nil, err
		}
		return &http.Client{
			Timeout:   HTTP2ClientTimeout,
			Transport: tr,
		}, nil
	}

	// APNs Provider Certificates
	certPEMBlock, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, err
	}

	keyPEMBlock, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)

	if err != nil {
		return nil, err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: secureSkip,
			Certificates:       []tls.Certificate{cert},
		},
	}

	if err := http2.ConfigureTransport(tr); err != nil {
		return nil, err
	}

	return &http.Client{
		Timeout:   HTTP2ClientTimeout,
		Transport: tr,
	}, nil
}

func NewClient(conf config.SectionApns) (*Client, error) {
	useAuthToken := conf.Kid != "" && conf.TeamID != ""
	c, err := NewConnection(conf.CertFile, conf.KeyFile, conf.SkipInsecure, useAuthToken)
	if err != nil {
		return nil, err
	}

	client := &Client{
		Host:         conf.Host,
		client:       c,
		kid:          conf.Kid,
		teamID:       conf.TeamID,
		keyFile:      conf.KeyFile,
		useAuthToken: useAuthToken,
	}
	if client.useAuthToken {
		if err := client.issueToken(); err != nil {
			return nil, err
		}
	}

	return client, nil
}
