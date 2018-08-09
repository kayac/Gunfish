package gunfish

import (
	"os"
	"time"

	"github.com/kayac/Gunfish/config"
)

// Stats stores metrics
type Stats struct {
	Pid                    int       `json:"pid"`
	DebugPort              int       `json:"debug_port"`
	Uptime                 int64     `json:"uptime"`
	StartAt                int64     `json:"start_at"`
	ServiceUnavailableAt   int64     `json:"su_at"`
	Period                 int64     `json:"period"`
	RetryAfter             int64     `json:"retry_after"`
	Workers                int64     `json:"workers"`
	QueueSize              int64     `json:"queue_size"`
	RetryQueueSize         int64     `json:"retry_queue_size"`
	WorkersQueueSize       int64     `json:"workers_queue_size"`
	CommandQueueSize       int64     `json:"cmdq_queue_size"`
	RetryCount             int64     `json:"retry_count"`
	RequestCount           int64     `json:"req_count"`
	SentCount              int64     `json:"sent_count"`
	ErrCount               int64     `json:"err_count"`
	CertificateNotAfter    time.Time `json:"certificate_not_after"`
	CertificateExpireUntil int64     `json:"certificate_expire_until"`
}

// NewStats initialize Stats
func NewStats(conf config.Config) Stats {
	return Stats{
		Pid:                 os.Getpid(),
		StartAt:             time.Now().Unix(),
		RetryAfter:          int64(RetryAfterSecond / time.Second),
		CertificateNotAfter: conf.Apns.CertificateNotAfter,
	}
}

// GetStats returns MemdStats of app
func (st *Stats) GetStats() *Stats {
	preUptime := st.Uptime
	st.Uptime = time.Now().Unix() - st.StartAt
	st.Period = st.Uptime - preUptime
	st.CertificateExpireUntil = int64(st.CertificateNotAfter.Sub(time.Now()).Seconds())
	return st
}
