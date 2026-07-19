package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration, validated at startup.
type Config struct {
	// Database
	DBUser     string
	DBPassword string
	DBHost     string
	DBPort     string
	DBName     string

	// RabbitMQ
	RabbitMQURL string

	// SMTP
	SMTPFrom     string
	SMTPHost     string
	SMTPPort     int
	SMTPPassword string

	// App
	AppPort string

	// Observability
	OTELEndpoint string
}

// Load reads .env (if present) and validates required environment variables.
// It returns an error if any required variaNewTracerProviderble is missing or invalid.
func Load() (*Config, error) {
	// Load .env file; ignore error if file doesn't exist (production uses real env vars)
	_ = godotenv.Load()

	cfg := &Config{
		DBUser:       os.Getenv("DB_USER"),
		DBPassword:   os.Getenv("DB_PASSWORD"),
		DBHost:       getEnvWithDefault("DB_HOST", "localhost"),
		DBPort:       getEnvWithDefault("DB_PORT", "5432"),
		DBName:       os.Getenv("DB_NAME"),
		RabbitMQURL:  os.Getenv("RABBITMQ_URL"),
		SMTPFrom:     os.Getenv("SMTP_FROM"),
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		AppPort:      getEnvWithDefault("APP_PORT", "8080"),
		OTELEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}

	// Parse SMTP port with a sensible default
	smtpPortStr := getEnvWithDefault("SMTP_PORT", "587")
	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT %q: %w", smtpPortStr, err)
	}
	cfg.SMTPPort = smtpPort

	// Validate required variables
	var missing []string
	if cfg.DBUser == "" {
		missing = append(missing, "DB_USER")
	}
	if cfg.DBPassword == "" {
		missing = append(missing, "DB_PASSWORD")
	}
	if cfg.DBName == "" {
		missing = append(missing, "DB_NAME")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}

func getEnvWithDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// GetEnv is retained for backward compatibility but prefer using Config struct directly.
func GetEnv(key string) string {
	return os.Getenv(key)
}
