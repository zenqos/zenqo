package middleware

import (
  	"net/http"
  	"sync"
  	"time"

  	zlog "github.com/zenqos/zenqo/internal/log"
  )

// rateBucket holds the request count and window reset time for a single client key.
type rateBucket struct {
  	count int
  	reset time.Time
  }

// RateLimitConfig configures the RateLimit middleware.
type RateLimitConfig struct {
  	// Requests is the maximum number of requests allowed per Window. Default: 100.
  	Requests int
  	// Window is the duration of each rate-limiting window. Default: 1 minute.
  	Window time.Duration
  	// KeyFunc extracts a unique string key from the request (e.g. client IP).
  	// Default: r.RemoteAddr
  	KeyFunc func(r *http.Request) string
  }

// DefaultRateLimitConfig returns a permissive default configuration.
func DefaultRateLimitConfig() RateLimitConfig {
  	return RateLimitConfig{
      		Requests: 100,
      		Window:   time.Minute,
      		KeyFunc:  func(r *http.Request) string { return r.RemoteAddr },
      	}
  }

// RateLimit returns a middleware that enforces a fixed-window rate limit per client.
// Clients that exceed the configured request count within a window receive
// 429 Too Many Requests. The in-memory store is safe for concurrent use.
//
// Example:
//
//	app.Use(middleware.RateLimit()) // 100 req/min per IP (default)
//
//	app.Use(middleware.RateLimit(middleware.RateLimitConfig{
//	    Requests: 20,
//	    Window:   30 * time.Second,
//	}))
func RateLimit(configs ...RateLimitConfig) func(http.Handler) http.Handler {
  	cfg := DefaultRateLimitConfig()
  	if len(configs) > 0 {
      		c := configs[0]
      		if c.Requests > 0 {
            			cfg.Requests = c.Requests
            		}
      		if c.Window > 0 {
            			cfg.Window = c.Window
            		}
      		if c.KeyFunc != nil {
            			cfg.KeyFunc = c.KeyFunc
            		}
      	}

  	var mu sync.Mutex
  	buckets := make(map[string]*rateBucket)

  	return func(next http.Handler) http.Handler {
      		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            			key := cfg.KeyFunc(r)
            			now := time.Now()

            			mu.Lock()
            			b, ok := buckets[key]
            			if !ok || now.After(b.reset) {
                    				b = &rateBucket{count: 0, reset: now.Add(cfg.Window)}
                    				buckets[key] = b
                    			}
            			b.count++
            			count := b.count
            			mu.Unlock()

            			if count > cfg.Requests {
                    				zlog.Log("RateLimit", "request limit exceeded for "+key)
                    				http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
                    				return
                    			}

            			next.ServeHTTP(w, r)
            		})
      	}
  }
