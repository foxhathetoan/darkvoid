package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jarviisha/darkvoid/docs" // register swag docs
	"github.com/jarviisha/darkvoid/internal/app/middleware"
	"github.com/jarviisha/darkvoid/internal/infrastructure/mailer"
	"github.com/jarviisha/darkvoid/pkg/config"
	"github.com/jarviisha/darkvoid/pkg/database"
	"github.com/jarviisha/darkvoid/pkg/jwt"
	"github.com/jarviisha/darkvoid/pkg/logger"
	pkgredis "github.com/jarviisha/darkvoid/pkg/redis"
	"github.com/jarviisha/darkvoid/pkg/storage"
	swaggerFiles "github.com/swaggo/files"
	"github.com/swaggo/swag"
)

// Application represents the entire application with all its dependencies
type Application struct {
	cfg        *config.Config
	log        *logger.Logger
	pool       *pgxpool.Pool
	redis      *pkgredis.Client // nil when Redis is disabled
	server     *Server
	jwtService *jwt.Service

	// Bounded Contexts
	User         *UserContext
	Storage      *StorageContext
	Post         *PostContext
	Feed         *FeedContext
	Notification *NotificationContext
	Search       *SearchContext
	Admin        *AdminContext
}

// New creates and initializes a new Application
func New(ctx context.Context, cfg *config.Config) (*Application, error) {
	app := &Application{
		cfg: cfg,
	}

	// Setup logger
	app.setupLogger()

	app.log.Info("application initializing",
		"service", cfg.App.Name,
		"version", cfg.App.Version,
		"environment", cfg.App.Environment,
	)

	// Setup database
	if err := app.setupDatabase(ctx); err != nil {
		return nil, fmt.Errorf("failed to setup database: %w", err)
	}

	// Setup Redis (optional — skipped when REDIS_ENABLED=false)
	if err := app.setupRedis(ctx); err != nil {
		return nil, fmt.Errorf("failed to setup redis: %w", err)
	}

	// Setup JWT service
	if err := app.setupJWT(); err != nil {
		return nil, fmt.Errorf("failed to setup JWT: %w", err)
	}

	// Initialize bounded contexts
	if err := app.setupContexts(); err != nil {
		return nil, fmt.Errorf("failed to setup contexts: %w", err)
	}

	// Auto-bootstrap root user from env vars (Cách 3)
	if err := app.bootstrapRootUser(ctx); err != nil {
		return nil, fmt.Errorf("failed to bootstrap root user: %w", err)
	}

	// Setup HTTP server
	app.setupServer()

	app.log.Info("application initialized successfully")
	return app, nil
}

// setupLogger initializes the application logger
func (app *Application) setupLogger() {
	app.log = logger.New(&logger.Config{
		Level:       app.cfg.Logger.Level,
		Format:      app.cfg.Logger.Format,
		Output:      os.Stdout, // Ensure output is set
		AddSource:   app.cfg.Logger.AddSource,
		Service:     app.cfg.App.Name,
		Version:     app.cfg.App.Version,
		Environment: app.cfg.App.Environment,
	})

	logger.SetDefault(app.log)

	app.log.Info("logger initialized",
		"level", app.cfg.Logger.Level,
		"format", app.cfg.Logger.Format,
	)
}

// setupDatabase initializes database connection pool
func (app *Application) setupDatabase(ctx context.Context) error {
	app.log.Info("initializing database connection")

	dbConfig := &database.Config{
		Host:            app.cfg.Database.Host,
		Port:            app.cfg.Database.Port,
		User:            app.cfg.Database.User,
		Password:        app.cfg.Database.Password,
		Database:        app.cfg.Database.Database,
		SSLMode:         app.cfg.Database.SSLMode,
		MaxConns:        app.cfg.Database.MaxConns,
		MinConns:        app.cfg.Database.MinConns,
		MaxConnLifetime: app.cfg.Database.MaxConnLifetime,
		MaxConnIdleTime: app.cfg.Database.MaxConnIdleTime,
	}

	app.log.Debug("database configuration",
		"host", dbConfig.Host,
		"port", dbConfig.Port,
		"database", dbConfig.Database,
		"max_conns", dbConfig.MaxConns,
	)

	// Create connection pool
	pool, err := database.NewPostgresPool(ctx, dbConfig)
	if err != nil {
		return err
	}

	// Ensure cleanup on error
	defer func() {
		if err != nil && pool != nil {
			pool.Close()
		}
	}()

	// Health check
	if err = database.HealthCheck(ctx, pool); err != nil {
		return err
	}

	app.pool = pool

	// Log connection stats
	stats := database.GetStats(pool)
	app.log.Info("database connected successfully",
		"max_conns", stats.MaxConns,
		"total_conns", stats.TotalConns,
		"idle_conns", stats.IdleConns,
	)

	return nil
}

