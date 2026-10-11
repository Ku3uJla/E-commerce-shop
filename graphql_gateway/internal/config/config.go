package config

import "os"

// Config holds all runtime settings for the GraphQL gateway.
type Config struct {
	HTTPAddr          string
	OrderServiceURL   string
	CatalogServiceURL string
	// Dev enables introspection and Apollo Sandbox.
	Dev bool
}

// Load reads configuration from the environment, applying sensible defaults.
func Load() Config {
	return Config{
		HTTPAddr:          getEnv("HTTP_ADDR", ":4000"),
		OrderServiceURL:   getEnv("ORDER_SERVICE_URL", "http://localhost:8082"),
		CatalogServiceURL: getEnv("CATALOG_SERVICE_URL", "http://localhost:8081"),
		Dev:               getEnv("APP_ENV", "development") != "production",
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
