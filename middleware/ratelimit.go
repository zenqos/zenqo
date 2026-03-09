package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimitConfig defines the configuration for the rate limiting middleware.
type RateLimitConfig struct {
	Max    int           // maximum requests per window; default: 100
	Window time.Duration // window duration; default: 1 minute
	// KeyFunc extracts the rate-limit key from a request.
	// Default: client IP address (r.RemoteAddr).
	KeyFunc func(r *http.Request) string
	// OnLimit is called to write the response when a client exceeds the limit.
	// Default: 429 JSON with Retry-After header.
	OnLimit http.HandlerFunc
	// OnLimitReached is an optional observability hook called before OnLimit.
	// Use it to record metrics, emit logs, or trigger alerts.
	// It does not replace OnLimit — both are called when the limit is exceeded.
	OnLimitReached func(r *http.Request, key string, count int)
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

// visitor holds the sliding-window counters for a single rate-limit key.
type visitor struct {
	prevCount   int       // request count for the previous window
	currCount   int       // request count for the current window
	windowStart time.Time // start time of the current window
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

// allow implements a sliding window counter algorithm.
//
// Unlike a fixed-window counter, this approach prevents clients from
// exploiting window boundaries to send 2× the allowed rate. The effective
// request count is weighted by how much of the previous window overlaps
// with the current point in time.
func (rl *rateLimiter) allow(key string, now time.Time) (allowed bool, count int, resetTime time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[key]
	if !ok {
		rl.visitors[key] = &visitor{currCount: 1, windowStart: now}
		return true, 1, now.Add(rl.window)
	}

	elapsed := now.Sub(v.windowStart)
	if elapsed >= rl.window {
		// Slide the window: the old current becomes the new previous.
		v.prevCount = v.currCount
		v.currCount = 0
		v.windowStart = now
		elapsed = 0
	}

	reset := v.windowStart.Add(rl.window)

	// Weight the previous window's count by how much it still overlaps.
	ratio := 1.0 - elapsed.Seconds()/rl.window.Seconds()
	weighted := int(float64(v.prevCount)*ratio) + v.currCount

	if weighted >= rl.max {
		return false, weighted, reset
	}
	v.currCount++
	return true, weighted + 1, reset
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
				// Keep entries for at least 2 windows so prevCount remains valid.
				if now.Sub(v.windowStart) >= 2*rl.window {
					delete(rl.visitors, key)
				}
			}
			rl.mu.Unlock()
		case <-rl.stop:
			return
		}
	}
}

// RateLimiter is a rate limiter with explicit lifecycle management.
// Use NewRateLimiter when you need to call Stop() (e.g. in tests to avoid goroutine leaks).
// For simple use cases, prefer RateLimit().
//
// Example:
//
//	rl := middleware.NewRateLimiter(middleware.RateLimitConfig{Max: 10, Window: time.Second})
//	app.Use(rl.Middleware())
//	defer rl.Stop() // terminates the background cleanup goroutine
type RateLimiter struct {
	rl  *rateLimiter
	cfg RateLimitConfig
}

// NewRateLimiter creates a RateLimiter with the given configuration.
func NewRateLimiter(configs ...RateLimitConfig) *RateLimiter {
	cfg := resolveCfg(configs)
	return &RateLimiter{rl: newRateLimiter(cfg.Max, cfg.Window), cfg: cfg}
}

// Middleware returns the rate-limiting http middleware for this RateLimiter.
func (lim *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return buildMiddleware(lim.rl, lim.cfg)
}

// Stop terminates the background cleanup goroutine.
// Call this when the middleware is no longer needed (e.g. in tests).
func (lim *RateLimiter) Stop() { lim.rl.Stop() }

func resolveCfg(configs []RateLimitConfig) RateLimitConfig {
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
	return cfg
}

func buildMiddleware(rl *rateLimiter, cfg RateLimitConfig) func(http.Handler) http.Handler {
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
				if cfg.OnLimitReached != nil {
					cfg.OnLimitReached(r, key, count)
				}
				cfg.OnLimit(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit returns a middleware that enforces request rate limiting
// using a sliding window counter algorithm.
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
// Note: RateLimit starts a background cleanup goroutine that runs for the
// lifetime of the process. In tests, use NewRateLimiter and defer Stop()
// to avoid goroutine leaks.
//
// Example:
//
//	app.Use(middleware.RateLimit())                                   // 100 req/min per IP
//	app.Use(middleware.RateLimit(middleware.RateLimitConfig{          // custom
//	    Max:    10,
//	    Window: time.Second,
//	}))
func RateLimit(configs ...RateLimitConfig) func(http.Handler) http.Handler {
	cfg := resolveCfg(configs)
	rl := newRateLimiter(cfg.Max, cfg.Window)
	return buildMiddleware(rl, cfg)
}
