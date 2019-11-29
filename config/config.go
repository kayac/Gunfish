package config

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/kayac/Gunfish/fcmv1"
	goconf "github.com/kayac/go-config"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Limit values
const (
	MaxWorkerNum           = 119   // Maximum of worker number
	MinWorkerNum           = 1     // Minimum of worker number
	MaxQueueSize           = 40960 // Maximum queue size.
	MinQueueSize           = 128   // Minimum Queue size.
	MaxRequestSize         = 5000  // Maximum of requset count.
	MinRequestSize         = 1     // Minimum of request size.
	LimitApnsTokenByteSize = 100   // Payload byte size.
)

const (
	// Default array size of posted data. If not configures at file, this value is set.
	DefaultRequestQueueSize = 2000
	// Default port number of provider server
	DefaultPort = 8003
	// Default supervisor's queue size. If not configures at file, this value is set.
	DefaultQueueSize = 1000
)

// Config is the configure of an APNS provider server
type Config struct {
	Apns     SectionApns     `toml:"apns"`
	Provider SectionProvider `toml:"provider"`
	FCM      SectionFCM      `toml:"fcm"`
	FCMv1    SectionFCMv1    `toml:"fcm_v1"`
}

// SectionProvider is Gunfish provider configuration
type SectionProvider struct {
	WorkerNum        int `toml:"worker_num"`
	QueueSize        int `toml:"queue_size"`
	RequestQueueSize int `toml:"max_request_size"`
	Port             int `toml:"port"`
	DebugPort        int
	MaxConnections   int    `toml:"max_connections"`
	ErrorHook        string `toml:"error_hook"`
}

// SectionApns is the configure which is loaded from gunfish.toml
type SectionApns struct {
	Host                string
	CertFile            string `toml:"cert_file"`
	KeyFile             string `toml:"key_file"`
	Kid                 string `toml:"kid"`
	TeamID              string `toml:"team_id"`
	CertificateNotAfter time.Time
	Enabled             bool
}

// SectionFCM is the configuration of fcm
type SectionFCM struct {
	APIKey  string `toml:"api_key"`
	Enabled bool
}

// SectionFCMv1 is the configuration of fcm/v1
type SectionFCMv1 struct {
	GoogleApplicationCredentials string `toml:"google_application_credentials"`
	Enabled                      bool
	ProjectID                    string
	TokenSource                  oauth2.TokenSource
}

// DefaultLoadConfig loads default /etc/gunfish.toml
func DefaultLoadConfig() (Config, error) {
	return LoadConfig("/etc/gunfish/gunfish.toml")
}

// LoadConfig reads gunfish.toml and loads on ApnsConfig struct
func LoadConfig(fn string) (Config, error) {
	var config Config

	if err := goconf.LoadWithEnvTOML(&config, fn); err != nil {
		return config, err
	}

	// if not set parameters, set default value.
	if config.Provider.RequestQueueSize == 0 {
		config.Provider.RequestQueueSize = DefaultRequestQueueSize
	}

	if config.Provider.QueueSize == 0 {
		config.Provider.QueueSize = DefaultQueueSize
	}

	if config.Provider.Port == 0 {
		config.Provider.Port = DefaultPort
	}

	// validates config parameters
	if err := (&config).validateConfig(); err != nil {
		return config, errors.Wrap(err, "validate config failed")
	}

	return config, nil
}

func (c *Config) validateConfig() error {
	if err := c.validateConfigProvider(); err != nil {
		return errors.Wrap(err, "[provider]")
	}
	if (c.Apns.CertFile != "" && c.Apns.KeyFile != "") || (c.Apns.TeamID != "" && c.Apns.Kid != "") {
		c.Apns.Enabled = true
		if err := c.validateConfigAPNs(); err != nil {
			return errors.Wrap(err, "[apns]")
		}
	}
	if c.FCM.APIKey != "" {
		c.FCM.Enabled = true
		if err := c.validateConfigFCM(); err != nil {
			return errors.Wrap(err, "[fcm]")
		}
	}
	if c.FCMv1.GoogleApplicationCredentials != "" {
		c.FCMv1.Enabled = true
		if err := c.validateConfigFCMv1(); err != nil {
			return errors.Wrap(err, "[fcm_v1]")
		}
	}
	return nil
}

func (c *Config) validateConfigProvider() error {
	if c.Provider.RequestQueueSize < MinRequestSize || c.Provider.RequestQueueSize > MaxRequestSize {
		return fmt.Errorf("MaxRequestSize was out of available range: %d. (%d-%d)", c.Provider.RequestQueueSize,
			MinRequestSize, MaxRequestSize)
	}

	if c.Provider.QueueSize < MinQueueSize || c.Provider.QueueSize > MaxQueueSize {
		return fmt.Errorf("QueueSize was out of available range: %d. (%d-%d)", c.Provider.QueueSize,
			MinQueueSize, MaxQueueSize)
	}

	if c.Provider.WorkerNum < MinWorkerNum || c.Provider.WorkerNum > MaxWorkerNum {
		return fmt.Errorf("WorkerNum was out of available range: %d. (%d-%d)", c.Provider.WorkerNum,
			MinWorkerNum, MaxWorkerNum)
	}

	return nil
}

func (c *Config) validateConfigFCM() error {
	return nil
}

func (c *Config) validateConfigFCMv1() error {
	b, err := ioutil.ReadFile(c.FCMv1.GoogleApplicationCredentials)
	if err != nil {
		return err
	}
	serviceAccount := make(map[string]string)
	if err := json.Unmarshal(b, &serviceAccount); err != nil {
		return err
	}
	if projectID := serviceAccount["project_id"]; projectID != "" {
		c.FCMv1.ProjectID = projectID
	} else {
		return fmt.Errorf("invalid service account json: %s project_id is not defined", c.FCMv1.GoogleApplicationCredentials)
	}

	conf, err := google.JWTConfigFromJSON(b, fcmv1.Scope)
	if err != nil {
		return err
	}
	c.FCMv1.TokenSource = conf.TokenSource(context.Background())

	return err
}

func (c *Config) validateConfigAPNs() error {
	if c.Apns.CertFile != "" && c.Apns.KeyFile != "" {
		// check certificate files and expiration
		cert, err := tls.LoadX509KeyPair(c.Apns.CertFile, c.Apns.KeyFile)
		if err != nil {
			return fmt.Errorf("Invalid certificate pair for APNS: %s", err)
		}
		now := time.Now()
		for _, _ct := range cert.Certificate {
			ct, err := x509.ParseCertificate(_ct)
			if err != nil {
				return fmt.Errorf("Cannot parse X509 certificate")
			}
			if now.Before(ct.NotBefore) || now.After(ct.NotAfter) {
				return fmt.Errorf("Certificate is expired. Subject: %s, NotBefore: %s, NotAfter: %s", ct.Subject, ct.NotBefore, ct.NotAfter)
			}
			if c.Apns.CertificateNotAfter.IsZero() || c.Apns.CertificateNotAfter.Before(ct.NotAfter) {
				// hold minimum not after
				c.Apns.CertificateNotAfter = ct.NotAfter
			}
		}
	}
	return nil
}
