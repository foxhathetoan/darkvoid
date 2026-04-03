package logger

import (
	"os"
	"strings"
)

// LoadConfigFromEnv loads logger configuration from environment variables
func LoadConfigFromEnv() *Config {
	cfg := DefaultConfig()

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.Level = strings.ToLower(level)
	}

	if format := os.Getenv("LOG_FORMAT"); format != "" {
		cfg.Format = strings.ToLower(format)
	}

	if addSource := os.Getenv("LOG_ADD_SOURCE"); addSource == "true" {
		cfg.AddSource = true
	}

	if service := os.Getenv("SERVICE_NAME"); service != "" {
		cfg.Service = service
	}

	if version := os.Getenv("SERVICE_VERSION"); version != "" {
		cfg.Version = version
	}

	if env := os.Getenv("ENVIRONMENT"); env != "" {
		cfg.Environment = env
	}

	return cfg
}

// Development returns logger config for development
func Development() *Config {
	return &Config{
		Level:       "debug",
		Format:      "text",
		Output:      os.Stdout,
		AddSource:   true,
		Environment: "development",
	}
}

// Production returns logger config for production
func Production() *Config {
	return &Config{
		Level:       "info",
		Format:      "json",
		Output:      os.Stdout,
		AddSource:   false,
		Environment: "production",
	}
}

// Testing returns logger config for testing
func Testing() *Config {
	return &Config{
		Level:       "debug",
		Format:      "text",
		Output:      os.Stdout,
		AddSource:   false,
		Environment: "testing",
	}
}
