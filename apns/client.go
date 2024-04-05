package apns

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/kayac/Gunfish/config"
	"golang.org/x/net/http2"
)

const (
	// HTTP2 client timeout
	HTTP2ClientTimeout = time.Second * 10
)

var ClientTransport = func(cert tls.Certificate) *http.Transport {
	return &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}
}

type authToken struct {
	jwt      string
	issuedAt time.Time
}

// Client is apns client
type Client struct {
	Host         string
	client       *http.Client
	authToken    authToken
	kid          string
	teamID       string
	key          []byte
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
		if h.ApnsPushType != "" {
			nreq.Header.Set("apns-push-type", h.ApnsPushType)
		}
	}

	// APNs provider token authenticaton
	if ac.useAuthToken {
		// If iat of jwt is more than 1 hour ago, returns 403 InvalidProviderToken.
		// So, recreate jwt earlier than 1 hour.
		if ac.authToken.issuedAt.Add(time.Hour - time.Minute).Before(time.Now()) {
			if err := ac.issueToken(); err != nil {
				return nil, err
			}
		}
		nreq.Header.Set("Authorization", "bearer "+ac.authToken.jwt)
	}

	return nreq, err
}

func (ac *Client) issueToken() error {
	/*
		tokenTime is a unixtime of nearest HH:00:00 or HH:30:00 from now-10 min.
		When many Gunfish processes are running in one service, these JWT tokens must be a same value at the same time.

		https://developer.apple.com/documentation/usernotifications/setting_up_a_remote_notification_server/sending_notification_requests_to_apns
		> Update the authentication token no more than once every 20 minutes.

	*/
	tokenTime := ((time.Now().Unix() - 600) / 1800) * 1800

	var err error
	ac.authToken.jwt, err = CreateJWT(ac.key, ac.kid, ac.teamID, tokenTime)
	if err != nil {
		return err
	}
	ac.authToken.issuedAt = time.Unix(tokenTime, 0)
	return nil
}

func NewClient(conf config.SectionApns) (*Client, error) {
	useAuthToken := conf.Kid != "" && conf.TeamID != ""
	tr := &http.Transport{}
	if !useAuthToken {
		certPEMBlock, err := os.ReadFile(conf.CertFile)
		if err != nil {
			return nil, err
		}

		keyPEMBlock, err := os.ReadFile(conf.KeyFile)
		if err != nil {
			return nil, err
		}

		cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
		if err != nil {
			return nil, err
		}
		tr = ClientTransport(cert)
	}

	if err := http2.ConfigureTransport(tr); err != nil {
		return nil, err
	}

	key, err := os.ReadFile(conf.KeyFile)
	if err != nil {
		return nil, err
	}

	client := &Client{
		Host: conf.Host,
		client: &http.Client{
			Timeout:   HTTP2ClientTimeout,
			Transport: tr,
		},
		kid:          conf.Kid,
		teamID:       conf.TeamID,
		key:          key,
		useAuthToken: useAuthToken,
	}
	if client.useAuthToken {
		if err := client.issueToken(); err != nil {
			return nil, err
		}
	}

	return client, nil
}
