package config

import "os"

type Config struct {
	Port      string
	Env       string
	JWTSecret string
}

func Load() Config {
	return Config{
		Port:      getEnv("PORT", "3000"),
		Env:       getEnv("APP_ENV", "development"),
		JWTSecret: getEnv("JWT_SECRET", "zenqo-example-secret-do-not-use-in-production"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
