// Package middleware provides built-in HTTP middleware for Zenqo applications.
//
// Available middleware:
//   - [CORS] — Cross-Origin Resource Sharing with configurable origins, methods, and headers
//   - [SecureHeaders] — security headers (X-Content-Type-Options, X-Frame-Options)
//
// Usage:
//
//	app := core.NewApp()
//	app.Use(middleware.SecureHeaders())
//	app.Use(middleware.CORS())
package middleware
