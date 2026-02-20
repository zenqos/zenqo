// Package main demonstrates a full CRUD API built with Zenqo.
//
// Run:
//
//	go run examples/crud/main.go
//
// Try it:
//
//	# List users (public)
//	curl http://localhost:3000/api/v1/users
//
//	# Create a user (requires auth)
//	curl -X POST http://localhost:3000/api/v1/users \
//	  -H "Content-Type: application/json" \
//	  -H "Authorization: Bearer secret-token" \
//	  -d '{"name":"Alice","email":"alice@example.com"}'
//
//	# Get a single user (public)
//	curl http://localhost:3000/api/v1/users/1
//
//	# Update a user (requires auth)
//	curl -X PUT http://localhost:3000/api/v1/users/1 \
//	  -H "Content-Type: application/json" \
//	  -H "Authorization: Bearer secret-token" \
//	  -d '{"name":"Alice Smith"}'
//
//	# Delete a user (requires auth)
//	curl -X DELETE http://localhost:3000/api/v1/users/1 \
//	  -H "Authorization: Bearer secret-token"
//
//	# Access without token → 403
//	curl -X POST http://localhost:3000/api/v1/users \
//	  -H "Content-Type: application/json" \
//	  -d '{"name":"Bob"}'
package main

import (
	"log"

	"zenqo/core"
	"zenqo/examples/crud/auth"
	"zenqo/examples/crud/user"
)

func main() {
	// Shared dependencies — created once and injected into modules that need them.
	tokenGuard := auth.NewTokenGuard("secret-token", "admin-token")

	app := core.NewApp().
		SetGlobalPrefix("/api/v1").
		UseModule(user.NewModule(tokenGuard))

	log.Fatal(app.Start(":3000"))
}
