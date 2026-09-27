package booking

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/whicu/slotify/internal/domain/booking"
	domainspace "github.com/whicu/slotify/internal/domain/space"
)

type ListBySpaceAndDate struct {
	log          *slog.Logger
	reservations ReservationRepository
}

func NewListBySpaceAndDate(
	log *slog.Logger,
	reservations ReservationRepository,
) *ListBySpaceAndDate {
	return &ListBySpaceAndDate{
		log:          log,
		reservations: reservations,
	}
}

type SlotOccupancy struct {
	Slot   int
	UserID string
}

type ListBySpaceAndDateOutput struct {
	SpaceID  string
	Date     string
	Occupied []SlotOccupancy
}

func (l *ListBySpaceAndDate) Execute(ctx context.Context, spaceID domainspace.SpaceID, date time.Time) (*ListBySpaceAndDateOutput, error) {
	l.log.DebugContext(ctx, "executing list by space and date",
		slog.String("space_id", spaceID.String()),
		slog.String("date", date.Format(time.DateOnly)),
	)

	all, err := l.reservations.ListBySpaceAndDate(ctx, spaceID, date)
	if err != nil {
		l.log.ErrorContext(ctx, "failed to list reservations by space and date", slog.Any("error", err))
		return nil, fmt.Errorf("list reservations: %w", err)
	}

	slices.SortFunc(all, func(a, b *booking.Reservation) int {
		return a.Slot().Int() - b.Slot().Int()
	})

	occupied := make([]SlotOccupancy, 0, len(all))
	for _, r := range all {
		occupied = append(occupied, SlotOccupancy{
			Slot:   r.Slot().Int(),
			UserID: r.UserID().String(),
		})
	}

	l.log.InfoContext(ctx, "bookings listed by space and date",
		slog.String("space_id", spaceID.String()),
		slog.String("date", date.Format("2006-01-02")),
		slog.Int("occupied_slots", len(occupied)),
	)

	return &ListBySpaceAndDateOutput{
		SpaceID:  spaceID.String(),
		Date:     date.Format(time.DateOnly),
		Occupied: occupied,
	}, nil
}