// setupRedis initializes Redis client. Skipped when REDIS_ENABLED=false.
func (app *Application) setupRedis(ctx context.Context) error {
	if !app.cfg.Redis.Enabled {
		app.log.Info("redis disabled (REDIS_ENABLED=false), feed cache will be no-op")
		return nil
	}

	app.log.Info("initializing redis connection", "host", app.cfg.Redis.Host, "port", app.cfg.Redis.Port)

	client, err := pkgredis.New(ctx, &pkgredis.Config{
		Host:     app.cfg.Redis.Host,
		Port:     app.cfg.Redis.Port,
		Password: app.cfg.Redis.Password,
		DB:       app.cfg.Redis.DB,
		PoolSize: app.cfg.Redis.PoolSize,
	})
	if err != nil {
		return err
	}

	app.redis = client
	app.log.Info("redis connected successfully")
	return nil
}

// setupJWT initializes JWT service
func (app *Application) setupJWT() error {
	app.log.Info("initializing JWT service")

	// TODO: Get JWT secret from config or environment
	// For now using a placeholder - CHANGE THIS IN PRODUCTION!
	jwtConfig := jwt.Config{
		Secret: []byte(app.cfg.JWT.Secret),
		Issuer: app.cfg.JWT.Issuer,
		Expiry: app.cfg.JWT.AccessTokenExpiry,
	}

	jwtService, err := jwt.NewService(jwtConfig)
	if err != nil {
		return fmt.Errorf("failed to create JWT service: %w", err)
	}

	app.jwtService = jwtService

	app.log.Info("JWT service initialized",
		"issuer", jwtConfig.Issuer,
		"expiry", jwtConfig.Expiry,
	)

	return nil
}

