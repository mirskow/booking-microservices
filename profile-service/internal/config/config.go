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

	SlotDuration time.Duration

	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	HTTPIdleTimeout  time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort: getEnv("HTTP_PORT", "8080"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		SlotDuration: getDurationEnv(
			"SLOT_DURATION",
			60*time.Minute,
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

	if err := validatePort(cfg.HTTPPort); err != nil {
		return nil, fmt.Errorf("HTTP_PORT: %w", err)
	}

	if cfg.SlotDuration <= 0 {
		return nil, fmt.Errorf("SLOT_DURATION must be greater than zero")
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

func getDurationEnv(
	key string,
	defaultValue time.Duration,
) time.Duration {
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

func validatePort(port string) error {
	value, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("must be a number")
	}

	if value < 1 || value > 65535 {
		return fmt.Errorf("must be between 1 and 65535")
	}

	return nil
}
