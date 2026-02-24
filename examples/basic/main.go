// Basic example — a working Zenqo server out of the box.
//
// Run:
//
//	go run .
//
// Test:
//
//	curl http://localhost:3000/
//	curl http://localhost:3000/health
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
