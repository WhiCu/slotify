package meta

import "context"

type (
	ctxDomainKey  struct{}
	ctxUseCaseKey struct{}
)

func WithDomain(ctx context.Context, domain string) context.Context {
	return context.WithValue(ctx, ctxDomainKey{}, domain)
}

func WithUseCase(ctx context.Context, usecase string) context.Context {
	return context.WithValue(ctx, ctxUseCaseKey{}, usecase)
}

func WithDomainAndUseCase(ctx context.Context, domain, usecase string) context.Context {
	ctx = WithDomain(ctx, domain)
	ctx = WithUseCase(ctx, usecase)
	return ctx
}

func DomainFromContext(ctx context.Context) (string, bool) {
	if domain, ok := ctx.Value(ctxDomainKey{}).(string); ok && domain != "" {
		return domain, true
	}
	return "", false
}

func UseCaseFromContext(ctx context.Context) (string, bool) {
	if uc, ok := ctx.Value(ctxUseCaseKey{}).(string); ok && uc != "" {
		return uc, true
	}
	return "", false
}
