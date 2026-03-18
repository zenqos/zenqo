// Basic example — a working Zenqo server out of the box.
//
// Run:
//
//	zenqo dev
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
	if err := app.New(cfg).Start(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
