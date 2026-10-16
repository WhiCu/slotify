package booking

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/whicu/slotify/internal/application/meta"
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

	// metrics
	occupiedSlotsRetrieved metric.Int64Histogram
}

func NewListBySpaceAndDate(
	log *slog.Logger,
	reservations ListBySpaceAndDateFinder,
) *ListBySpaceAndDate {
	occupiedSlotsRetrieved, err := meter.Int64Histogram(
		"slotify.booking.space_date.occupied_slots",
		metric.WithDescription("Distribution of occupied slots returned per space and date query"),
		metric.WithUnit("{slot}"),
	)
	if err != nil {
		panic(err)
	}

	return &ListBySpaceAndDate{
		log:                    log,
		reservations:           reservations,
		occupiedSlotsRetrieved: occupiedSlotsRetrieved,
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
) (output *ListBySpaceAndDateOutput, err error) {
	ctx, endObserve := meta.ObserveUseCase(ctx, "booking", "list_by_space_and_date")
	defer func() { endObserve(err) }()

	ctx, span := tracer.Start(ctx, "ListBySpaceAndDate.Execute",
		trace.WithAttributes(
			attribute.String("domain", "booking"),
			attribute.String("usecase", "list_by_space_and_date"),
			attribute.String("space_id", spaceID.String()),
			attribute.String("date", date.String()),
		),
	)
	defer span.End()
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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
	span.SetAttributes(attribute.Int("occupied.count", len(occupied)))
	span.SetStatus(codes.Ok, "bookings successfully listed by space and date")

	l.occupiedSlotsRetrieved.Record(ctx, int64(len(occupied)))

	return &ListBySpaceAndDateOutput{
		SpaceID:  spaceID,
		Date:     date,
		Occupied: occupied,
	}, nil
}
