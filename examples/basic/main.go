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
