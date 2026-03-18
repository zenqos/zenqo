package openapi

// SecurityDef pairs a scheme name with its definition for use in Config.Security.
type SecurityDef struct {
	name   string
	scheme *SecurityScheme
}

// BearerAuth creates a Bearer token (JWT) security scheme.
//
// Example:
//
//	openapi.Mount(app, openapi.Config{
//	    Security: []openapi.SecurityDef{openapi.BearerAuth()},
//	})
func BearerAuth() SecurityDef {
	return SecurityDef{
		name: "bearerAuth",
		scheme: &SecurityScheme{
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
		},
	}
}

// APIKeyHeader creates an API key security scheme sent via a custom header.
//
// Example:
//
//	openapi.APIKeyHeader("X-API-Key")
func APIKeyHeader(headerName string) SecurityDef {
	return SecurityDef{
		name: "apiKeyHeader",
		scheme: &SecurityScheme{
			Type: "apiKey",
			In:   "header",
			Name: headerName,
		},
	}
}

// APIKeyCookie creates an API key security scheme sent via a cookie.
func APIKeyCookie(cookieName string) SecurityDef {
	return SecurityDef{
		name: "apiKeyCookie",
		scheme: &SecurityScheme{
			Type: "apiKey",
			In:   "cookie",
			Name: cookieName,
		},
	}
}

// OAuth2Config holds OAuth2 flow configuration.
type OAuth2Config struct {
	AuthorizationURL string
	TokenURL         string
	Scopes           map[string]string
}

// OAuth2AuthCode creates an OAuth2 authorization code flow security scheme.
func OAuth2AuthCode(cfg OAuth2Config) SecurityDef {
	return SecurityDef{
		name: "oauth2",
		scheme: &SecurityScheme{
			Type: "oauth2",
			Flows: &OAuthFlows{
				AuthorizationCode: &OAuthFlow{
					AuthorizationURL: cfg.AuthorizationURL,
					TokenURL:         cfg.TokenURL,
					Scopes:           cfg.Scopes,
				},
			},
		},
	}
}
