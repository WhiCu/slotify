package meta

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	Tracer = otel.Tracer("github.com/whicu/slotify/internal/application")
	Meter  = otel.Meter("github.com/whicu/slotify/internal/application")

	ucDuration, _ = Meter.Float64Histogram(
		"slotify.usecase.duration",
		metric.WithDescription("Duration of application use cases"),
		metric.WithUnit("s"),
	)
	ucCalls, _ = Meter.Int64Counter(
		"slotify.usecase.calls",
		metric.WithDescription("Total number of use case executions"),
		metric.WithUnit("{call}"),
	)
	ucActive, _ = Meter.Int64UpDownCounter(
		"slotify.usecase.active",
		metric.WithDescription("Number of currently executing (in-flight) use cases"),
		metric.WithUnit("{call}"),
	)
)

func ObserveUseCase(ctx context.Context, domain, usecase string) (context.Context, func(error)) {
	start := time.Now()
	ctx = WithDomainAndUseCase(ctx, domain, usecase)

	activeAttrs := metric.WithAttributes(
		attribute.String("domain", domain),
		attribute.String("usecase", usecase),
	)

	ucActive.Add(ctx, 1, activeAttrs)

	return ctx, func(err error) {
		ucActive.Add(ctx, -1, activeAttrs)

		status := "ok"
		if err != nil {
			status = "error"
		}

		resultAttrs := metric.WithAttributes(
			attribute.String("domain", domain),
			attribute.String("usecase", usecase),
			attribute.String("status", status),
		)

		ucCalls.Add(ctx, 1, resultAttrs)
		ucDuration.Record(ctx, time.Since(start).Seconds(), resultAttrs)
	}
}
