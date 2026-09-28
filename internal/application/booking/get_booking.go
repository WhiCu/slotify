package booking

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

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
) (*GetBookingOutput, error) {
	g.log.DebugContext(
		ctx,
		"executing get booking",
		slog.String("reservation_id", id.String()),
	)

	r, err := g.reservations.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrReservationNotFound
		}

		return nil, fmt.Errorf("find reservation: %w", err)
	}

	return &GetBookingOutput{
		ID:        r.ID(),
		SpaceID:   r.SpaceID(),
		UserID:    r.UserID(),
		Date:      r.Date(),
		Slot:      r.Slot(),
		CreatedAt: r.CreatedAt(),
	}, nil
}
