package gunfish

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/sirupsen/logrus"
)

// Config is the configure of an APNS provider server
type Config struct {
	Apns     SectionApns     `toml:"apns"`
	Provider SectionProvider `toml:"provider"`
	FCM      SectionFCM      `toml:"fcm"`
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
	SkipInsecure        bool   `toml:"skip_insecure"`
	CertFile            string `toml:"cert_file"`
	KeyFile             string `toml:"key_file"`
	CertificateNotAfter time.Time
	enabled             bool
}

// SectionFCM is the configuration of fcm
type SectionFCM struct {
	APIKey  string `toml:"api_key"`
	enabled bool
}

// DefaultLoadConfig loads default /etc/gunfish.toml
func DefaultLoadConfig() (Config, error) {
	return LoadConfig("/etc/gunfish/gunfish.toml")
}

// LoadConfig reads gunfish.toml and loads on ApnsConfig struct
func LoadConfig(fn string) (Config, error) {
	var config Config

	if _, err := toml.DecodeFile(fn, &config); err != nil {
		LogWithFields(logrus.Fields{"type": "load_config"}).Warnf("%v %s %s", config, err, fn)
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
		LogWithFields(logrus.Fields{"type": "load_config"}).Error(err)
		return config, err
	}

	return config, nil
}

func (c *Config) validateConfig() error {
	if c.Apns.CertFile != "" && c.Apns.KeyFile != "" {
		c.Apns.enabled = true
		if err := c.validateConfigApns(); err != nil {
			return err
		}
	}
	if c.FCM.APIKey != "" {
		c.FCM.enabled = true
		if err := c.validateConfigFCM(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) validateConfigFCM() error {
	return nil
}

func (c *Config) validateConfigApns() error {
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
