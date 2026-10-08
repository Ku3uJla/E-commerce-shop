package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime settings for the auth service.
type Config struct {
	HTTPAddr    string
	JWTKey      string
	JWTExpiry   time.Duration
	ServiceName string
	ServicePort int
	ConsulAddr  string
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
		HTTPAddr:    getEnv("HTTP_ADDR"),
		JWTKey:      getEnv("JWT_KEY"),
		JWTExpiry:   getDuration("JWT_EXPIRY", 24*time.Hour),
		ServiceName: getEnv("SERVICE_NAME"),
		ServicePort: getEnvInt("SERVICE_PORT", 8080),
		ConsulAddr:  getEnv("CONSUL_ADDR"),
		DB: DBConfig{
			Host:     getEnv("DB_HOST"),
			Port:     getEnv("DB_PORT"),
			User:     getEnv("DB_USER"),
			Password: getEnv("DB_PASSWORD"),
			Name:     getEnv("DB_NAME"),
			SSLMode:  getEnv("DB_SSLMODE"),
		},
	}
}

func getEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	log.Fatalf("%s is empty", key)
	return ""
}

func getDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	if d, err := time.ParseDuration(value); err == nil {
		return d
	}
	if hours, err := strconv.Atoi(value); err == nil {
		return time.Duration(hours) * time.Hour
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
