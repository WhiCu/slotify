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

type CreateBooking struct {
	log          *slog.Logger
	ids          IDGenerator
	reservations ReservationRepository
	spaces       SpaceRepository
	transactor   Transactor
}

func NewCreateBooking(
	log *slog.Logger,
	ids IDGenerator,
	reservations ReservationRepository,
	spaces SpaceRepository,
	transactor Transactor,
) *CreateBooking {
	return &CreateBooking{
		log:          log,
		ids:          ids,
		reservations: reservations,
		spaces:       spaces,
		transactor:   transactor,
	}
}

type CreateBookingInput struct {
	SpaceID   domainspace.SpaceID
	UserID    domainuser.UserID
	Date      time.Time
	StartSlot int
	EndSlot   int
	People    int
}

type CreateBookingOutput struct {
	ReservationIDs []domainbooking.ReservationID
	SpaceID        domainspace.SpaceID
	Date           time.Time
	StartSlot      int
	EndSlot        int
}

func (c *CreateBooking) Execute(ctx context.Context, in CreateBookingInput) (*CreateBookingOutput, error) {
	c.log.DebugContext(ctx, "executing create booking",
		slog.String("space_id", in.SpaceID.String()),
		slog.String("user_id", in.UserID.String()),
		slog.Int("start_slot", in.StartSlot),
		slog.Int("end_slot", in.EndSlot),
	)

	if in.StartSlot > in.EndSlot {
		return nil, domain.ErrInvalidArgument(fmt.Errorf("start_slot (%d) must be <= end_slot (%d)", in.StartSlot, in.EndSlot))
	}

	slots, err := buildSlotRange(in.StartSlot, in.EndSlot)
	if err != nil {
		c.log.WarnContext(ctx, "invalid slot range", slog.Any("error", err))
		return nil, err
	}

	if len(slots) == 0 {
		return nil, ErrNoSlots
	}

	var reservationIDs []domainbooking.ReservationID

	err = c.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		if txErr := c.validateSpace(ctx, in.SpaceID, in.People); txErr != nil {
			return txErr
		}

		conflict, txErr := c.reservations.HasConflict(ctx, in.SpaceID, in.Date, slots)
		if txErr != nil {
			c.log.ErrorContext(ctx, "failed to check slot conflicts", slog.Any("error", txErr))
			return fmt.Errorf("check conflicts: %w", txErr)
		}
		if conflict {
			c.log.WarnContext(ctx, "slot conflict detected",
				slog.String("space_id", in.SpaceID.String()),
				slog.Int("start_slot", in.StartSlot),
				slog.Int("end_slot", in.EndSlot),
			)
			return ErrSlotsConflict
		}

		reservationIDs, txErr = c.createAndSaveReservations(ctx, in, slots)
		return txErr
	})

	if err != nil {
		c.log.ErrorContext(ctx, "create booking failed", slog.Any("error", err))
		return nil, err
	}

	c.log.InfoContext(ctx, "booking created",
		slog.String("space_id", in.SpaceID.String()),
		slog.String("user_id", in.UserID.String()),
		slog.Int("slots_count", len(slots)),
	)

	return &CreateBookingOutput{
		ReservationIDs: reservationIDs,
		SpaceID:        in.SpaceID,
		Date:           in.Date,
		StartSlot:      in.StartSlot,
		EndSlot:        in.EndSlot,
	}, nil
}

func (c *CreateBooking) validateSpace(ctx context.Context, spaceID domainspace.SpaceID, people int) error {
	s, err := c.spaces.FindByID(ctx, spaceID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.log.WarnContext(ctx, "space not found", slog.String("space_id", spaceID.String()))
			return ErrSpaceNotFound
		}
		return fmt.Errorf("find space: %w", err)
	}

	if !s.IsActive() {
		c.log.WarnContext(ctx, "space is not active", slog.String("space_id", spaceID.String()))
		return ErrSpaceNotActive
	}

	p := people
	if p == 0 {
		p = 1
	}
	if !s.CanHost(p) {
		c.log.WarnContext(ctx, "space cannot host requested people",
			slog.String("space_id", spaceID.String()),
			slog.Int("people", p),
			slog.Int("capacity", s.Capacity()),
		)
		return ErrCapacityExceeded
	}

	return nil
}

func (c *CreateBooking) createAndSaveReservations(ctx context.Context, in CreateBookingInput, slots []domainbooking.Slot) ([]domainbooking.ReservationID, error) {
	now := time.Now()
	reservations, err := domainbooking.NewReservations(
		in.SpaceID,
		in.UserID,
		in.Date,
		slots,
		func() domainbooking.ReservationID { return c.ids.NewID() },
		now,
	)
	if err != nil {
		c.log.ErrorContext(ctx, "failed to create reservation entities", slog.Any("error", err))
		return nil, err
	}

	if err = c.reservations.SaveAll(ctx, reservations); err != nil {
		c.log.ErrorContext(ctx, "failed to save reservations", slog.Any("error", err))
		return nil, fmt.Errorf("save reservations: %w", err)
	}

	ids := make([]domainbooking.ReservationID, 0, len(reservations))
	for _, r := range reservations {
		ids = append(ids, r.ID())
	}

	return ids, nil
}

func buildSlotRange(start, end int) ([]domainbooking.Slot, error) {
	slots := make([]domainbooking.Slot, 0, end-start+1)
	for i := start; i <= end; i++ {
		s, err := domainbooking.NewSlot(i)
		if err != nil {
			return nil, domain.ErrInvalidArgument(err)
		}
		slots = append(slots, s)
	}
	return slots, nil
}
