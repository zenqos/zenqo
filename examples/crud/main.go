// CRUD example — User API with the recommended project structure.
//
// Run:
//
//	go run .
//
// Test:
//
//	curl http://localhost:3000/api/v1/users
//	curl -X POST http://localhost:3000/api/v1/users -d '{"name":"Charlie","email":"charlie@example.com"}'
//	curl http://localhost:3000/api/v1/users/1
//	curl -X PUT http://localhost:3000/api/v1/users/1 -d '{"name":"Alice Kim"}'
//	curl -X DELETE http://localhost:3000/api/v1/users/1
package main

import (
	"log"

	"my-app/internal/app"
	"my-app/internal/config"
)

func main() {
	cfg := config.Load()
	log.Fatal(app.New(cfg).Start(":" + cfg.Port))
}