// setupContexts initializes all bounded contexts
func (app *Application) setupContexts() error {
	app.log.Info("initializing bounded contexts")

	// Initialize storage provider from config
	store, err := storage.New(storage.Config{
		Provider: app.cfg.Storage.Provider,
		BaseURL:  app.cfg.Storage.BaseURL,
		LocalDir: app.cfg.Storage.LocalDir,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	app.log.Info("storage initialized", "provider", app.cfg.Storage.Provider, "base_url", app.cfg.Storage.BaseURL)

	// Initialize mailer
	m := mailer.New(mailer.Config{
		Provider: app.cfg.Mailer.Provider,
		Host:     app.cfg.Mailer.Host,
		Port:     app.cfg.Mailer.Port,
		Username: app.cfg.Mailer.Username,
		Password: app.cfg.Mailer.Password,
		From:     app.cfg.Mailer.From,
		BaseURL:  app.cfg.Mailer.BaseURL,
	})
	app.log.Info("mailer initialized", "provider", app.cfg.Mailer.Provider)

	templates, err := mailer.LoadTemplates()
	if err != nil {
		return fmt.Errorf("failed to load email templates: %w", err)
	}

	// Initialize User context (account management + profile + follow + email)
	app.User = SetupUserContext(app.pool, app.jwtService, store, app.cfg.RefreshToken.Expiry, !app.cfg.IsDevelopment(), m, templates, app.cfg.Mailer.BaseURL)
	app.log.Info("user context initialized")

	// Initialize Storage context (media upload)
	app.Storage = SetupStorageContext(store)
	app.log.Info("storage context initialized")

	// Initialize Post context
	app.Post = SetupPostContext(app.pool, store, app.User.UserRepo, app.redis)
	app.log.Info("post context initialized")

	// Initialize Feed context (depends on Post and User)
	app.Feed = SetupFeedContext(store, app.Post, app.User, app.User.FollowService, app.redis, app.cfg.Dike)
	app.log.Info("feed context initialized")

	// Initialize Notification context (depends on User, Post, Follow)
	app.Notification = SetupNotificationContext(app.pool, store, app.User, app.Post, app.User.FollowService, app.redis)
	app.log.Info("notification context initialized")

	// Initialize Search context (depends on User and Post repos)
	app.Search = SetupSearchContext(app.pool, app.User.UserRepo, app.Post.HashtagRepo, store)
	app.log.Info("search context initialized")

	// Initialize Admin context (role management + user administration)
	app.Admin = SetupAdminContext(app.pool, store)
	app.Admin.AdminService.WithNotificationEmitter(app.Notification.NotifService)
	app.log.Info("admin context initialized")

	return nil
}

// bootstrapRootUser creates the initial root user from env vars if no users exist yet.
// Skipped when ROOT_EMAIL or ROOT_PASSWORD is not set.
func (app *Application) bootstrapRootUser(ctx context.Context) error {
	cfg := app.cfg.Root
	if cfg.Email == "" || cfg.Password == "" {
		app.log.Info("root bootstrap disabled (ROOT_EMAIL or ROOT_PASSWORD not set)")
		return nil
	}

	created, err := app.User.UserService.BootstrapRootUser(
		ctx,
		cfg.Email,
		cfg.Password,
		cfg.Username,
		cfg.DisplayName,
	)
	if err != nil {
		return err
	}
	if created {
		app.log.Info("root user bootstrapped",
			"email", cfg.Email,
			"username", cfg.Username,
		)
	}
	return nil
}

// setupServer initializes HTTP server and registers routes
func (app *Application) setupServer() {
	app.log.Info("initializing HTTP server")

	app.server = NewServer(app.cfg, app.log, app.pool)

	// Register routes
	app.registerRoutes()

	app.log.Info("HTTP server initialized",
		"host", app.cfg.Server.Host,
		"port", app.cfg.Server.Port,
	)
}

// registerRoutes registers all HTTP routes
func (app *Application) registerRoutes() {
	router := app.server.Router()

	// Swagger UI — two separate docs: app (public API) and admin.
	//
	// /swagger/app/   — endpoints for the mobile/web application (excludes admin tag)
	// /swagger/admin/ — endpoints for the admin panel (admin tag only, auth-protected)
	router.Get("/swagger/app/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/app/doc.json"),
		httpSwagger.InstanceName("app"),
	))
	router.Get("/swagger/app/doc.json", swaggerFilterHandler(false, false))
	router.Get("/swagger/app/doc.yaml", swaggerFilterHandler(false, true))

	router.Get("/swagger/admin/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/admin/doc.json"),
		httpSwagger.InstanceName("admin"),
	))
	router.Get("/swagger/admin/doc.json", swaggerFilterHandler(true, false))
	router.Get("/swagger/admin/doc.yaml", swaggerFilterHandler(true, true))

	// Static file server for local storage provider
	if app.cfg.Storage.Provider == "local" {
		absDir, _ := filepath.Abs(app.cfg.Storage.LocalDir)
		router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(absDir))))
		app.log.Info("static file server mounted", "dir", absDir, "path", "/static/")
	}

	// Build auth middleware once, shared by all bounded contexts
	auth := middleware.NewAuthMiddleware(app.jwtService)

	// API v1 routes — two sibling groups sharing the same /api/v1 prefix.
	//
	// Group A (no timeout): SSE streaming endpoint only.
	//   chi captures the middleware stack at Group creation time, so routes in
	//   this group never see middleware.Timeout and keep a plain ResponseWriter
	//   that implements http.Flusher — required for SSE.
	//
	// Group B (with timeout): all regular REST endpoints.
	router.Route("/api/v1", func(r chi.Router) {
		// Group A — SSE, no request timeout
		r.Group(func(r chi.Router) {
			app.Notification.RegisterSSERoute(r, auth)
		})

		// Group B — REST endpoints, request timeout enforced
		r.Group(func(r chi.Router) {
			r.Use(app.server.RequestTimeout())

			app.User.RegisterRoutes(r, auth)
			app.Storage.RegisterRoutes(r, auth)
			app.Post.RegisterRoutes(r, auth)
			app.Feed.RegisterRoutes(r, auth)
			app.Notification.RegisterRoutes(r, auth)
			app.Search.RegisterRoutes(r, auth)
			app.Admin.RegisterRoutes(r, auth)
		})
	})
}

// Start starts the HTTP server
func (app *Application) Start() error {
	app.log.Info("starting application",
		"host", app.cfg.Server.Host,
		"port", app.cfg.Server.Port,
	)

	// Start notification SSE broker (Redis Pub/Sub subscriber)
	app.Notification.StartBroker(context.Background())

	if err := app.server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the application
func (app *Application) Shutdown(ctx context.Context) error {
	app.log.Info("shutting down application")

	// Close SSE connections before shutting down the HTTP server so that
	// long-lived Stream handlers return promptly instead of blocking Shutdown.
	if app.Notification != nil && app.Notification.Broker != nil {
		app.log.Info("closing SSE connections")
		app.Notification.Broker.Shutdown()
	}

	// Shutdown HTTP server
	if app.server != nil {
		app.log.Info("shutting down HTTP server")
		if err := app.server.Shutdown(ctx); err != nil {
			app.log.Error("failed to shutdown HTTP server", "error", err)
			return err
		}
	}

	// Close Redis connection
	if app.redis != nil {
		app.log.Info("closing redis connection")
		_ = app.redis.Close()
	}

	// Close database connection
	if app.pool != nil {
		app.log.Info("closing database connections")
		database.Close(app.pool)
	}

	app.log.Info("application stopped")
	return nil
}

// GracefulShutdown performs graceful shutdown with timeout
func (app *Application) GracefulShutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return app.Shutdown(ctx)
}

