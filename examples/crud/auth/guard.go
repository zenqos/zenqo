// Package auth provides a simple Bearer token Guard for demonstration purposes.
package auth

import (
	"net/http"
	"strings"
)

// TokenGuard validates a Bearer token from the Authorization header.
// Implements core.Guard — can be used at the route, controller, or global level.
type TokenGuard struct {
	validTokens map[string]bool
}

func NewTokenGuard(tokens ...string) *TokenGuard {
	m := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		m[t] = true
	}
	return &TokenGuard{validTokens: m}
}

func (g *TokenGuard) CanActivate(r *http.Request) (bool, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return false, nil
	}
	token := strings.TrimPrefix(header, "Bearer ")
	return g.validTokens[token], nil
}
