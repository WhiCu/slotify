package booking

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	domainbooking "github.com/whicu/slotify/internal/domain/booking"
	domainspace "github.com/whicu/slotify/internal/domain/space"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

type ListByUserFinder interface {
	ListByUserID(
		ctx context.Context,
		userID domainuser.UserID,
	) ([]*domainbooking.Reservation, error)
}

type ListByUser struct {
	log          *slog.Logger
	reservations ListByUserFinder
}

func NewListByUser(
	log *slog.Logger,
	reservations ListByUserFinder,
) *ListByUser {
	return &ListByUser{
		log:          log,
		reservations: reservations,
	}
}

type BookingGroup struct {
	SpaceID   domainspace.SpaceID
	Date      domainbooking.Date
	StartSlot domainbooking.Slot
	EndSlot   domainbooking.Slot
	IDs       []domainbooking.ReservationID
}

type ListByUserOutput struct {
	Bookings []BookingGroup
}

func (l *ListByUser) Execute(
	ctx context.Context,
	userID domainuser.UserID,
) (*ListByUserOutput, error) {
	l.log.DebugContext(
		ctx,
		"executing list bookings by user",
		slog.String("user_id", userID.String()),
	)

	all, err := l.reservations.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list reservations: %w", err)
	}

	groups := aggregateReservations(all)

	l.log.InfoContext(
		ctx,
		"bookings listed by user",
		slog.String("user_id", userID.String()),
		slog.Int("groups", len(groups)),
		slog.Int("total_slots", len(all)),
	)

	return &ListByUserOutput{
		Bookings: groups,
	}, nil
}

func aggregateReservations(
	reservations []*domainbooking.Reservation,
) []BookingGroup {
	type groupKey struct {
		spaceID domainspace.SpaceID
		date    domainbooking.Date
	}

	grouped := make(map[groupKey][]*domainbooking.Reservation)

	for _, r := range reservations {
		key := groupKey{
			spaceID: r.SpaceID(),
			date:    r.Date(),
		}

		grouped[key] = append(grouped[key], r)
	}

	groups := make([]BookingGroup, 0, len(grouped))

	for key, reservations := range grouped {
		sort.Slice(
			reservations,
			func(i, j int) bool {
				return reservations[i].Slot() < reservations[j].Slot()
			},
		)

		for _, rng := range splitIntoContiguousRanges(reservations) {
			ids := make(
				[]domainbooking.ReservationID,
				0,
				len(rng),
			)

			for _, r := range rng {
				ids = append(ids, r.ID())
			}

			groups = append(groups, BookingGroup{
				SpaceID:   key.spaceID,
				Date:      key.date,
				StartSlot: rng[0].Slot(),
				EndSlot:   rng[len(rng)-1].Slot(),
				IDs:       ids,
			})
		}
	}

	sort.Slice(
		groups,
		func(i, j int) bool {
			di := groups[i].Date.Format(time.DateOnly)
			dj := groups[j].Date.Format(time.DateOnly)

			if di != dj {
				return di < dj
			}

			if groups[i].SpaceID != groups[j].SpaceID {
				return groups[i].SpaceID.String() < groups[j].SpaceID.String()
			}

			return groups[i].StartSlot < groups[j].StartSlot
		},
	)

	return groups
}

func splitIntoContiguousRanges(
	sorted []*domainbooking.Reservation,
) [][]*domainbooking.Reservation {
	if len(sorted) == 0 {
		return nil
	}

	ranges := make([][]*domainbooking.Reservation, 0)

	current := []*domainbooking.Reservation{
		sorted[0],
	}

	for i := 1; i < len(sorted); i++ {
		prev := sorted[i-1].Slot()
		curr := sorted[i].Slot()

		if curr == prev+1 {
			current = append(current, sorted[i])
			continue
		}

		ranges = append(ranges, current)
		current = []*domainbooking.Reservation{
			sorted[i],
		}
	}

	ranges = append(ranges, current)

	return ranges
}
