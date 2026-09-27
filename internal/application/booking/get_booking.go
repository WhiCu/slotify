package booking

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/whicu/slotify/internal/domain"
	domainbooking "github.com/whicu/slotify/internal/domain/booking"
)

type GetBooking struct {
	log          *slog.Logger
	reservations ReservationRepository
}

func NewGetBooking(
	log *slog.Logger,
	reservations ReservationRepository,
) *GetBooking {
	return &GetBooking{
		log:          log,
		reservations: reservations,
	}
}

type GetBookingOutput struct {
	ID        domainbooking.ReservationID
	SpaceID   string
	UserID    string
	Date      string
	Slot      int
	CreatedAt string
}

func (g *GetBooking) Execute(ctx context.Context, id domainbooking.ReservationID) (*GetBookingOutput, error) {
	g.log.DebugContext(ctx, "executing get booking", slog.String("reservation_id", id.String()))

	r, err := g.reservations.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			g.log.WarnContext(ctx, "reservation not found", slog.String("reservation_id", id.String()))
			return nil, ErrReservationNotFound
		}
		g.log.ErrorContext(ctx, "failed to find reservation",
			slog.String("reservation_id", id.String()),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("find reservation: %w", err)
	}

	return &GetBookingOutput{
		ID:        r.ID(),
		SpaceID:   r.SpaceID().String(),
		UserID:    r.UserID().String(),
		Date:      r.Date().Format(time.DateOnly),
		Slot:      r.Slot().Int(),
		CreatedAt: r.CreatedAt().Format(time.RFC3339),
	}, nil
}
