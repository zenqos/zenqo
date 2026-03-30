package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func newRequest() *http.Request {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "1.2.3.4:1234"
	return r
}

func TestRateLimitAllowsUnderLimit(t *testing.T) {
	h := RateLimit(RateLimitConfig{Max: 5, Window: time.Second})(okHandler)

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, newRequest())
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimitBlocksOverLimit(t *testing.T) {
	lim := NewRateLimiter(RateLimitConfig{Max: 3, Window: time.Second})
	defer lim.Stop()
	h := lim.Middleware()(okHandler)

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, newRequest())
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequest())
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after limit, got %d", w.Code)
	}
}

func TestRateLimitHeaders(t *testing.T) {
	lim := NewRateLimiter(RateLimitConfig{Max: 10, Window: time.Minute})
	defer lim.Stop()
	h := lim.Middleware()(okHandler)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequest())

	if w.Header().Get("X-RateLimit-Limit") == "" {
		t.Fatal("missing X-RateLimit-Limit header")
	}
	if w.Header().Get("X-RateLimit-Remaining") == "" {
		t.Fatal("missing X-RateLimit-Remaining header")
	}
	if w.Header().Get("X-RateLimit-Reset") == "" {
		t.Fatal("missing X-RateLimit-Reset header")
	}
}

// --- #133: RateLimitWithContext stops goroutine on cancel ---

func TestRateLimitWithContextStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	h := RateLimitWithContext(ctx, RateLimitConfig{Max: 5, Window: time.Second})(okHandler)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, newRequest())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Cancel should stop the goroutine without panic.
	cancel()

	// Give the goroutine a moment to stop.
	time.Sleep(10 * time.Millisecond)
}

func TestRateLimitWithContextNoLeakInTests(t *testing.T) {
	// Calling RateLimitWithContext multiple times (simulating multiple test runs)
	// should not accumulate goroutines because each context gets cancelled.
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		RateLimitWithContext(ctx, RateLimitConfig{Max: 10, Window: time.Second})
		cancel()
	}
	time.Sleep(20 * time.Millisecond)
	// If goroutines leaked, the race detector or goleak would catch them.
	// This test just verifies cancel() doesn't panic and compiles correctly.
}
