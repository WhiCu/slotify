package di

import (
	"context"
	"log/slog"

	metaapp "github.com/whicu/slotify/internal/application/meta"
	"github.com/whicu/slotify/internal/presentation/http"
	"github.com/whicu/slotify/pkg/logger"
)

func UserIdExtractor() logger.ExtractorFunc {
	return func(ctx context.Context) []slog.Attr {
		var attrs []slog.Attr
		if userID, ok := http.UserIDFromContext(ctx); ok {
			attrs = append(attrs, slog.String("user_id", userID.String()))
		}
		return attrs
	}
}

func UserRoleExtractor() logger.ExtractorFunc {
	return func(ctx context.Context) []slog.Attr {
		var attrs []slog.Attr
		if role, ok := http.UserRoleFromContext(ctx); ok {
			attrs = append(attrs, slog.String("user_role", role.String()))
		}
		return attrs
	}
}

func RequestIdExtractor() logger.ExtractorFunc {
	return func(ctx context.Context) []slog.Attr {
		var attrs []slog.Attr
		if reqID, ok := http.RequestIDFromContext(ctx); ok {
			attrs = append(attrs, slog.String("request_id", reqID))
		}
		return attrs
	}
}

func DomainExtractor() logger.ExtractorFunc {
	return func(ctx context.Context) []slog.Attr {
		var attrs []slog.Attr
		if domain, ok := metaapp.DomainFromContext(ctx); ok {
			attrs = append(attrs, slog.String("s", domain))
		}
		return attrs
	}
}

func UseCaseExtractor() logger.ExtractorFunc {
	return func(ctx context.Context) []slog.Attr {
		var attrs []slog.Attr
		if useCase, ok := metaapp.UseCaseFromContext(ctx); ok {
			attrs = append(attrs, slog.String("event_name", useCase))
		}
		return attrs
	}
}
