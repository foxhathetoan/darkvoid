package httputil

import (
	"context"

	"github.com/google/uuid"
)

// Context keys for storing values in HTTP request context
type contextKey string

const (
	// ContextKeyUserID stores the authenticated user ID
	ContextKeyUserID contextKey = "user_id"

	// ContextKeyRequestID stores the request ID for tracing
	ContextKeyRequestID contextKey = "request_id"

	// ContextKeyTenantID stores the tenant/organization ID for multi-tenancy
	ContextKeyTenantID contextKey = "tenant_id"
)

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) *uuid.UUID {
	if val := ctx.Value(ContextKeyUserID); val != nil {
		if id, ok := val.(uuid.UUID); ok {
			return &id
		}
	}
	return nil
}

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, userID)
}

// GetRequestID extracts request ID from context
func GetRequestID(ctx context.Context) string {
	if val := ctx.Value(ContextKeyRequestID); val != nil {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestID, requestID)
}

// GetTenantID extracts tenant ID from context
func GetTenantID(ctx context.Context) *uuid.UUID {
	if val := ctx.Value(ContextKeyTenantID); val != nil {
		if id, ok := val.(uuid.UUID); ok {
			return &id
		}
	}
	return nil
}

// WithTenantID adds tenant ID to context
func WithTenantID(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, ContextKeyTenantID, tenantID)
}
