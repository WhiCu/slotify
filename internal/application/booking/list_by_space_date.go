package booking

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	domainbooking "github.com/whicu/slotify/internal/domain/booking"
	domainspace "github.com/whicu/slotify/internal/domain/space"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

type ListBySpaceAndDateFinder interface {
	ListBySpaceAndDate(
		ctx context.Context,
		spaceID domainspace.SpaceID,
		date domainbooking.Date,
	) ([]*domainbooking.Reservation, error)
}

type ListBySpaceAndDate struct {
	log          *slog.Logger
	reservations ListBySpaceAndDateFinder
}

func NewListBySpaceAndDate(
	log *slog.Logger,
	reservations ListBySpaceAndDateFinder,
) *ListBySpaceAndDate {
	return &ListBySpaceAndDate{
		log:          log,
		reservations: reservations,
	}
}

type SlotOccupancy struct {
	Slot   domainbooking.Slot
	UserID domainuser.UserID
}

type ListBySpaceAndDateOutput struct {
	SpaceID  domainspace.SpaceID
	Date     domainbooking.Date
	Occupied []SlotOccupancy
}

func (l *ListBySpaceAndDate) Execute(
	ctx context.Context,
	spaceID domainspace.SpaceID,
	date domainbooking.Date,
) (*ListBySpaceAndDateOutput, error) {
	l.log.DebugContext(
		ctx,
		"executing list by space and date",
		slog.String("space_id", spaceID.String()),
	)

	all, err := l.reservations.ListBySpaceAndDate(
		ctx,
		spaceID,
		date,
	)
	if err != nil {
		return nil, fmt.Errorf("list reservations: %w", err)
	}

	slices.SortFunc(all, func(a, b *domainbooking.Reservation) int {
		return a.Slot().Int() - b.Slot().Int()
	})

	occupied := make([]SlotOccupancy, 0, len(all))

	for _, r := range all {
		occupied = append(occupied, SlotOccupancy{
			Slot:   r.Slot(),
			UserID: r.UserID(),
		})
	}

	l.log.InfoContext(
		ctx,
		"bookings listed by space and date",
		slog.String("space_id", spaceID.String()),
		slog.Int("occupied_slots", len(occupied)),
	)

	return &ListBySpaceAndDateOutput{
		SpaceID:  spaceID,
		Date:     date,
		Occupied: occupied,
	}, nil
}
