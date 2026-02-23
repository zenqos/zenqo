package config

import (
	"log"
	"os"
)

type Config struct {
	Port string
	Env  string
}

func Load() Config {
	return Config{
		Port: getEnv("PORT", "3000"),
		Env:  getEnv("APP_ENV", "development"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("[config] required environment variable %q is not set", key)
	}
	return v
}
