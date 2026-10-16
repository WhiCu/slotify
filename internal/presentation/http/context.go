package http

import (
	"context"
	"net/netip"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/whicu/slotify/internal/domain/user"
)

type (
	ctxIDKey   struct{}
	ctxRoleKey struct{}
	ctxIPKey   struct{}
)

var (
	userIDContextKey   = ctxIDKey{}
	userRoleContextKey = ctxRoleKey{}
	clientIPContextKey = ctxIPKey{}
)

func WithUserID(ctx context.Context, id user.UserID) context.Context {
	return context.WithValue(ctx, userIDContextKey, id)
}

func UserIDFromContext(ctx context.Context) (user.UserID, bool) {
	id, ok := ctx.Value(userIDContextKey).(user.UserID)
	if !ok {
		return user.UserID{}, false
	}
	return id, true
}

func WithClientIP(ctx context.Context, ip netip.Addr) context.Context {
	return context.WithValue(ctx, clientIPContextKey, ip)
}

func ClientIPFromContext(ctx context.Context) (netip.Addr, bool) {
	ip, ok := ctx.Value(clientIPContextKey).(netip.Addr)
	return ip, ok
}

func WithUserRole(ctx context.Context, role user.Role) context.Context {
	return context.WithValue(ctx, userRoleContextKey, role)
}

func UserRoleFromContext(ctx context.Context) (user.Role, bool) {
	role, ok := ctx.Value(userRoleContextKey).(user.Role)
	if !ok {
		return user.Unknown, false
	}
	return role, true
}

func RequestIDFromContext(ctx context.Context) (string, bool) {
	if reqID := middleware.GetReqID(ctx); reqID != "" {
		return reqID, true
	}
	return "", false
}

func WithRequestID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, middleware.RequestIDKey, reqID)
}
