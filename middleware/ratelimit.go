package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimitConfig defines the configuration for the rate limiting middleware.
type RateLimitConfig struct {
	Max     int                          // maximum requests per window; default: 100
	Window  time.Duration                // window duration; default: 1 minute
	KeyFunc func(r *http.Request) string // key extractor; default: client IP
	OnLimit http.HandlerFunc             // handler when limit exceeded; default: 429 JSON
}

// DefaultRateLimitConfig returns a sensible rate limit configuration.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Max:    100,
		Window: time.Minute,
		KeyFunc: func(r *http.Request) string {
			return r.RemoteAddr
		},
		OnLimit: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"code":429,"message":"rate limit exceeded"}`)) //nolint:errcheck
		},
	}
}

type visitor struct {
	count       int
	windowStart time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	max      int
	window   time.Duration
	stop     chan struct{}
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		max:      max,
		window:   window,
		stop:     make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

// Stop terminates the background cleanup goroutine.
// Call this when the middleware is no longer needed (e.g. in tests).
func (rl *rateLimiter) Stop() { close(rl.stop) }

func (rl *rateLimiter) allow(key string, now time.Time) (allowed bool, count int, resetTime time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[key]
	if !ok || now.Sub(v.windowStart) >= rl.window {
		rl.visitors[key] = &visitor{count: 1, windowStart: now}
		return true, 1, now.Add(rl.window)
	}

	v.count++
	reset := v.windowStart.Add(rl.window)
	if v.count > rl.max {
		return false, v.count, reset
	}
	return true, v.count, reset
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(2 * rl.window)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for key, v := range rl.visitors {
				if now.Sub(v.windowStart) >= rl.window {
					delete(rl.visitors, key)
				}
			}
			rl.mu.Unlock()
		case <-rl.stop:
			return
		}
	}
}

// RateLimit returns a middleware that enforces request rate limiting
// using a fixed-window counter algorithm.
//
// Call with no arguments for the default configuration (100 req/min per IP),
// or pass a RateLimitConfig to customize.
//
// Response headers set on every request:
//   - X-RateLimit-Limit: maximum requests per window
//   - X-RateLimit-Remaining: remaining requests in the current window
//   - X-RateLimit-Reset: Unix timestamp when the window resets
//
// When the limit is exceeded, the middleware also sets:
//   - Retry-After: seconds until the window resets
//
// Example:
//
//	app.Use(middleware.RateLimit())                                   // 100 req/min per IP
//	app.Use(middleware.RateLimit(middleware.RateLimitConfig{          // custom
//	    Max:    10,
//	    Window: time.Second,
//	}))
func RateLimit(configs ...RateLimitConfig) func(http.Handler) http.Handler {
	cfg := DefaultRateLimitConfig()
	if len(configs) > 0 {
		cfg = configs[0]
		if cfg.Max == 0 {
			cfg.Max = 100
		}
		if cfg.Window == 0 {
			cfg.Window = time.Minute
		}
		if cfg.KeyFunc == nil {
			cfg.KeyFunc = DefaultRateLimitConfig().KeyFunc
		}
		if cfg.OnLimit == nil {
			cfg.OnLimit = DefaultRateLimitConfig().OnLimit
		}
	}

	rl := newRateLimiter(cfg.Max, cfg.Window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := cfg.KeyFunc(r)
			allowed, count, resetTime := rl.allow(key, time.Now())

			remaining := cfg.Max - count
			if remaining < 0 {
				remaining = 0
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.Max))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

			if !allowed {
				retryAfter := int(time.Until(resetTime).Seconds())
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				cfg.OnLimit(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
