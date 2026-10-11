package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime settings for the notification service.
type Config struct {
	HTTPAddr     string
	ServiceName  string
	ServicePort  int
	ConsulAddr   string
	KafkaBrokers []string
}

// Load reads configuration from the environment, applying sensible defaults.
func Load() Config {
	return Config{
		HTTPAddr:     getEnv("HTTP_ADDR", ":8084"),
		ServiceName:  getEnv("SERVICE_NAME", "notification-service"),
		ServicePort:  getEnvInt("SERVICE_PORT", 8084),
		ConsulAddr:   getEnv("CONSUL_ADDR", "consul:8500"),
		KafkaBrokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	parsed, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return parsed
}
