package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	goconf "github.com/kayac/go-config"
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
}

// SectionProvider is Gunfish provider configuration
type SectionProvider struct {
	WorkerNum                  int `toml:"worker_num"`
	QueueSize                  int `toml:"queue_size"`
	RequestQueueSize           int `toml:"max_request_size"`
	Port                       int `toml:"port"`
	DebugPort                  int
	MaxConnections             int    `toml:"max_connections"`
	ErrorHook                  string `toml:"error_hook"`
	ErrorHookTo                string `toml:"error_hook_to"`
	ErrorHookCommandPersistent bool   `toml:"error_hook_command_persistent"`
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
		return config, err
	}

	return config, nil
}

func (c *Config) validateConfig() error {
	if err := c.validateConfigProvider(); err != nil {
		return err
	}
	if (c.Apns.CertFile != "" && c.Apns.KeyFile != "") || (c.Apns.TeamID != "" && c.Apns.Kid != "") {
		c.Apns.Enabled = true
		if err := c.validateConfigAPNs(); err != nil {
			return err
		}
	}
	if c.FCM.APIKey != "" {
		c.FCM.Enabled = true
		if err := c.validateConfigFCM(); err != nil {
			return err
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
