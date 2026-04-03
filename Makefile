.PHONY: help sqlc-generate sqlc-clean swagger-init swagger-generate swagger-serve build run test test-v test-cover test-cover-html test-feature lint deps clean dev docker-up docker-down docker-logs migrate-up migrate-down migrate-up-user migrate-up-post migrate-up-notification migrate-down-notification migrate-create migrate-status migrate-force db-reset generate install-tools

# Load .env if it exists
-include .env
export

# Tool paths
MIGRATE := $(shell go env GOPATH)/bin/migrate

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

sqlc-generate: ## Generate SQLC code from SQL files
	sqlc generate

sqlc-clean: ## Clean generated SQLC code
	rm -rf internal/feature/user/db/*.go internal/feature/post/db/*.go internal/feature/notification/db/*.go

swagger-init: ## Initialize Swagger in the project (run once)
	swag init -g cmd/api/main.go -o docs --parseInternal

swagger-generate: ## Generate/update Swagger documentation
	swag fmt
	swag init -g cmd/api/main.go -o docs --parseInternal

swagger-serve: ## Serve Swagger UI locally (requires swagger-ui-dist)
	@echo "Swagger docs generated at: docs/swagger.json"
	@echo "View at: http://localhost:8080/swagger/index.html (when server is running)"

build: ## Build the application
	go build -o bin/api ./cmd/api

run: ## Run the application
	go run ./cmd/api

test: ## Run all tests
	go test ./...

test-v: ## Run all tests with verbose output
	go test -v ./...

test-cover: ## Run all tests with coverage report
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out
	@rm -f coverage.out

test-cover-html: ## Run all tests and open HTML coverage report
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out
	@rm -f coverage.out

test-feature: ## Run tests for a specific feature (usage: make test-feature feature=user)
	@if [ -z "$(feature)" ]; then \
		echo "Error: feature required. Usage: make test-feature feature=user"; \
		exit 1; \
	fi
	go test -v ./internal/feature/$(feature)/...

lint: ## Run golangci-lint
	golangci-lint run

deps: ## Install dependencies
	go mod download
	go mod tidy

clean: ## Clean build artifacts
	rm -rf bin/

dev: ## Run in development mode with hot reload (requires air)
	air

docker-up: ## Start Docker containers (PostgreSQL, etc.)
	docker-compose up -d

docker-down: ## Stop Docker containers
	docker-compose down

docker-logs: ## View Docker container logs
	docker-compose logs -f

migrate-up: ## Run all pending migrations (user, post, notification)
	@echo "Running user migrations..."
	$(MIGRATE) -path migrations/user -database "$(DATABASE_URL)&x-migrations-table=schema_migrations_user" up
	@echo "Running post migrations..."
	$(MIGRATE) -path migrations/post -database "$(DATABASE_URL)&x-migrations-table=schema_migrations_post" up
	@echo "Running notification migrations..."
	$(MIGRATE) -path migrations/notification -database "$(DATABASE_URL)&x-migrations-table=schema_migrations_notification" up

migrate-down: ## Rollback last migration for all modules (notification, post, user)
	@echo "Rolling back notification migrations..."
	$(MIGRATE) -path migrations/notification -database "$(DATABASE_URL)&x-migrations-table=schema_migrations_notification" down 1
	@echo "Rolling back post migrations..."
	$(MIGRATE) -path migrations/post -database "$(DATABASE_URL)&x-migrations-table=schema_migrations_post" down 1
	@echo "Rolling back user migrations..."
	$(MIGRATE) -path migrations/user -database "$(DATABASE_URL)&x-migrations-table=schema_migrations_user" down 1

migrate-up-user: ## Run pending migrations for user module only
	@echo "Running user migrations..."
	$(MIGRATE) -path migrations/user -database "$(DATABASE_URL)&x-migrations-table=schema_migrations_user" up

migrate-up-post: ## Run pending migrations for post module only
	@echo "Running post migrations..."
	$(MIGRATE) -path migrations/post -database "$(DATABASE_URL)&x-migrations-table=schema_migrations_post" up

migrate-up-notification: ## Run pending migrations for notification module only
	@echo "Running notification migrations..."
	$(MIGRATE) -path migrations/notification -database "$(DATABASE_URL)&x-migrations-table=schema_migrations_notification" up

migrate-down-notification: ## Rollback last migration for notification module
	@echo "Rolling back notification migrations..."
	$(MIGRATE) -path migrations/notification -database "$(DATABASE_URL)&x-migrations-table=schema_migrations_notification" down 1

migrate-create: ## Create a new migration (usage: make migrate-create module=user name=migration_name)
	@if [ -z "$(name)" ]; then \
		echo "Error: migration name required. Usage: make migrate-create module=user name=migration_name"; \
		exit 1; \
	fi
	@if [ -z "$(module)" ]; then \
		echo "Error: module required. Usage: make migrate-create module=user name=migration_name"; \
		exit 1; \
	fi
	@echo "Creating migration: $(name) in module: $(module)"
	$(MIGRATE) create -ext sql -dir migrations/$(module) -seq $(name)

migrate-status: ## Show current migration status for all modules
	@echo "=== User migration status ==="
	-$(MIGRATE) -path migrations/user -database "$(DATABASE_URL)&x-migrations-table=schema_migrations_user" version
	@echo "=== Post migration status ==="
	-$(MIGRATE) -path migrations/post -database "$(DATABASE_URL)&x-migrations-table=schema_migrations_post" version
	@echo "=== Notification migration status ==="
	-$(MIGRATE) -path migrations/notification -database "$(DATABASE_URL)&x-migrations-table=schema_migrations_notification" version

migrate-force: ## Force migration to specific version (usage: make migrate-force module=user version=1)
	@if [ -z "$(version)" ]; then \
		echo "Error: version required. Usage: make migrate-force module=user version=1"; \
		exit 1; \
	fi
	@if [ -z "$(module)" ]; then \
		echo "Error: module required. Usage: make migrate-force module=user version=1"; \
		exit 1; \
	fi
	@echo "WARNING: Forcing $(module) migration to version $(version)"
	$(MIGRATE) -path migrations/$(module) -database "$(DATABASE_URL)&x-migrations-table=schema_migrations_$(module)" force $(version)

db-reset: ## Reset database (drop and recreate)
	@echo "WARNING: This will delete all data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose down -v; \
		docker-compose up -d; \
	fi

generate: sqlc-generate swagger-generate ## Generate all code (SQLC + Swagger)

install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/cosmtrek/air@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "Done! Tools installed."
