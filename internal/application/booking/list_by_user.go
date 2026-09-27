package booking

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	domainbooking "github.com/whicu/slotify/internal/domain/booking"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

type ListByUser struct {
	log          *slog.Logger
	reservations ReservationRepository
}

func NewListByUser(
	log *slog.Logger,
	reservations ReservationRepository,
) *ListByUser {
	return &ListByUser{
		log:          log,
		reservations: reservations,
	}
}

type BookingGroup struct {
	SpaceID   string
	Date      string
	StartSlot int
	EndSlot   int
	IDs       []domainbooking.ReservationID
}

type ListByUserOutput struct {
	Bookings []BookingGroup
}

func (l *ListByUser) Execute(ctx context.Context, userID domainuser.UserID) (*ListByUserOutput, error) {
	l.log.DebugContext(ctx, "executing list bookings by user", slog.String("user_id", userID.String()))

	all, err := l.reservations.ListByUserID(ctx, userID)
	if err != nil {
		l.log.ErrorContext(ctx, "failed to list reservations by user", slog.Any("error", err))
		return nil, fmt.Errorf("list reservations: %w", err)
	}

	groups := aggregateReservations(all)

	l.log.InfoContext(ctx, "bookings listed by user",
		slog.String("user_id", userID.String()),
		slog.Int("groups", len(groups)),
		slog.Int("total_slots", len(all)),
	)

	return &ListByUserOutput{Bookings: groups}, nil
}

func aggregateReservations(reservations []*domainbooking.Reservation) []BookingGroup {
	type groupKey struct {
		spaceID string
		date    string
	}

	grouped := make(map[groupKey][]*domainbooking.Reservation)
	for _, r := range reservations {
		key := groupKey{
			spaceID: r.SpaceID().String(),
			date:    r.Date().Format(time.DateOnly),
		}
		grouped[key] = append(grouped[key], r)
	}

	groups := make([]BookingGroup, 0, len(grouped))
	for key, rr := range grouped {
		sort.Slice(rr, func(i, j int) bool {
			return rr[i].Slot().Int() < rr[j].Slot().Int()
		})

		ranges := splitIntoContiguousRanges(rr)
		for _, rng := range ranges {
			ids := make([]domainbooking.ReservationID, 0, len(rng))
			for _, r := range rng {
				ids = append(ids, r.ID())
			}
			groups = append(groups, BookingGroup{
				SpaceID:   key.spaceID,
				Date:      key.date,
				StartSlot: rng[0].Slot().Int(),
				EndSlot:   rng[len(rng)-1].Slot().Int(),
				IDs:       ids,
			})
		}
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Date != groups[j].Date {
			return groups[i].Date < groups[j].Date
		}
		if groups[i].SpaceID != groups[j].SpaceID {
			return groups[i].SpaceID < groups[j].SpaceID
		}
		return groups[i].StartSlot < groups[j].StartSlot
	})

	return groups
}

func splitIntoContiguousRanges(sorted []*domainbooking.Reservation) [][]*domainbooking.Reservation {
	if len(sorted) == 0 {
		return nil
	}

	var ranges [][]*domainbooking.Reservation
	current := []*domainbooking.Reservation{sorted[0]}

	for i := 1; i < len(sorted); i++ {
		if sorted[i].Slot().Int() == sorted[i-1].Slot().Int()+1 {
			current = append(current, sorted[i])
		} else {
			ranges = append(ranges, current)
			current = []*domainbooking.Reservation{sorted[i]}
		}
	}
	ranges = append(ranges, current)

	return ranges
}
