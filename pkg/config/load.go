package config

import "time"

// loadAppConfig loads application configuration
func loadAppConfig() AppConfig {
	return AppConfig{
		Name:        getEnv("SERVICE_NAME", "darkvoid"),
		Version:     getEnv("SERVICE_VERSION", "1.0.0"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

// loadDatabaseConfig loads database configuration
func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvInt("DB_PORT", 5432),
		User:            getEnv("DB_USER", "postgres"),
		Password:        getEnv("DB_PASSWORD", "postgres"),
		Database:        getEnv("DB_NAME", "darkvoid"),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxConns:        int32(getEnvInt("DB_MAX_CONNS", 25)), //nolint:gosec // env config values are small
		MinConns:        int32(getEnvInt("DB_MIN_CONNS", 5)),  //nolint:gosec // env config values are small
		MaxConnLifetime: getEnvDuration("DB_MAX_CONN_LIFETIME", time.Hour),
		MaxConnIdleTime: getEnvDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
	}
}

// loadLoggerConfig loads logger configuration
func loadLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:     getEnv("LOG_LEVEL", "info"),
		Format:    getEnv("LOG_FORMAT", "json"),
		AddSource: getEnvBool("LOG_ADD_SOURCE", false),
	}
}

// loadServerConfig loads server configuration
func loadServerConfig() ServerConfig {
	return ServerConfig{
		Host:              getEnv("SERVER_HOST", "0.0.0.0"),
		Port:              getEnvInt("SERVER_PORT", 8080),
		ReadTimeout:       getEnvDuration("SERVER_READ_TIMEOUT", 10*time.Second),
		WriteTimeout:      getEnvDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:       getEnvDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
		RequestTimeout:    getEnvDuration("SERVER_REQUEST_TIMEOUT", 60*time.Second),
		AllowedOrigins:    getEnvSlice("CORS_ALLOWED_ORIGINS", []string{"*"}),
		RateLimitRequests: getEnvInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:   getEnvDuration("RATE_LIMIT_WINDOW", 1*time.Minute),
	}
}

// loadStorageConfig loads storage configuration
func loadStorageConfig() StorageConfig {
	return StorageConfig{
		Provider: getEnv("STORAGE_PROVIDER", "local"),
		BaseURL:  getEnv("STORAGE_BASE_URL", "http://localhost:8080/static"),
		LocalDir: getEnv("STORAGE_LOCAL_DIR", "./uploads"),

		S3Endpoint:  getEnv("STORAGE_S3_ENDPOINT", ""),
		S3Bucket:    getEnv("STORAGE_S3_BUCKET", "darkvoid"),
		S3Region:    getEnv("STORAGE_S3_REGION", "us-east-1"),
		S3AccessKey: getEnv("STORAGE_S3_ACCESS_KEY", ""),
		S3SecretKey: getEnv("STORAGE_S3_SECRET_KEY", ""),
		S3UseSSL:    getEnvBool("STORAGE_S3_USE_SSL", false),
	}
}

// loadJWTConfig loads JWT configuration
func loadJWTConfig() JWTConfig {
	return JWTConfig{
		Secret:            getEnv("JWT_SECRET", ""),
		Issuer:            getEnv("JWT_ISSUER", "darkvoid"),
		AccessTokenExpiry: getEnvDuration("JWT_ACCESS_TOKEN_EXPIRY", 15*time.Minute),
	}
}

// loadRefreshTokenConfig loads refresh token configuration
func loadRefreshTokenConfig() RefreshTokenConfig {
	return RefreshTokenConfig{
		Expiry: getEnvDuration("REFRESH_TOKEN_EXPIRY", 7*24*time.Hour),
	}
}

// loadRootConfig loads root bootstrap configuration.
// ROOT_EMAIL and ROOT_PASSWORD must both be set to enable auto-bootstrap.
func loadRootConfig() RootConfig {
	return RootConfig{
		Email:       getEnv("ROOT_EMAIL", ""),
		Password:    getEnv("ROOT_PASSWORD", ""),
		Username:    getEnv("ROOT_USERNAME", "root"),
		DisplayName: getEnv("ROOT_DISPLAY_NAME", "Root Admin"),
	}
}

// loadDikeConfig loads Dike ranking service configuration from environment variables.
// Set DIKE_ENABLED=true to delegate feed scoring to Dike via gRPC.
//
//	DIKE_ENABLED    (default: false)
//	DIKE_ADDR       (default: localhost:50051)
//	DIKE_PROJECT_ID (default: darkvoid)
//	DIKE_API_KEY    (default: "")
//	DIKE_MODEL_ID   (default: "" → uses project default model)
func loadDikeConfig() DikeConfig {
	return DikeConfig{
		Enabled:   getEnvBool("DIKE_ENABLED", false),
		Addr:      getEnv("DIKE_ADDR", "localhost:50051"),
		ProjectID: getEnv("DIKE_PROJECT_ID", "darkvoid"),
		APIKey:    getEnv("DIKE_API_KEY", ""),
		ModelID:   getEnv("DIKE_MODEL_ID", ""),
	}
}

// loadMailerConfig loads mailer configuration from environment variables.
// Set MAILER_PROVIDER=smtp to enable real email delivery.
//
//	MAILER_PROVIDER (default: nop)
//	MAILER_HOST     (default: "")
//	MAILER_PORT     (default: 587)
//	MAILER_USERNAME (default: "")
//	MAILER_PASSWORD (default: "")
//	MAILER_FROM     (default: "noreply@darkvoid.app")
//	MAILER_BASE_URL (default: "http://localhost:3000")
func loadMailerConfig() MailerConfig {
	return MailerConfig{
		Provider: getEnv("MAILER_PROVIDER", "nop"),
		Host:     getEnv("MAILER_HOST", ""),
		Port:     getEnvInt("MAILER_PORT", 587),
		Username: getEnv("MAILER_USERNAME", ""),
		Password: getEnv("MAILER_PASSWORD", ""),
		From:     getEnv("MAILER_FROM", "noreply@darkvoid.app"),
		BaseURL:  getEnv("MAILER_BASE_URL", "http://localhost:3000"),
	}
}

// loadRedisConfig loads Redis configuration from environment variables.
// Set REDIS_ENABLED=true to enable caching; all other vars have sensible defaults.
//
//	REDIS_ENABLED   (default: false)
//	REDIS_HOST      (default: localhost)
//	REDIS_PORT      (default: 6379)
//	REDIS_PASSWORD  (default: "")
//	REDIS_DB        (default: 0)
//	REDIS_POOL_SIZE (default: 10)
func loadRedisConfig() RedisConfig {
	return RedisConfig{
		Enabled:  getEnvBool("REDIS_ENABLED", false),
		Host:     getEnv("REDIS_HOST", "localhost"),
		Port:     getEnvInt("REDIS_PORT", 6379),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       getEnvInt("REDIS_DB", 0),
		PoolSize: getEnvInt("REDIS_POOL_SIZE", 10),
	}
}
