package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/zenqos/zenqo/core"
)

type ctxKey string

const ClaimsKey ctxKey = "claims"

// JWTGuard implements core.Guard to protect routes with JWT authentication.
// Returns 401 Unauthorized (using the improved Guard interface) when the token
// is missing or invalid.
type JWTGuard struct {
	Secret string
}

func (g *JWTGuard) CanActivate(r *http.Request) (bool, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return false, core.ErrUnauthorized("missing authorization header")
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return false, core.ErrUnauthorized("invalid authorization format")
	}

	claims, err := ParseToken(g.Secret, parts[1])
	if err != nil {
		return false, core.ErrUnauthorized("invalid or expired token")
	}

	// Store claims in context for handlers to use
	ctx := context.WithValue(r.Context(), ClaimsKey, claims)
	*r = *r.WithContext(ctx)
	return true, nil
}

// GetClaims extracts JWT claims from the request context.
func GetClaims(r *http.Request) *Claims {
	c, _ := r.Context().Value(ClaimsKey).(*Claims)
	return c
}
