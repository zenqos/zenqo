package middleware

import (
	"net"
	"net/http"
	"strings"
)

// RealIP is a middleware that sets r.RemoteAddr to the client's real IP
// extracted from X-Forwarded-For or X-Real-Ip headers.
//
// Only use this behind a trusted reverse proxy; otherwise clients can
// spoof their IP address.
func RealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ip := realIP(r); ip != "" {
			r.RemoteAddr = ip
		}
		next.ServeHTTP(w, r)
	})
}

func realIP(r *http.Request) string {
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
