package app

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	appMiddleware "github.com/jarviisha/darkvoid/internal/app/middleware"
	adminHandler "github.com/jarviisha/darkvoid/internal/feature/admin/handler"
	adminService "github.com/jarviisha/darkvoid/internal/feature/admin/service"
	"github.com/jarviisha/darkvoid/internal/feature/user/db"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/internal/feature/user/repository"
	"github.com/jarviisha/darkvoid/internal/infrastructure/database"
	"github.com/jarviisha/darkvoid/pkg/storage"
)

// AdminContext holds all dependencies for the admin bounded context.
type AdminContext struct {
	RoleRepo     *repository.RoleRepository
	AdminService *adminService.AdminService
	AdminHandler *adminHandler.AdminHandler
}

// SetupAdminContext initializes the admin context.
// It receives the shared pool and user repository so that admin operations
// can reuse the same DB connection without importing the user service.
func SetupAdminContext(pool *pgxpool.Pool, store storage.Storage) *AdminContext {
	roleRepo := repository.NewRoleRepository(pool)
	userStoreAdapter := newAdminUserStoreAdapter(pool)
	svc := adminService.NewAdminService(userStoreAdapter, roleRepo, store)
	h := adminHandler.NewAdminHandler(svc)

	return &AdminContext{
		RoleRepo:     roleRepo,
		AdminService: svc,
		AdminHandler: h,
	}
}

// RegisterRoutes registers all /admin/* routes.
// All routes require authentication + admin role.
func (ctx *AdminContext) RegisterRoutes(r chi.Router, auth appMiddleware.AuthMiddleware) {
	r.Route("/admin", func(r chi.Router) {
		r.Use(auth.Required)
		r.Use(appMiddleware.RequireRole(ctx.AdminService, "admin"))

		// Stats
		r.Get("/stats", ctx.AdminHandler.GetStats)

		// User management
		r.Get("/users", ctx.AdminHandler.ListUsers)
		r.Get("/users/{id}", ctx.AdminHandler.GetUser)
		r.Patch("/users/{id}/status", ctx.AdminHandler.SetUserStatus)
		r.Get("/users/{id}/roles", ctx.AdminHandler.GetUserRoles)
		r.Post("/users/{id}/roles", ctx.AdminHandler.AssignRole)
		r.Delete("/users/{id}/roles/{roleId}", ctx.AdminHandler.RemoveRole)

		// Role management
		r.Get("/roles", ctx.AdminHandler.ListRoles)
		r.Post("/roles", ctx.AdminHandler.CreateRole)

		// Notifications
		r.Post("/notifications/users/{id}", ctx.AdminHandler.SendNotificationToUser)
		r.Post("/notifications/broadcast", ctx.AdminHandler.BroadcastNotification)
	})
}

// ─── adminUserStoreAdapter ────────────────────────────────────────────────────
// Adapts the user DB queries to the adminService.userStore interface without
// importing the user service (bounded context boundary preserved).

type adminUserStoreAdapter struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func newAdminUserStoreAdapter(pool *pgxpool.Pool) *adminUserStoreAdapter {
	return &adminUserStoreAdapter{
		queries: db.New(pool),
		pool:    pool,
	}
}

func (a *adminUserStoreAdapter) GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	row, err := a.queries.GetUserByIDAny(ctx, id)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return dbUserToAdminEntity(row), nil
}

func (a *adminUserStoreAdapter) AdminListUsers(ctx context.Context, filter adminService.AdminListUsersFilter) ([]*entity.User, error) {
	rows, err := a.queries.AdminListUsers(ctx, db.AdminListUsersParams{
		Limit:    filter.Limit,
		Offset:   filter.Offset,
		IsActive: filter.IsActive,
		Query:    filter.Query,
	})
	if err != nil {
		return nil, database.MapDBError(err)
	}
	users := make([]*entity.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, dbUserToAdminEntity(row))
	}
	return users, nil
}

func (a *adminUserStoreAdapter) AdminCountUsers(ctx context.Context, filter adminService.AdminListUsersFilter) (int64, error) {
	count, err := a.queries.AdminCountUsers(ctx, db.AdminCountUsersParams{
		IsActive: filter.IsActive,
		Query:    filter.Query,
	})
	if err != nil {
		return 0, database.MapDBError(err)
	}
	return count, nil
}

func (a *adminUserStoreAdapter) AdminSetUserActive(ctx context.Context, id uuid.UUID, isActive bool, updatedBy uuid.UUID) error {
	err := a.queries.AdminSetUserActive(ctx, db.AdminSetUserActiveParams{
		ID:        id,
		IsActive:  isActive,
		UpdatedBy: pgtype.UUID{Bytes: updatedBy, Valid: true},
	})
	return database.MapDBError(err)
}

func (a *adminUserStoreAdapter) ListAllActiveUserIDs(ctx context.Context) ([]uuid.UUID, error) {
	ids, err := a.queries.ListAllActiveUserIDs(ctx)
	if err != nil {
		return nil, database.MapDBError(err)
	}
	return ids, nil
}

// dbUserToAdminEntity converts a sqlc UsrUser to a domain User entity.
// Kept private to this adapter to avoid leaking the conversion outside app layer.
func dbUserToAdminEntity(u db.UsrUser) *entity.User {
	user := &entity.User{
		ID:             u.ID,
		Username:       u.Username,
		Email:          u.Email,
		IsActive:       u.IsActive,
		DisplayName:    u.DisplayName,
		Bio:            u.Bio,
		AvatarKey:      u.AvatarKey,
		CoverKey:       u.CoverKey,
		Website:        u.Website,
		Location:       u.Location,
		CreatedAt:      u.CreatedAt.Time,
		FollowerCount:  u.FollowerCount,
		FollowingCount: u.FollowingCount,
	}
	if u.UpdatedAt.Valid {
		t := u.UpdatedAt.Time
		user.UpdatedAt = &t
	}
	if u.CreatedBy.Valid {
		id := u.CreatedBy.Bytes
		user.CreatedBy = (*uuid.UUID)(&id)
	}
	if u.UpdatedBy.Valid {
		id := u.UpdatedBy.Bytes
		user.UpdatedBy = (*uuid.UUID)(&id)
	}
	return user
}
