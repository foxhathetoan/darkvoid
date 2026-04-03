package database

import (
	"fmt"
	"os"
	"strconv"
)

// LoadConfigFromEnv loads database configuration from environment variables
func LoadConfigFromEnv() *Config {
	cfg := DefaultConfig()

	cfg.Host = getEnv("DB_HOST", cfg.Host)
	cfg.User = getEnv("DB_USER", cfg.User)
	cfg.Password = getEnv("DB_PASSWORD", cfg.Password)
	cfg.Database = getEnv("DB_NAME", cfg.Database)
	cfg.SSLMode = getEnv("DB_SSLMODE", cfg.SSLMode)

	cfg.Port = getEnvInt("DB_PORT", cfg.Port)
	cfg.MaxConns = int32(getEnvInt("DB_MAX_CONNS", int(cfg.MaxConns))) //nolint:gosec // env config values are small
	cfg.MinConns = int32(getEnvInt("DB_MIN_CONNS", int(cfg.MinConns))) //nolint:gosec // env config values are small

	return cfg
}

// LoadConfigFromDSN loads database configuration from connection string
func LoadConfigFromDSN(dsn string) (*Config, error) {
	// For simple use case, return default and use DSN directly
	// In production, you might want to parse DSN
	cfg := DefaultConfig()

	if dsn == "" {
		return nil, fmt.Errorf("DSN cannot be empty")
	}

	// This is simplified - you can use pgx.ParseConfig to parse DSN properly
	return cfg, nil
}

// ToConnectionString converts Config to PostgreSQL connection string
func (c *Config) ToConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s pool_max_conns=%d pool_min_conns=%d",
		c.Host,
		c.Port,
		c.User,
		c.Password,
		c.Database,
		c.SSLMode,
		c.MaxConns,
		c.MinConns,
	)
}

// Validate validates database configuration
func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Port == 0 {
		return fmt.Errorf("database port is required")
	}
	if c.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database == "" {
		return fmt.Errorf("database name is required")
	}
	if c.MaxConns < 1 {
		return fmt.Errorf("max connections must be at least 1")
	}
	if c.MinConns < 0 {
		return fmt.Errorf("min connections cannot be negative")
	}
	if c.MinConns > c.MaxConns {
		return fmt.Errorf("min connections cannot exceed max connections")
	}
	if c.MaxConnLifetime < 0 {
		return fmt.Errorf("max connection lifetime cannot be negative")
	}
	if c.MaxConnIdleTime < 0 {
		return fmt.Errorf("max connection idle time cannot be negative")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	s := getEnv(key, "")
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return fallback
}

// Example environment variables:
// export DB_HOST=localhost
// export DB_PORT=5432
// export DB_USER=postgres
// export DB_PASSWORD=secret
// export DB_NAME=darkvoid
// export DB_SSLMODE=disable
// export DB_MAX_CONNS=25
// export DB_MIN_CONNS=5
