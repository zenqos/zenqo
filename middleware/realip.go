package middleware

import (
	"net"
	"net/http"
	"strings"
)

// RealIP is a middleware that sets r.RemoteAddr to the client's real IP
// extracted from X-Forwarded-For or X-Real-Ip headers.
//
// WARNING: This middleware unconditionally trusts the forwarded headers.
// Any client can send a spoofed X-Forwarded-For header, bypassing
// rate limiting, audit logging, and any IP-based access controls.
//
// Only use RealIP when ALL of the following are true:
//   - The application sits behind a trusted reverse proxy (nginx, AWS ALB, etc.)
//   - The proxy strips or overwrites X-Forwarded-For before forwarding
//
// For production deployments, prefer RealIPWithConfig with an explicit
// TrustedProxies allowlist so only headers from known proxy CIDRs are trusted:
//
//	app.Use(middleware.RealIPWithConfig(middleware.RealIPConfig{
//	    TrustedProxies: []string{"10.0.0.0/8", "172.16.0.0/12"},
//	}))
func RealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ip := realIPFromHeaders(r); ip != "" {
			r.RemoteAddr = ip
		}
		next.ServeHTTP(w, r)
	})
}

// RealIPConfig holds configuration for the RealIPWithConfig middleware.
type RealIPConfig struct {
	// TrustedProxies is a list of CIDR ranges whose forwarded-for headers
	// will be trusted. If empty, all sources are trusted (same as RealIP).
	//
	// Example:
	//   TrustedProxies: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	TrustedProxies []string
}

// RealIPWithConfig is like RealIP but only rewrites r.RemoteAddr when the
// direct peer IP is within one of the configured trusted proxy CIDR ranges.
//
// This prevents clients from spoofing X-Forwarded-For to bypass IP-based
// controls such as rate limiting, audit logging, or geo-restrictions.
//
// Example:
//
//	app.Use(middleware.RealIPWithConfig(middleware.RealIPConfig{
//	    TrustedProxies: []string{"10.0.0.0/8", "172.16.0.0/12"},
//	}))
func RealIPWithConfig(cfg RealIPConfig) func(http.Handler) http.Handler {
	nets := parseCIDRs(cfg.TrustedProxies)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(nets) == 0 || isTrustedProxy(r.RemoteAddr, nets) {
				if ip := realIPFromHeaders(r); ip != "" {
					r.RemoteAddr = ip
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// realIPFromHeaders extracts the client IP from X-Forwarded-For or X-Real-Ip.
// Returns "" if neither header is present or contains a valid IP.
func realIPFromHeaders(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first (client) IP from the comma-separated list.
		if i := strings.IndexByte(xff, ','); i > 0 {
			xff = xff[:i]
		}
		ip := strings.TrimSpace(xff)
		if net.ParseIP(ip) != nil {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		ip := strings.TrimSpace(xri)
		if net.ParseIP(ip) != nil {
			return ip
		}
	}
	return ""
}

// parseCIDRs parses a slice of CIDR strings, silently ignoring invalid entries.
func parseCIDRs(cidrs []string) []*net.IPNet {
	result := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, ipNet, err := net.ParseCIDR(c)
		if err == nil {
			result = append(result, ipNet)
		}
	}
	return result
}

// isTrustedProxy reports whether the given addr (host:port or bare IP) falls
// within any of the trusted CIDR ranges.
func isTrustedProxy(addr string, nets []*net.IPNet) bool {
	host := addr
	if h, _, err := net.SplitHostPort(addr); err == nil {
		host = h
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
