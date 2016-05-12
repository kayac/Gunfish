package gunfish

import (
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/Sirupsen/logrus"
)

// Config is the configure of an APNS provider server
type Config struct {
	Apns     SectionApns     `toml:apns`
	Provider SectionProvider `toml:provider`
}

// SectionProvider is Gunfish provider configuration
type SectionProvider struct {
	WorkerNum        int `toml:"worker_num"`
	QueueSize        int `toml:"queue_size"`
	RequestQueueSize int `toml:"max_request_size"`
	Port             int `toml:"port"`
	DebugPort        int
	MaxConnections   int `toml:"max_connections"`
}

// SectionApns is the configure which is loaded from gunfish.toml
type SectionApns struct {
	Host          string
	SkipInsecure  bool   `toml:"skip_insecure"`
	CertFile      string `toml:"cert_file"`
	KeyFile       string `toml:"key_file"`
	SenderNum     int    `toml:"sender_num"`
	RequestPerSec int    `toml:"request_per_sec"`
	ErrorHook     string `toml:"error_hook"`
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

	if config.Apns.SenderNum == 0 {
		config.Apns.SenderNum = DefaultApnsSenderNum
	}

	if config.Provider.Port == 0 {
		config.Provider.Port = DefaultPort
	}

	// validates config parameters
	if err := config.validateConfig(); err != nil {
		LogWithFields(logrus.Fields{"type": "load_config"}).Error(err)
		return config, err
	}

	return config, nil
}

func (c Config) validateConfig() error {
	if c.Apns.CertFile == "" || c.Apns.KeyFile == "" {
		return fmt.Errorf("Not specified a cert or key file.")
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

	if c.Apns.SenderNum < MinSenderNum || c.Apns.SenderNum > MaxSenderNum {
		return fmt.Errorf("APNS SenderNum was out of available range: %d. (%d-%d)", c.Apns.SenderNum,
			MinSenderNum, MaxSenderNum)
	}

	if c.Apns.ErrorHook == "" {
		return fmt.Errorf("ErrorHook cannot be empty.")
	}

	return nil
}
