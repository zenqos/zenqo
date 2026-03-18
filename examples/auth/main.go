// Auth example — JWT authentication with Guards, Interceptors, and Bind+Validation.
//
// Run:
//
//	zenqo dev
//
// Test:
//
//	# Register a user
//	curl -X POST http://localhost:3000/api/v1/auth/register -H 'Content-Type: application/json' -d '{"name":"Alice","email":"alice@example.com","password":"secret123"}'
//
//	# Login (returns JWT token)
//	curl -X POST http://localhost:3000/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"alice@example.com","password":"secret123"}'
//
//	# Access protected route (replace TOKEN with the JWT from login)
//	curl http://localhost:3000/api/v1/users/me -H "Authorization: Bearer TOKEN"
//
//	# List all users (protected)
//	curl http://localhost:3000/api/v1/users/ -H "Authorization: Bearer TOKEN"
package main

import (
	"log"

	"my-app/internal/app"
	"my-app/internal/config"
)

func main() {
	cfg := config.Load()
	if err := app.New(cfg).Start(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
