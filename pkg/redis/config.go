package redis

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds Redis connection configuration.
type Config struct {
	Host         string
	Port         int
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
}

// DefaultConfig returns sensible defaults for local development.
func DefaultConfig() *Config {
	return &Config{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	}
}

// LoadConfigFromEnv loads Redis configuration from environment variables.
//
//	REDIS_HOST         (default: localhost)
//	REDIS_PORT         (default: 6379)
//	REDIS_PASSWORD     (default: "")
//	REDIS_DB           (default: 0)
//	REDIS_POOL_SIZE    (default: 10)
func LoadConfigFromEnv() *Config {
	cfg := DefaultConfig()
	cfg.Host = getEnv("REDIS_HOST", cfg.Host)
	cfg.Port = getEnvInt("REDIS_PORT", cfg.Port)
	cfg.Password = getEnv("REDIS_PASSWORD", cfg.Password)
	cfg.DB = getEnvInt("REDIS_DB", cfg.DB)
	cfg.PoolSize = getEnvInt("REDIS_POOL_SIZE", cfg.PoolSize)
	return cfg
}

// Addr returns "host:port".
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Validate checks that required fields are set.
func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("redis host is required")
	}
	if c.Port == 0 {
		return fmt.Errorf("redis port is required")
	}
	if c.DB < 0 {
		return fmt.Errorf("redis DB must be >= 0")
	}
	if c.PoolSize < 1 {
		return fmt.Errorf("redis pool size must be at least 1")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, err := strconv.Atoi(getEnv(key, "")); err == nil {
		return v
	}
	return fallback
}
