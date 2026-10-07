package gunfish

import (
	"errors"
	"testing"
	"time"

	"github.com/kayac/Gunfish/config"
	"github.com/sirupsen/logrus"
)

// Retries are enqueued by timers (time.AfterFunc), which may fire after the supervisor is shut down.
func TestRetryAfterShutdown(t *testing.T) {
	conf := config.Config{
		Provider: config.SectionProvider{
			WorkerNum:        1,
			QueueSize:        128,
			RequestQueueSize: 1000,
		},
	}
	sup, err := StartSupervisor(&conf)
	if err != nil {
		t.Fatal(err)
	}
	sup.Shutdown()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic after shutdown: %v", r)
		}
	}()
	req := Request{Notification: struct{}{}}
	// fired timer of retry ticker
	sup.resend(req, time.Second)
	// fired timer of retry for FCM QUOTA_EXCEEDED
	retry(sup.retryq, req, errors.New("QUOTA_EXCEEDED"), logrus.Fields{})
}
