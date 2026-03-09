package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"
)

const (
	defaultCSRFCookieName = "_csrf"
	defaultCSRFHeaderName = "X-CSRF-Token"
	csrfTokenBytes        = 32
)

// CSRFConfig holds configuration for the CSRF middleware.
type CSRFConfig struct {
	// CookieName is the name of the CSRF cookie (default: "_csrf").
	CookieName string
	// HeaderName is the request header expected to carry the CSRF token
	// (default: "X-CSRF-Token").
	HeaderName string
	// CookieSecure sets the Secure flag on the CSRF cookie (default: true).
	// Set to false only in local development without TLS.
	CookieSecure bool
	// CookieSameSite sets the SameSite policy on the CSRF cookie
	// (default: SameSiteLaxMode).
	CookieSameSite http.SameSite
	// CookieMaxAge is the lifetime of the CSRF cookie in seconds (default: 86400).
	CookieMaxAge int
	// Skipper, when set, skips CSRF protection for matching requests.
	// By default, requests with an Authorization: Bearer header are skipped
	// because stateless API clients are not vulnerable to CSRF.
	Skipper func(r *http.Request) bool
}

// DefaultCSRFConfig returns the default CSRF middleware configuration.
func DefaultCSRFConfig() CSRFConfig {
	return CSRFConfig{
		CookieName:     defaultCSRFCookieName,
		HeaderName:     defaultCSRFHeaderName,
		CookieSecure:   true,
		CookieSameSite: http.SameSiteLaxMode,
		CookieMaxAge:   86400,
		Skipper: func(r *http.Request) bool {
			// Stateless Bearer-token clients do not use cookies and are
			// therefore not vulnerable to CSRF attacks.
			return strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ")
		},
	}
}

// CSRF returns a middleware that protects against Cross-Site Request Forgery
// using the Double Submit Cookie pattern.
//
// How it works:
//  1. On any request, if no CSRF cookie is present a new random token is set.
//  2. For state-mutating methods (POST, PUT, PATCH, DELETE) the middleware
//     validates that the X-CSRF-Token request header matches the cookie value
//     using a constant-time comparison to prevent timing-based side channels.
//  3. Requests with Authorization: Bearer are exempt (stateless API clients).
//
// Frontend usage — include the cookie value in every mutating request:
//
//	fetch('/api/resource', {
//	    method: 'POST',
//	    headers: { 'X-CSRF-Token': getCookie('_csrf') },
//	    body: JSON.stringify(data),
//	})
//
// Example:
//
//	app.Use(middleware.CSRF())
//
//	// With custom config (e.g. for HTTP-only development):
//	app.Use(middleware.CSRF(middleware.CSRFConfig{CookieSecure: false}))
func CSRF(configs ...CSRFConfig) func(http.Handler) http.Handler {
	cfg := DefaultCSRFConfig()
	if len(configs) > 0 {
		c := configs[0]
		if c.CookieName != "" {
			cfg.CookieName = c.CookieName
		}
		if c.HeaderName != "" {
			cfg.HeaderName = c.HeaderName
		}
		if c.CookieSameSite != 0 {
			cfg.CookieSameSite = c.CookieSameSite
		}
		if c.CookieMaxAge != 0 {
			cfg.CookieMaxAge = c.CookieMaxAge
		}
		cfg.CookieSecure = c.CookieSecure
		if c.Skipper != nil {
			cfg.Skipper = c.Skipper
		}
	}

	safeMethods := map[string]bool{
		http.MethodGet:     true,
		http.MethodHead:    true,
		http.MethodOptions: true,
		http.MethodTrace:   true,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Skipper != nil && cfg.Skipper(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Read existing token from cookie, or mint a new one.
			token := ""
			if cookie, err := r.Cookie(cfg.CookieName); err == nil {
				token = cookie.Value
			}
			if token == "" {
				var err error
				token, err = csrfGenerateToken()
				if err != nil {
					// crypto/rand is unavailable — fail closed rather than
					// silently degrading to weak entropy.
					http.Error(w, `{"code":500,"message":"internal server error"}`, http.StatusInternalServerError)
					return
				}
				http.SetCookie(w, &http.Cookie{
					Name:     cfg.CookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: false, // must be JS-readable for Double Submit pattern
					Secure:   cfg.CookieSecure,
					SameSite: cfg.CookieSameSite,
					MaxAge:   cfg.CookieMaxAge,
					Expires:  time.Now().Add(time.Duration(cfg.CookieMaxAge) * time.Second),
				})
			}

			// Validate token on state-mutating methods.
			// subtle.ConstantTimeCompare prevents timing-based side channels
			// that could theoretically leak token bytes in controlled environments.
			if !safeMethods[r.Method] {
				headerToken := r.Header.Get(cfg.HeaderName)
				tokenMatch := len(headerToken) > 0 &&
					subtle.ConstantTimeCompare([]byte(headerToken), []byte(token)) == 1
				if !tokenMatch {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(`{"code":403,"message":"invalid or missing CSRF token"}`)) //nolint:errcheck
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// csrfGenerateToken generates a cryptographically random CSRF token.
// Returns an error if the OS random source is unavailable — callers must
// handle this and fail closed rather than falling back to weak entropy.
func csrfGenerateToken() (string, error) {
	b := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", errors.New("csrf: failed to generate secure random token")
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
