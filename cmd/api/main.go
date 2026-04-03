package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jarviisha/darkvoid/internal/app"
	"github.com/jarviisha/darkvoid/pkg/config"

	_ "github.com/jarviisha/darkvoid/docs" // Import generated Swagger docs
)

//	@title			DarkVoid API
//	@version		1.0
//	@description	Social network platform API using bounded context architecture
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.email	support@darkvoid.com

//	@license.name	MIT
//	@license.url	https://opensource.org/licenses/MIT

//	@host		localhost:8080
//	@BasePath	/api/v1

// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Type "Bearer" followed by a space and JWT token.
func main() {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize application
	application, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Start application server in goroutine
	go func() {
		if err := application.Start(); err != nil {
			log.Fatalf("Failed to start application: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	// Graceful shutdown with 30 second timeout
	if err := application.GracefulShutdown(30 * time.Second); err != nil {
		log.Fatalf("Failed to gracefully shutdown: %v", err)
	}
}