// swaggerFilterHandler returns an HTTP handler that serves a filtered subset
// of the generated Swagger spec.
//
//   - adminOnly=true  → keep only paths that have at least one operation tagged "admin"
//   - adminOnly=false → exclude all paths where every operation is tagged "admin"
//
// This lets us serve two Swagger UIs from a single swag-generated spec:
//
//	/swagger/app/doc.json   — app (mobile/web) endpoints
//	/swagger/admin/doc.json — admin panel endpoints
func swaggerFilterHandler(adminOnly bool, asYAML bool) http.HandlerFunc {
	// swaggerFiles side-effect import ensures the embed FS is populated.
	_ = swaggerFiles.Handler
	return func(w http.ResponseWriter, r *http.Request) {
		raw := swag.GetSwagger("swagger")
		if raw == nil {
			http.Error(w, "swagger spec not found", http.StatusNotFound)
			return
		}

		var spec map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw.ReadDoc()), &spec); err != nil {
			http.Error(w, "failed to parse swagger spec", http.StatusInternalServerError)
			return
		}

		// Filter the "paths" object.
		if rawPaths, ok := spec["paths"]; ok {
			var paths map[string]map[string]json.RawMessage
			if err := json.Unmarshal(rawPaths, &paths); err == nil {
				filtered := make(map[string]map[string]json.RawMessage, len(paths))
				for path, methods := range paths {
					if keepPath(methods, adminOnly) {
						filtered[path] = methods
					}
				}
				if b, err := json.Marshal(filtered); err == nil {
					spec["paths"] = b
				}
			}
		}

		// Filter "definitions" to only those referenced by the kept paths.
		if rawDefs, ok := spec["definitions"]; ok {
			var defs map[string]json.RawMessage
			if err := json.Unmarshal(rawDefs, &defs); err == nil {
				used := make(map[string]bool)
				if b, err := json.Marshal(spec["paths"]); err == nil {
					collectDefRefs(b, used)
				}
				// Transitively expand: a definition may reference other definitions.
				for {
					prev := len(used)
					for name := range used {
						if defRaw, exists := defs[name]; exists {
							collectDefRefs(defRaw, used)
						}
					}
					if len(used) == prev {
						break
					}
				}
				filtered := make(map[string]json.RawMessage, len(used))
				for name := range used {
					if raw, exists := defs[name]; exists {
						filtered[name] = raw
					}
				}
				if b, err := json.Marshal(filtered); err == nil {
					spec["definitions"] = b
				}
			}
		}

		if asYAML {
			// Convert the JSON spec to a plain map so yaml.Marshal produces clean YAML.
			var specAny map[string]any
			out, err := json.Marshal(spec)
			if err != nil {
				http.Error(w, "failed to encode swagger spec", http.StatusInternalServerError)
				return
			}
			if err = json.Unmarshal(out, &specAny); err != nil {
				http.Error(w, "failed to decode swagger spec", http.StatusInternalServerError)
				return
			}
			yamlOut, err := yaml.Marshal(specAny)
			if err != nil {
				http.Error(w, "failed to encode swagger spec as yaml", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/yaml")
			_, _ = w.Write(yamlOut)
			return
		}

		out, err := json.Marshal(spec)
		if err != nil {
			http.Error(w, "failed to encode swagger spec", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(out)
	}
}

// keepPath reports whether a path entry should be included given the filter mode.
//
//   - adminOnly=true  → keep paths whose operations are tagged "admin" or "auth"
//   - adminOnly=false → keep paths whose operations are not tagged "admin"
//
// "auth" paths are included in both specs so that both Swagger UIs expose the
// authentication collection.
func keepPath(methods map[string]json.RawMessage, adminOnly bool) bool {
	for method, raw := range methods {
		// Skip non-operation keys such as "parameters".
		if method == "parameters" {
			continue
		}
		var op struct {
			Tags []string `json:"tags"`
		}
		if err := json.Unmarshal(raw, &op); err != nil {
			continue
		}
		isAdmin := slices.Contains(op.Tags, "admin")
		isAuth := slices.Contains(op.Tags, "auth")
		if adminOnly && !isAdmin && !isAuth {
			return false
		}
		if !adminOnly && isAdmin {
			return false
		}
	}
	return true
}

// collectDefRefs finds all "#/definitions/<Name>" $ref values in data
// and adds each Name to the used set.
func collectDefRefs(data []byte, used map[string]bool) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return
	}
	walkDefRefs(v, used)
}

func walkDefRefs(v any, used map[string]bool) {
	switch val := v.(type) {
	case map[string]any:
		for k, child := range val {
			if k == "$ref" {
				if ref, ok := child.(string); ok && strings.HasPrefix(ref, "#/definitions/") {
					used[strings.TrimPrefix(ref, "#/definitions/")] = true
				}
				continue
			}
			walkDefRefs(child, used)
		}
	case []any:
		for _, child := range val {
			walkDefRefs(child, used)
		}
	}
}
