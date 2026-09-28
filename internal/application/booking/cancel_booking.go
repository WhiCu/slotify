package booking

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/whicu/slotify/internal/domain"
	domainbooking "github.com/whicu/slotify/internal/domain/booking"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

var (
	ErrReservationNotFound = errors.New(
		"application/booking: reservation not found",
	)
	ErrNotOwnerOrAdmin = errors.New(
		"application/booking: only owner or admin can perform this action",
	)
	ErrCannotCancelEmpty = errors.New(
		"application/booking: at least one reservation is required",
	)
)

type CancelBookingUserFinder interface {
	FindByIDForUpdate(
		ctx context.Context,
		id domainuser.UserID,
	) (*domainuser.User, error)
}

type CancelBookingFinder interface {
	FindByIDForUpdate(
		ctx context.Context,
		id domainbooking.ReservationID,
	) (*domainbooking.Reservation, error)
}

type CancelBookingDeleter interface {
	DeleteByIDs(
		ctx context.Context,
		ids []domainbooking.ReservationID,
	) error
}

type CancelBooking struct {
	log          *slog.Logger
	users        CancelBookingUserFinder
	reservations CancelBookingFinder
	deleter      CancelBookingDeleter
	transactor   Transactor
}

func NewCancelBooking(
	log *slog.Logger,
	users CancelBookingUserFinder,
	reservations CancelBookingFinder,
	deleter CancelBookingDeleter,
	transactor Transactor,
) *CancelBooking {
	return &CancelBooking{
		log:          log,
		users:        users,
		reservations: reservations,
		deleter:      deleter,
		transactor:   transactor,
	}
}

type CancelBookingInput struct {
	ReservationIDs []domainbooking.ReservationID
	ActorID        domainuser.UserID
}

func (c *CancelBooking) Execute(
	ctx context.Context,
	in CancelBookingInput,
) error {
	c.log.DebugContext(
		ctx,
		"executing cancel booking",
		slog.String("actor_id", in.ActorID.String()),
		slog.Int("reservation_count", len(in.ReservationIDs)),
	)

	if len(in.ReservationIDs) == 0 {
		return ErrCannotCancelEmpty
	}

	ids := slices.Clone(in.ReservationIDs)
	slices.SortFunc(ids, func(a, b domainbooking.ReservationID) int {
		return bytes.Compare(a[:], b[:])
	})

	err := c.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		actor, err := c.users.FindByIDForUpdate(ctx, in.ActorID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return ErrNotOwnerOrAdmin
			}

			return fmt.Errorf("find actor: %w", err)
		}

		for _, id := range ids {
			r, errFind := c.reservations.FindByIDForUpdate(ctx, id)
			if errFind != nil {
				if errors.Is(errFind, domain.ErrNotFound) {
					return ErrReservationNotFound
				}

				return fmt.Errorf(
					"find reservation %s: %w",
					id.String(),
					errFind,
				)
			}

			if !actor.IsAdmin() && r.UserID() != actor.ID() {
				return ErrNotOwnerOrAdmin
			}
		}

		if errDelete := c.deleter.DeleteByIDs(ctx, ids); errDelete != nil {
			return fmt.Errorf("delete reservations: %w", errDelete)
		}

		return nil
	})

	if err != nil {
		return err
	}

	c.log.InfoContext(
		ctx,
		"booking cancelled",
		slog.String("actor_id", in.ActorID.String()),
		slog.Int("cancelled_count", len(ids)),
	)

	return nil
}
