package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig defines the allowed origins, methods, headers, and cache duration
// for Cross-Origin Resource Sharing.
type CORSConfig struct {
	AllowOrigins     []string // default: ["*"]
	AllowMethods     []string // default: GET, POST, PUT, PATCH, DELETE, OPTIONS
	AllowHeaders     []string // default: Origin, Content-Type, Accept, Authorization
	AllowCredentials bool     // default: false; set true to allow cookies/auth headers
	MaxAge           int      // preflight cache duration in seconds; default: 86400 (24h)
}

// DefaultCORSConfig returns a permissive CORS configuration suitable for development.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		MaxAge:       86400,
	}
}

// CORS returns a middleware that enables Cross-Origin Resource Sharing.
// Call with no arguments for a permissive default, or pass a CORSConfig.
//
// Example:
//
//	app.Use(middleware.CORS())                              // allow everything
//	app.Use(middleware.CORS(middleware.CORSConfig{          // production
//	    AllowOrigins: []string{"https://myapp.com"},
//	}))
func CORS(configs ...CORSConfig) func(http.Handler) http.Handler {
	cfg := DefaultCORSConfig()
	if len(configs) > 0 {
		cfg = configs[0]
	}

	// Guard against the invalid combination of wildcard origin + credentials.
	// The CORS spec (RFC 9110 / Fetch) forbids this: browsers reject responses
	// that set both Access-Control-Allow-Origin: * and
	// Access-Control-Allow-Credentials: true, making credentials inaccessible.
	// Catching it at startup prevents silent misconfiguration.
	allowAll := len(cfg.AllowOrigins) == 1 && cfg.AllowOrigins[0] == "*"
	if allowAll && cfg.AllowCredentials {
		panic("cors: AllowCredentials cannot be true when AllowOrigins is [\"*\"]. " +
			"Specify explicit origins instead (e.g. []string{\"https://myapp.com\"}).")
	}

	methods := strings.Join(cfg.AllowMethods, ", ")
	headers := strings.Join(cfg.AllowHeaders, ", ")
	maxAge := strconv.Itoa(cfg.MaxAge)

	originSet := make(map[string]bool, len(cfg.AllowOrigins))
	for _, o := range cfg.AllowOrigins {
		originSet[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Always declare Vary: Origin when the response can differ by origin.
			// Without this, CDNs and reverse proxies may cache a response intended
			// for one origin and serve it to another (cache poisoning).
			if origin != "" {
				w.Header().Add("Vary", "Origin")
			}

			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if originSet[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Preflight
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", methods)
				w.Header().Set("Access-Control-Allow-Headers", headers)
				w.Header().Set("Access-Control-Max-Age", maxAge)
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
