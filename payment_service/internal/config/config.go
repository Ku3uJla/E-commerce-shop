package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime settings for the payment service.
type Config struct {
	HTTPAddr    string
	ServiceName string
	ServicePort int
	ConsulAddr  string
	// KafkaBrokers is a comma-separated list of host:port.
	KafkaBrokers []string
	// MaxAmount (cents): the simulated gateway declines bigger payments.
	MaxAmount int64
	// FailureRate (0..1): share of payments the simulated gateway declines at random.
	FailureRate float64
	DB          DBConfig
}

// DBConfig holds PostgreSQL connection settings.
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// DSN renders the PostgreSQL connection string.
func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

// Load reads configuration from the environment, applying sensible defaults.
func Load() Config {
	return Config{
		HTTPAddr:     getEnv("HTTP_ADDR", ":8083"),
		ServiceName:  getEnv("SERVICE_NAME", "payment-service"),
		ServicePort:  getEnvInt("SERVICE_PORT", 8083),
		ConsulAddr:   getEnv("CONSUL_ADDR", "consul:8500"),
		KafkaBrokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		MaxAmount:    int64(getEnvInt("PAYMENT_MAX_AMOUNT", 100000)),
		FailureRate:  getEnvFloat("PAYMENT_FAILURE_RATE", 0),
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "payment_service"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvFloat(key string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(os.Getenv(key), 64)
	if err != nil {
		return fallback
	}
	return parsed
}
