package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port         int
	DatabaseURL  string
	SlotDuration time.Duration
}

func Load() (*Config, error) {
	slotDuration, err := time.ParseDuration(os.Getenv("USER_SLOT_DURATION"))
	if err != nil {
		return nil, fmt.Errorf("parse USER_SLOT_DURATION: %w", err)
	}

	portStr := os.Getenv("PROFILE_PORT")
	if portStr == "" {
		return nil, fmt.Errorf("PROFILE_PORT is required")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("PROFILE_PORT must be a valid port between 1 and 65535")
	}

	databaseURL := os.Getenv("PROFILE_DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("PROFILE_DATABASE_URL is required")
	}

	return &Config{
		Port:         port,
		DatabaseURL:  databaseURL,
		SlotDuration: slotDuration,
	}, nil
}
