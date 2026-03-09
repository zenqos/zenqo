package middleware

import (
	"fmt"
	"net/http"
)

// SecureHeadersConfig holds options for the SecureHeadersWithConfig middleware.
type SecureHeadersConfig struct {
	// HSTSMaxAge sets the max-age for Strict-Transport-Security in seconds.
	// Set to 0 to omit the header (e.g. for plain-HTTP development servers).
	// Recommended value for production: 31536000 (1 year).
	HSTSMaxAge int
	// HSTSIncludeSubdomains appends "; includeSubDomains" to the HSTS header.
	HSTSIncludeSubdomains bool
	// HSTSPreload appends "; preload" to the HSTS header.
	// Only set this if you intend to submit your domain to the HSTS preload list.
	HSTSPreload bool
	// ContentSecurityPolicy sets the Content-Security-Policy header value.
	// An empty string omits the header.
	ContentSecurityPolicy string
	// ReferrerPolicy sets the Referrer-Policy header value.
	// Default: "strict-origin-when-cross-origin".
	ReferrerPolicy string
	// PermissionsPolicy sets the Permissions-Policy header value.
	// An empty string omits the header.
	PermissionsPolicy string
}

// DefaultSecureHeadersConfig returns a production-ready secure headers configuration.
func DefaultSecureHeadersConfig() SecureHeadersConfig {
	return SecureHeadersConfig{
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubdomains: true,
		ReferrerPolicy:        "strict-origin-when-cross-origin",
	}
}

// SecureHeaders sets baseline security response headers on every response:
//   - X-Content-Type-Options: nosniff
//   - X-Frame-Options: DENY
//   - Strict-Transport-Security: max-age=31536000; includeSubDomains
//   - Referrer-Policy: strict-origin-when-cross-origin
//
// For full customization use SecureHeadersWithConfig.
func SecureHeaders() func(http.Handler) http.Handler {
	return SecureHeadersWithConfig(DefaultSecureHeadersConfig())
}

// SecureHeadersWithConfig returns a middleware that sets HTTP security headers
// according to the provided configuration.
//
// Example:
//
//	app.Use(middleware.SecureHeadersWithConfig(middleware.SecureHeadersConfig{
//	    HSTSMaxAge:            31536000,
//	    ContentSecurityPolicy: "default-src 'self'",
//	    PermissionsPolicy:     "geolocation=(), microphone=()",
//	}))
func SecureHeadersWithConfig(cfg SecureHeadersConfig) func(http.Handler) http.Handler {
	// Pre-compute the HSTS header value once.
	var hstsValue string
	if cfg.HSTSMaxAge > 0 {
		hstsValue = fmt.Sprintf("max-age=%d", cfg.HSTSMaxAge)
		if cfg.HSTSIncludeSubdomains {
			hstsValue += "; includeSubDomains"
		}
		if cfg.HSTSPreload {
			hstsValue += "; preload"
		}
	}

	referrer := cfg.ReferrerPolicy
	if referrer == "" {
		referrer = "strict-origin-when-cross-origin"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", referrer)

			if hstsValue != "" {
				h.Set("Strict-Transport-Security", hstsValue)
			}
			if cfg.ContentSecurityPolicy != "" {
				h.Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
			}
			if cfg.PermissionsPolicy != "" {
				h.Set("Permissions-Policy", cfg.PermissionsPolicy)
			}

			next.ServeHTTP(w, r)
		})
	}
}
