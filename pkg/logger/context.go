package logger

import (
	"context"
	"log/slog"
)

type ExtractorFunc func(ctx context.Context) []slog.Attr

type ctxAttrsKey struct{}

func AttrsExtractor(ctx context.Context) []slog.Attr {
	if attrs, ok := ctx.Value(ctxAttrsKey{}).([]slog.Attr); ok {
		return attrs
	}
	return nil
}

type ContextHandler struct {
	next       slog.Handler
	extractors []ExtractorFunc
}

func NewContextHandler(next slog.Handler, extractors ...ExtractorFunc) *ContextHandler {
	return &ContextHandler{
		next:       next,
		extractors: extractors,
	}
}

func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		for _, extract := range h.extractors {
			if attrs := extract(ctx); len(attrs) > 0 {
				r.AddAttrs(attrs...)
			}
		}
	}
	return h.next.Handle(ctx, r)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{
		next:       h.next.WithAttrs(attrs),
		extractors: h.extractors,
	}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{
		next:       h.next.WithGroup(name),
		extractors: h.extractors,
	}
}
