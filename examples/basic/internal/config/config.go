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

// getEnv returns the env value or a fallback default.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// mustGetEnv returns the env value or exits if not set.
// Use this for required secrets like DB credentials, API keys, etc.
func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("[config] required environment variable %q is not set", key)
	}
	return v
}
