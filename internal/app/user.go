package app

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jarviisha/darkvoid/internal/app/middleware"
	"github.com/jarviisha/darkvoid/internal/feature/user/handler"
	"github.com/jarviisha/darkvoid/internal/feature/user/repository"
	"github.com/jarviisha/darkvoid/internal/feature/user/service"
	"github.com/jarviisha/darkvoid/internal/infrastructure/mailer"
	"github.com/jarviisha/darkvoid/pkg/jwt"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// UserContext represents the User bounded context with all its dependencies.
type UserContext struct {
	// Repositories
	UserRepo         *repository.UserRepository
	RefreshTokenRepo *repository.RefreshTokenRepository
	FollowRepo       *repository.FollowRepository

	// Services
	UserService         *service.UserService
	RefreshTokenService *service.RefreshTokenService
	AuthService         *service.AuthService
	FollowService       *service.FollowService

	// Services (email)
	EmailService *service.EmailService

	// Handlers
	UserHandler    *handler.UserHandler
	AuthHandler    *handler.AuthHandler
	ProfileHandler *handler.ProfileHandler
	FollowHandler  *handler.FollowHandler
	EmailHandler   *handler.EmailHandler
}

// SetupUserContext initializes the User context with all required dependencies.
// secureCookie controls the Secure flag on the refresh token cookie — set to false in development (HTTP).
func SetupUserContext(pool *pgxpool.Pool, jwtService *jwt.Service, store storage.Storage, refreshTokenExpiry time.Duration, secureCookie bool, m mailer.Mailer, templates *mailer.Templates, mailerBaseURL string) *UserContext {
	// Repositories
	userRepo := repository.NewUserRepository(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(pool)
	followRepo := repository.NewFollowRepository(pool)
	emailTokenRepo := repository.NewEmailTokenRepository(pool)

	// Services
	userService := service.NewUserService(userRepo, store)
	refreshTokenService := service.NewRefreshTokenServiceWithExpiry(refreshTokenRepo, refreshTokenExpiry)
	authService := service.NewAuthService(userRepo, userService, jwtService, refreshTokenService, store)
	followService := service.NewFollowService(followRepo)
	emailService := service.NewEmailService(m, templates, emailTokenRepo, userRepo, mailerBaseURL)

	// Wire email sender into auth service for fire-and-forget after register
	authService.WithEmailSender(emailService)

	// Handlers
	userHandler := handler.NewUserHandler(userService, userService, store)
	authHandler := handler.NewAuthHandler(authService, store, secureCookie)
	profileHandler := handler.NewProfileHandler(userService, followService, store)
	followHandler := handler.NewFollowHandler(followService, userService)
	emailHandler := handler.NewEmailHandler(emailService)

	return &UserContext{
		UserRepo:            userRepo,
		RefreshTokenRepo:    refreshTokenRepo,
		FollowRepo:          followRepo,
		UserService:         userService,
		RefreshTokenService: refreshTokenService,
		AuthService:         authService,
		FollowService:       followService,
		EmailService:        emailService,
		UserHandler:         userHandler,
		AuthHandler:         authHandler,
		ProfileHandler:      profileHandler,
		FollowHandler:       followHandler,
		EmailHandler:        emailHandler,
	}
}

// RegisterRoutes registers all routes for the User context.
func (ctx *UserContext) RegisterRoutes(r chi.Router, auth middleware.AuthMiddleware) {
	// Public auth routes
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", ctx.AuthHandler.Register)
		r.Post("/login", ctx.AuthHandler.Login)
		r.Post("/refresh", ctx.AuthHandler.RefreshToken)
		r.Post("/logout", ctx.AuthHandler.Logout)
		r.Post("/verify-email", ctx.EmailHandler.VerifyEmail)
		r.Post("/resend-verification", ctx.EmailHandler.ResendVerification)
		r.Post("/forgot-password", ctx.EmailHandler.ForgotPassword)
		r.Post("/reset-password", ctx.EmailHandler.ResetPassword)
	})

	// Protected auth routes
	r.Group(func(r chi.Router) {
		r.Use(auth.Required)
		r.Get("/auth/me", ctx.AuthHandler.GetMe)
		r.Post("/auth/logout-all", ctx.AuthHandler.LogoutAllSessions)
		r.Put("/auth/password", ctx.AuthHandler.ChangePassword)
	})

	// /me — current user (protected)
	r.Group(func(r chi.Router) {
		r.Use(auth.Required)
		r.Get("/me", ctx.ProfileHandler.GetMyProfile)
		r.Put("/me", ctx.ProfileHandler.UpdateMyProfile)
		r.Put("/me/avatar", ctx.ProfileHandler.UploadAvatar)
		r.Put("/me/cover", ctx.ProfileHandler.UploadCover)
	})

	// /users/{userKey} — userKey is a UUID or username (with ?by=username)
	r.Route("/users/{userKey}", func(r chi.Router) {
		// Public — profile uses OptionalAuth to enrich is_following when viewer is logged in
		r.With(auth.Optional).Get("/profile", ctx.ProfileHandler.GetUserProfile)
		r.Get("/followers", ctx.FollowHandler.GetFollowers)
		r.Get("/following", ctx.FollowHandler.GetFollowing)

		// Protected
		r.Group(func(r chi.Router) {
			r.Use(auth.Required)
			r.Get("/", ctx.UserHandler.GetUser)
			r.Put("/", ctx.UserHandler.UpdateUser)
			r.Delete("/", ctx.UserHandler.DeactivateUser)
			r.Post("/follow", ctx.FollowHandler.Follow)
			r.Delete("/follow", ctx.FollowHandler.Unfollow)
		})
	})
}
