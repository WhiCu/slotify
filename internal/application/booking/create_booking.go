package booking

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/whicu/slotify/internal/domain"
	domainbooking "github.com/whicu/slotify/internal/domain/booking"
	domainspace "github.com/whicu/slotify/internal/domain/space"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

var (
	ErrSpaceNotFound    = errors.New("application/booking: space not found")
	ErrSpaceNotActive   = errors.New("application/booking: space is not active")
	ErrCapacityExceeded = errors.New("application/booking: space cannot host requested number of people")
	ErrSlotsConflict    = errors.New("application/booking: one or more slots are already booked")
	ErrUserNotFound     = errors.New("application/booking: user not found")
	ErrNoSlots          = errors.New("application/booking: at least one slot is required")
)

type IDGenerator interface {
	NewID() uuid.UUID
}

type Transactor interface {
	RunInTransaction(ctx context.Context, fn func(context.Context) error) error
}

type CreateBookingUserFinder interface {
	FindByIDForUpdate(
		ctx context.Context,
		id domainuser.UserID,
	) (*domainuser.User, error)
}

type CreateBookingSpaceFinder interface {
	FindByIDForUpdate(
		ctx context.Context,
		id domainspace.SpaceID,
	) (*domainspace.Space, error)
}

type CreateBookingConflictChecker interface {
	HasConflict(
		ctx context.Context,
		spaceID domainspace.SpaceID,
		date domainbooking.Date,
		slots []domainbooking.Slot,
	) (bool, error)
}

type CreateBookingSaver interface {
	SaveAll(
		ctx context.Context,
		reservations []*domainbooking.Reservation,
	) error
}

type CreateBooking struct {
	log          *slog.Logger
	ids          IDGenerator
	users        CreateBookingUserFinder
	spaces       CreateBookingSpaceFinder
	reservations CreateBookingConflictChecker
	saver        CreateBookingSaver
	transactor   Transactor
}

func NewCreateBooking(
	log *slog.Logger,
	ids IDGenerator,
	users CreateBookingUserFinder,
	spaces CreateBookingSpaceFinder,
	reservations CreateBookingConflictChecker,
	saver CreateBookingSaver,
	transactor Transactor,
) *CreateBooking {
	return &CreateBooking{
		log:          log,
		ids:          ids,
		users:        users,
		spaces:       spaces,
		reservations: reservations,
		saver:        saver,
		transactor:   transactor,
	}
}

type CreateBookingInput struct {
	SpaceID   domainspace.SpaceID
	UserID    domainuser.UserID
	Date      domainbooking.Date
	StartSlot int
	EndSlot   int
	People    int
}

type CreateBookingOutput struct {
	ReservationIDs []domainbooking.ReservationID
	SpaceID        domainspace.SpaceID
	Date           domainbooking.Date
	StartSlot      int
	EndSlot        int
}

func (c *CreateBooking) Execute(
	ctx context.Context,
	in CreateBookingInput,
) (*CreateBookingOutput, error) {
	c.log.DebugContext(
		ctx,
		"executing create booking",
		slog.String("space_id", in.SpaceID.String()),
		slog.String("user_id", in.UserID.String()),
		slog.Int("start_slot", in.StartSlot),
		slog.Int("end_slot", in.EndSlot),
	)

	if in.StartSlot > in.EndSlot {
		return nil, domain.ErrInvalidArgument(
			fmt.Errorf("start_slot (%d) must be <= end_slot (%d)", in.StartSlot, in.EndSlot),
		)
	}

	if in.People < 0 {
		return nil, domain.ErrInvalidArgument(errors.New("people must be non-negative"))
	}

	slots, err := buildSlotRange(in.StartSlot, in.EndSlot)
	if err != nil {
		return nil, err
	}

	if len(slots) == 0 {
		return nil, ErrNoSlots
	}

	var reservationIDs []domainbooking.ReservationID

	err = c.transactor.RunInTransaction(ctx, func(txCtx context.Context) error {
		var txErr error
		reservationIDs, txErr = c.executeInTx(txCtx, in, slots)
		return txErr
	})

	if err != nil {
		return nil, err
	}

	c.log.InfoContext(
		ctx,
		"booking created",
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

func (c *CreateBooking) executeInTx(
	ctx context.Context,
	in CreateBookingInput,
	slots []domainbooking.Slot,
) ([]domainbooking.ReservationID, error) {
	if _, err := c.users.FindByIDForUpdate(ctx, in.UserID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	s, err := c.spaces.FindByIDForUpdate(ctx, in.SpaceID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrSpaceNotFound
		}
		return nil, fmt.Errorf("find space: %w", err)
	}

	if !s.IsActive() {
		return nil, ErrSpaceNotActive
	}

	people := in.People
	if people == 0 {
		people = 1
	}

	if !s.CanHost(people) {
		return nil, ErrCapacityExceeded
	}

	conflict, err := c.reservations.HasConflict(ctx, in.SpaceID, in.Date, slots)
	if err != nil {
		return nil, fmt.Errorf("check conflicts: %w", err)
	}

	if conflict {
		return nil, ErrSlotsConflict
	}

	reservations, err := c.createReservations(in, slots)
	if err != nil {
		return nil, err
	}

	if errSaveAll := c.saver.SaveAll(ctx, reservations); errSaveAll != nil {
		return nil, fmt.Errorf("save reservations: %w", errSaveAll)
	}

	ids := make([]domainbooking.ReservationID, 0, len(reservations))
	for _, r := range reservations {
		ids = append(ids, r.ID())
	}

	return ids, nil
}

func (c *CreateBooking) createReservations(
	in CreateBookingInput,
	slots []domainbooking.Slot,
) ([]*domainbooking.Reservation, error) {
	reservations, err := domainbooking.NewReservations(
		in.SpaceID,
		in.UserID,
		in.Date,
		slots,
		func() domainbooking.ReservationID {
			return c.ids.NewID()
		},
		time.Now(),
	)
	if err != nil {
		return nil, err
	}

	return reservations, nil
}

func buildSlotRange(
	start int,
	end int,
) ([]domainbooking.Slot, error) {
	if _, err := domainbooking.NewSlot(start); err != nil {
		return nil, domain.ErrInvalidArgument(err)
	}

	if _, err := domainbooking.NewSlot(end); err != nil {
		return nil, domain.ErrInvalidArgument(err)
	}

	slots := make([]domainbooking.Slot, 0, end-start+1)

	for n := start; n <= end; n++ {
		slots = append(slots, domainbooking.Slot(n))
	}

	return slots, nil
}
