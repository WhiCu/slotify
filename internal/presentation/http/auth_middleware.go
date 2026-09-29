package http

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"uuid"

	api "github.com/whicu/slotify/api/http"
	"github.com/whicu/slotify/internal/domain/user"
	"github.com/whicu/slotify/internal/infrastructure/crypto"
)

var (
	ErrUnauthenticated = errors.New("http: user unauthenticated")
	ErrForbidden       = errors.New("http: insufficient permissions")
)

type SecurityHandler struct {
	log      *slog.Logger
	verifier *crypto.TokenCodec
}

func NewSecurityHandler(log *slog.Logger, verifier *crypto.TokenCodec) *SecurityHandler {
	return &SecurityHandler{
		log:      log,
		verifier: verifier,
	}
}

func (h *SecurityHandler) HandleBearerAuth(ctx context.Context, op api.OperationName, t api.BearerAuth) (context.Context, error) {
	h.log.DebugContext(ctx, "authenticating request",
		slog.String("operation", op),
		slog.String("token", t.Token),
		slog.Any("roles", t.Roles),
	)

	userIDRaw, roleRaw, err := h.verifier.Decode2Type[string, string](
		t.Token,
		"user_id",
		"role",
	)
	if err != nil {
		switch {
		case errors.Is(err, crypto.ErrTokenExpired):
			h.log.WarnContext(ctx, "bearer token expired",
				slog.String("operation", op),
				slog.Any("error", err),
			)
			return ctx, ErrUnauthenticated
		case errors.Is(err, crypto.ErrTokenMalformed):
			h.log.WarnContext(ctx, "bearer token malformed",
				slog.String("operation", op),
				slog.Any("error", err),
			)
			return ctx, ErrUnauthenticated
		default:
			h.log.ErrorContext(ctx, "failed to verify bearer token",
				slog.String("operation", op),
				slog.Any("error", err),
			)
			return ctx, err
		}
	}

	userID, err := uuid.Parse(userIDRaw)
	if err != nil {
		h.log.WarnContext(
			ctx,
			"bearer token contains invalid user id",
			slog.String("operation", op),
			slog.Any("error", err),
		)
		return ctx, ErrUnauthenticated
	}

	role, err := user.RoleFromString(roleRaw)
	if err != nil {
		h.log.WarnContext(
			ctx,
			"bearer token contains invalid role",
			slog.String("operation", op),
			slog.Any("error", err),
		)
		return ctx, ErrUnauthenticated
	}

	if len(t.Roles) > 0 && !roleAllowed(role, t.Roles) {
		h.log.WarnContext(ctx, "privileged endpoint rejected: insufficient role",
			slog.String("operation", op),
			slog.String("user_id", userID.String()),
			slog.String("token_role", role.String()),
			slog.Any("required_roles", t.Roles),
		)
		return ctx, ErrForbidden
	}

	h.log.DebugContext(ctx, "request successfully authenticated",
		slog.String("operation", op),
		slog.String("user_id", userID.String()),
	)

	return WithUserID(ctx, user.UserID(userID)), nil
}

func roleAllowed(role user.Role, required []string) bool {
	return slices.Contains(required, role.String())
}
