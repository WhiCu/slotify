package storage

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/whicu/slotify/internal/infrastructure/storage/pg"
	"github.com/whicu/slotify/pkg/errkit"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type txKey struct{}

func (s *Storage) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	ctx, span := tracer.Start(ctx, "db.transaction",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	err := s.runInTx(ctx, fn)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "committed")
	return nil
}

func (s *Storage) runInTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}

	ctxWithTx := context.WithValue(ctx, txKey{}, tx)

	defer func() {
		if p := recover(); p != nil {
			rollbackErr := tx.Rollback(context.WithoutCancel(ctx))
			if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
				s.log.ErrorContext(ctx, "storage transaction rollback error on panic", slog.Any("panic", p), slog.Any("error", rollbackErr))
				err = errors.Join(errkit.NewPanicError(p), rollbackErr)
				return
			}
			s.log.ErrorContext(ctx, "storage transaction panic", slog.Any("panic", p))
			err = errkit.NewPanicError(p)
			return
		}
		if err != nil {
			s.log.DebugContext(ctx, "storage transaction rollback", slog.Any("error", err))
			if rollbackErr := tx.Rollback(context.WithoutCancel(ctx)); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
				err = errors.Join(err, rollbackErr)
			}
		}
	}()

	if err = fn(ctxWithTx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Storage) GetQueries(ctx context.Context) *pg.Queries {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return s.queries.WithTx(tx)
	}
	return s.queries
}
