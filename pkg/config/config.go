package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	App          AppConfig
	Database     DatabaseConfig
	Logger       LoggerConfig
	Server       ServerConfig
	JWT          JWTConfig
	RefreshToken RefreshTokenConfig
	Storage      StorageConfig
	Root         RootConfig
	Redis        RedisConfig
	Dike         DikeConfig
	Mailer       MailerConfig
}

// DikeConfig holds configuration for the Dike ranking service.
// Set DIKE_ENABLED=true to delegate feed scoring to Dike via gRPC.
// When disabled, the local formula-based scorer is used.
type DikeConfig struct {
	Enabled   bool   // use Dike for scoring instead of local formula
	Addr      string // gRPC address, e.g. "localhost:50051"
	ProjectID string // Dike project identifier
	APIKey    string // Dike API key
	ModelID   string // optional: model ID or alias (empty = project default)
}

// RedisConfig holds Redis connection configuration.
// When Enabled is false, the feed falls back to pull on-the-fly (no caching).
type RedisConfig struct {
	Enabled  bool
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

// RootConfig holds bootstrap configuration for the initial root/admin account.
// When ROOT_EMAIL and ROOT_PASSWORD are set, the app auto-creates a root user on
// first startup if no users exist yet.
type RootConfig struct {
	// Email is the root user's email address. Leave empty to disable auto-bootstrap.
	Email string
	// Password is the root user's initial plaintext password.
	// It is only used during bootstrap; the value is never stored.
	Password string
	// Username is the root user's login username (default: "root").
	Username string
	// DisplayName is the root user's display name (default: "Root Admin").
	DisplayName string
}

// StorageConfig holds file storage configuration
type StorageConfig struct {
	// Provider selects the storage backend: "local" or "s3"
	Provider string

	// BaseURL is the public base URL used to build file URLs from keys.
	// e.g. "http://localhost:8080/static" or "https://cdn.example.com"
	BaseURL string

	// Local provider settings
	LocalDir string // e.g. "./uploads"

	// S3-compatible provider settings (S3, MinIO, GCS)
	S3Endpoint  string
	S3Bucket    string
	S3Region    string
	S3AccessKey string
	S3SecretKey string
	S3UseSSL    bool
}

// MailerConfig holds email sending configuration.
// Set MAILER_PROVIDER=smtp to enable real email delivery.
// When set to "nop" (default), emails are logged but not sent.
type MailerConfig struct {
	Provider string // "smtp" or "nop"
	Host     string
	Port     int
	Username string
	Password string
	From     string
	BaseURL  string // application URL for building links in emails
}

// AppConfig holds application-level configuration
type AppConfig struct {
	Name        string
	Version     string
	Environment string // development, staging, production
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level     string // debug, info, warn, error
	Format    string // json, text
	AddSource bool
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host              string
	Port              int
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	RequestTimeout    time.Duration // Request timeout for middleware
	AllowedOrigins    []string      // CORS allowed origins
	RateLimitRequests int           // Rate limit: requests per window
	RateLimitWindow   time.Duration // Rate limit: time window
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret            string
	Issuer            string
	AccessTokenExpiry time.Duration
}

// RefreshTokenConfig holds refresh token configuration
type RefreshTokenConfig struct {
	Expiry time.Duration
}

// Load loads configuration from environment variables.
// It automatically loads .env file if present (silently ignored if not found).
func Load() (*Config, error) {
	// Load .env file if it exists — errors are silently ignored (e.g. production)
	_ = godotenv.Load()

	cfg := &Config{
		App:          loadAppConfig(),
		Database:     loadDatabaseConfig(),
		Logger:       loadLoggerConfig(),
		Server:       loadServerConfig(),
		JWT:          loadJWTConfig(),
		RefreshToken: loadRefreshTokenConfig(),
		Storage:      loadStorageConfig(),
		Root:         loadRootConfig(),
		Redis:        loadRedisConfig(),
		Dike:         loadDikeConfig(),
		Mailer:       loadMailerConfig(),
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate app config
	if c.App.Name == "" {
		return fmt.Errorf("app name is required")
	}

	// Validate database config
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Port == 0 {
		return fmt.Errorf("database port is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}
	if c.Database.MaxConns < 1 {
		return fmt.Errorf("database max connections must be at least 1")
	}
	if c.Database.MinConns < 0 {
		return fmt.Errorf("database min connections cannot be negative")
	}
	if c.Database.MinConns > c.Database.MaxConns {
		return fmt.Errorf("database min connections cannot exceed max connections")
	}

	// Validate logger config
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logger.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logger.Level)
	}

	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[c.Logger.Format] {
		return fmt.Errorf("invalid log format: %s", c.Logger.Format)
	}

	// Validate server config
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Validate JWT config
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}
	if c.JWT.Issuer == "" {
		return fmt.Errorf("JWT issuer is required")
	}
	if c.JWT.AccessTokenExpiry <= 0 {
		return fmt.Errorf("JWT access token expiry must be positive")
	}
	if c.RefreshToken.Expiry <= 0 {
		return fmt.Errorf("refresh token expiry must be positive")
	}

	return nil
}

// IsDevelopment checks if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}

// IsProduction checks if running in production environment
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

// IsStaging checks if running in staging environment
func (c *Config) IsStaging() bool {
	return c.App.Environment == "staging"
}
