package gunfish

import (
	"os"
	"sync/atomic"
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

// GetStats returns a snapshot of MemdStats of app.
// Counters are updated by other goroutines, so they are read atomically.
func (st *Stats) GetStats() *Stats {
	uptime := time.Now().Unix() - st.StartAt
	preUptime := atomic.SwapInt64(&st.Uptime, uptime)
	period := uptime - preUptime
	atomic.StoreInt64(&st.Period, period)

	s := &Stats{
		Pid:                  st.Pid,
		DebugPort:            st.DebugPort,
		Uptime:               uptime,
		StartAt:              st.StartAt,
		ServiceUnavailableAt: atomic.LoadInt64(&st.ServiceUnavailableAt),
		Period:               period,
		RetryAfter:           atomic.LoadInt64(&st.RetryAfter),
		Workers:              atomic.LoadInt64(&st.Workers),
		QueueSize:            atomic.LoadInt64(&st.QueueSize),
		RetryQueueSize:       atomic.LoadInt64(&st.RetryQueueSize),
		WorkersQueueSize:     atomic.LoadInt64(&st.WorkersQueueSize),
		CommandQueueSize:     atomic.LoadInt64(&st.CommandQueueSize),
		RetryCount:           atomic.LoadInt64(&st.RetryCount),
		RequestCount:         atomic.LoadInt64(&st.RequestCount),
		SentCount:            atomic.LoadInt64(&st.SentCount),
		ErrCount:             atomic.LoadInt64(&st.ErrCount),
		CertificateNotAfter:  st.CertificateNotAfter,
	}
	if !st.CertificateNotAfter.IsZero() {
		s.CertificateExpireUntil = int64(time.Until(st.CertificateNotAfter).Seconds())
	}
	return s
}
