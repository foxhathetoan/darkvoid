package database

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestConfig returns database configuration for testing
func TestConfig() *Config {
	return &Config{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		Database: "darkvoid_test",
		SSLMode:  "disable",
		MaxConns: 10,
		MinConns: 2,
	}
}

// SetupTestDB creates a test database connection
// Note: This requires a running PostgreSQL instance
func SetupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()
	cfg := TestConfig()

	pool, err := NewPostgresPool(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create test database pool: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

// TruncateTables truncates all tables in test database
func TruncateTables(ctx context.Context, pool *pgxpool.Pool, tables ...string) error {
	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		if _, err := pool.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}
	return nil
}

// SeedTestData is a helper to seed test data
// Usage in tests:
//
//	pool := database.SetupTestDB(t)
//	defer database.TruncateTables(ctx, pool, "users", "roles", "user_roles")
func SeedTestData(ctx context.Context, pool *pgxpool.Pool, setupFunc func(context.Context, *pgxpool.Pool) error) error {
	return setupFunc(ctx, pool)
}

// For integration tests using testcontainers (future enhancement):
//
// import "github.com/testcontainers/testcontainers-go/modules/postgres"
//
// func SetupTestContainer(t *testing.T) *pgxpool.Pool {
//     ctx := context.Background()
//
//     container, err := postgres.RunContainer(ctx,
//         testcontainers.WithImage("postgres:16-alpine"),
//         postgres.WithDatabase("darkvoid_test"),
//         postgres.WithUsername("postgres"),
//         postgres.WithPassword("postgres"),
//     )
//     if err != nil {
//         t.Fatal(err)
//     }
//
//     connStr, _ := container.ConnectionString(ctx)
//     pool, _ := pgxpool.New(ctx, connStr)
//
//     t.Cleanup(func() {
//         pool.Close()
//         container.Terminate(ctx)
//     })
//
//     return pool
// }
