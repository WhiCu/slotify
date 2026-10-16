package booking

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/whicu/slotify/internal/application/meta"
	"github.com/whicu/slotify/internal/domain"
	domainbooking "github.com/whicu/slotify/internal/domain/booking"
	domainspace "github.com/whicu/slotify/internal/domain/space"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

type GetBookingFinder interface {
	FindByID(
		ctx context.Context,
		id domainbooking.ReservationID,
	) (*domainbooking.Reservation, error)
}

type GetBooking struct {
	log          *slog.Logger
	reservations GetBookingFinder
}

func NewGetBooking(
	log *slog.Logger,
	reservations GetBookingFinder,
) *GetBooking {
	return &GetBooking{
		log:          log,
		reservations: reservations,
	}
}

type GetBookingOutput struct {
	ID        domainbooking.ReservationID
	SpaceID   domainspace.SpaceID
	UserID    domainuser.UserID
	Date      domainbooking.Date
	Slot      domainbooking.Slot
	CreatedAt time.Time
}

func (g *GetBooking) Execute(
	ctx context.Context,
	id domainbooking.ReservationID,
) (output *GetBookingOutput, err error) {
	ctx, endObserve := meta.ObserveUseCase(ctx, "booking", "get")
	defer func() { endObserve(err) }()

	ctx, span := tracer.Start(ctx, "GetBooking.Execute",
		trace.WithAttributes(
			attribute.String("domain", "booking"),
			attribute.String("usecase", "get"),
			attribute.String("reservation_id", id.String()),
		),
	)
	defer span.End()
	g.log.DebugContext(
		ctx,
		"executing get booking",
		slog.String("reservation_id", id.String()),
	)

	r, err := g.reservations.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			span.RecordError(ErrReservationNotFound)
			span.SetStatus(codes.Error, ErrReservationNotFound.Error())
			return nil, ErrReservationNotFound
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("find reservation: %w", err)
	}

	span.SetAttributes(
		attribute.String("space_id", r.SpaceID().String()),
		attribute.String("user_id", r.UserID().String()),
		attribute.String("date", r.Date().String()),
		attribute.Int("slot", r.Slot().Int()),
	)
	span.SetStatus(codes.Ok, "booking successfully retrieved")

	return &GetBookingOutput{
		ID:        r.ID(),
		SpaceID:   r.SpaceID(),
		UserID:    r.UserID(),
		Date:      r.Date(),
		Slot:      r.Slot(),
		CreatedAt: r.CreatedAt(),
	}, nil
}
