package gunfish

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func resetRetryAfterStats() {
	atomic.StoreInt64(&(srvStats.ServiceUnavailableAt), 0)
	atomic.StoreInt64(&(srvStats.RetryAfter), 10)
}

func TestUpdateServiceUnavailable(t *testing.T) {
	resetRetryAfterStats()
	t.Cleanup(resetRetryAfterStats)

	base := int64(1700000000)
	steps := []struct {
		now  int64
		want int64
	}{
		{base, 10},      // first time: reset to RetryAfterSecond
		{base, 14},      // elapsed 0s: +4
		{base, 18},      // elapsed 0s: +4
		{base + 1, 20},  // elapsed 1s: +2
		{base + 3, 22},  // elapsed 2s: +2
		{base + 8, 22},  // elapsed 5s: +0
		{base + 20, 22}, // elapsed 12s: +0
		{base + 81, 10}, // elapsed 61s: reset to RetryAfterSecond
	}
	for i, s := range steps {
		if got := updateServiceUnavailable(s.now); got != s.want {
			t.Errorf("step %d: Retry-After got %d want %d", i, got, s.want)
		}
	}

	// Retry-After is capped by ResetRetryAfterSecond.
	for range 20 {
		updateServiceUnavailable(base + 81)
	}
	if got := updateServiceUnavailable(base + 81); got != 60 {
		t.Errorf("Retry-After must be capped: got %d want 60", got)
	}
}

func TestSetRetryAfter(t *testing.T) {
	resetRetryAfterStats()
	t.Cleanup(resetRetryAfterStats)

	w := httptest.NewRecorder()
	setRetryAfter(w, httptest.NewRequest(http.MethodPost, "/push/apns", nil), "queue is full")
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status got %d want %d", w.Code, http.StatusServiceUnavailable)
	}
	if got := w.Header().Get("Retry-After"); got != "10" {
		t.Errorf("Retry-After got %s want 10", got)
	}
	if got := w.Body.String(); got != `{"reason":"queue is full"}` {
		t.Errorf("body got %s", got)
	}
}
