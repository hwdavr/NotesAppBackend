package config

import (
	"log"
	"os"
)

type Config struct {
	Addr          string
	DatabaseURL   string
	Auth0Issuer   string
	Auth0Audience string
	Auth0JWKSURL  string
}

func FromEnv() Config {
	return Config{
		Addr:          getenv("ADDR", ":8080"),
		DatabaseURL:   must("DATABASE_URL"),
		Auth0Issuer:   getenv("AUTH0_ISSUER", "https://dev-9sa8k5kv.us.auth0.com/"),
		Auth0Audience: getenv("AUTH0_AUDIENCE", "https://notes-app.api"),
		Auth0JWKSURL:  getenv("AUTH0_JWKS_URL", "https://dev-9sa8k5kv.us.auth0.com/.well-known/jwks.json"),
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func must(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("missing required env %s", key)
	}
	return value
}
