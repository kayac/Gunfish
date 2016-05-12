package gunfish

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/Sirupsen/logrus"
	"golang.org/x/net/http2"
)

// ApnsClient is Client
type ApnsClient struct {
	host   string
	client *http.Client
}

// NewApnsClient returns ApnsClient
func NewApnsClient(host string, c *http.Client) ApnsClient {
	return ApnsClient{
		host:   host,
		client: c,
	}
}

// SendToApns sends notifications to apns
func (ac *ApnsClient) SendToApns(req Request) (*Response, error) {
	nreq, err := ac.NewRequest(req.Token, &req.Header, req.Payload)
	if err != nil {
		return nil, err
	}

	res, err := ac.client.Do(nreq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	ret := &Response{
		ApnsID:     res.Header.Get("apns-id"),
		StatusCode: res.StatusCode,
	}

	if res.StatusCode != http.StatusOK {
		var er ErrorResponse
		body, err := ioutil.ReadAll(res.Body)
		// ioutil Error
		if err != nil {
			LogWithFields(logrus.Fields{
				"type": "http/2-client",
			}).Error(err)
			return ret, err
		}
		// Unmarshal Error
		if err := json.Unmarshal(body, &er); err != nil {
			LogWithFields(logrus.Fields{
				"type": "http/2-client",
				"body": string(body),
			}).Error(err)
			return ret, err
		}
		LogWithFields(logrus.Fields{
			"type": "http/2-client",
			"body": string(body),
		}).Warnf("Catch error response.")

		return ret, &er
	}

	return ret, nil
}

// NewRequest creates request for apns
func (ac *ApnsClient) NewRequest(token string, h *Header, payload Payload) (*http.Request, error) {
	u, err := url.Parse(fmt.Sprintf("%s/3/device/%s", ac.host, token))
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

	return nreq, err
}

// NewConnection establishes a http2 connection
func NewConnection(certFile, keyFile string, secuskip bool) (*http.Client, error) {
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
			InsecureSkipVerify: secuskip,
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
