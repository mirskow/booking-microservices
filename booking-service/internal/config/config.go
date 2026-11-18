package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort string

	DatabaseURL string

	UserServiceURL         string
	NotificationServiceURL string

	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	HTTPIdleTimeout  time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort: getEnv("HTTP_PORT", "8080"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		UserServiceURL: getEnv(
			"USER_SERVICE_URL",
			"http://localhost:8081",
		),

		NotificationServiceURL: getEnv(
			"NOTIFICATION_SERVICE_URL",
			"http://localhost:8082",
		),

		HTTPReadTimeout: getDurationEnv(
			"HTTP_READ_TIMEOUT",
			10*time.Second,
		),

		HTTPWriteTimeout: getDurationEnv(
			"HTTP_WRITE_TIMEOUT",
			15*time.Second,
		),

		HTTPIdleTimeout: getDurationEnv(
			"HTTP_IDLE_TIMEOUT",
			60*time.Second,
		),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if _, err := strconv.Atoi(cfg.HTTPPort); err != nil {
		return nil, fmt.Errorf("invalid HTTP_PORT: %w", err)
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	return value
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}

	return duration
}
