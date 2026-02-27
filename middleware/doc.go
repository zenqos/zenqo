// Package middleware provides built-in HTTP middleware for Zenqo applications.
//
// Available middleware:
//   - [RequestID] — injects a unique request ID into each request
//   - [RealIP] — extracts the client's real IP from proxy headers
//   - [Logger] — logs method, path, status code, and duration
//   - [CORS] — Cross-Origin Resource Sharing with configurable origins, methods, and headers
//   - [SecureHeaders] — security headers (X-Content-Type-Options, X-Frame-Options)
//
// Usage:
//
//	app := core.NewApp()
//	app.Use(middleware.SecureHeaders())
//	app.Use(middleware.CORS())
package middleware
