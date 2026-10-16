package booking

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/whicu/slotify/internal/application/meta"
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

	// metrics
	userBookingsRetrieved metric.Int64Histogram
}

func NewListByUser(
	log *slog.Logger,
	reservations ListByUserFinder,
) *ListByUser {
	userBookingsRetrieved, err := meter.Int64Histogram(
		"slotify.booking.user.bookings_count",
		metric.WithDescription("Number of booking groups returned for user bookings list"),
		metric.WithUnit("{booking}"),
	)
	if err != nil {
		panic(err)
	}

	return &ListByUser{
		log:                   log,
		reservations:          reservations,
		userBookingsRetrieved: userBookingsRetrieved,
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
) (output *ListByUserOutput, err error) {
	ctx, endObserve := meta.ObserveUseCase(ctx, "booking", "list_by_user")
	defer func() { endObserve(err) }()

	ctx, span := tracer.Start(ctx, "ListByUser.Execute",
		trace.WithAttributes(
			attribute.String("domain", "booking"),
			attribute.String("usecase", "list_by_user"),
			attribute.String("user_id", userID.String()),
		),
	)
	defer span.End()
	l.log.DebugContext(
		ctx,
		"executing list bookings by user",
		slog.String("user_id", userID.String()),
	)

	all, err := l.reservations.ListByUserID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
	span.SetAttributes(
		attribute.Int("groups.count", len(groups)),
		attribute.Int("slots.count", len(all)),
	)
	span.SetStatus(codes.Ok, "bookings successfully listed by user")

	l.userBookingsRetrieved.Record(ctx, int64(len(groups)))

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
