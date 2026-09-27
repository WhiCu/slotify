package booking

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/whicu/slotify/internal/domain"
	domainbooking "github.com/whicu/slotify/internal/domain/booking"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

type CancelBooking struct {
	log          *slog.Logger
	reservations ReservationRepository
	users        UserRepository
	transactor   Transactor
}

func NewCancelBooking(
	log *slog.Logger,
	reservations ReservationRepository,
	users UserRepository,
	transactor Transactor,
) *CancelBooking {
	return &CancelBooking{
		log:          log,
		reservations: reservations,
		users:        users,
		transactor:   transactor,
	}
}

type CancelBookingInput struct {
	ReservationIDs []domainbooking.ReservationID
	ActorID        domainuser.UserID
}

func (c *CancelBooking) Execute(ctx context.Context, in CancelBookingInput) error {
	c.log.DebugContext(ctx, "executing cancel booking",
		slog.String("actor_id", in.ActorID.String()),
		slog.Int("reservation_count", len(in.ReservationIDs)),
	)

	if len(in.ReservationIDs) == 0 {
		return ErrNoSlots
	}

	err := c.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		actor, txErr := c.users.FindByID(ctx, in.ActorID)
		if txErr != nil {
			if errors.Is(txErr, domain.ErrNotFound) {
				return ErrNotOwnerOrAdmin
			}
			return fmt.Errorf("find actor: %w", txErr)
		}

		for _, rid := range in.ReservationIDs {
			r, errFind := c.reservations.FindByID(ctx, rid)
			if errFind != nil {
				if errors.Is(errFind, domain.ErrNotFound) {
					c.log.WarnContext(ctx, "reservation not found",
						slog.String("reservation_id", rid.String()),
					)
					return ErrReservationNotFound
				}
				return fmt.Errorf("find reservation %s: %w", rid.String(), errFind)
			}

			if r.UserID() != in.ActorID && !actor.IsAdmin() {
				c.log.WarnContext(ctx, "actor is not owner or admin",
					slog.String("actor_id", in.ActorID.String()),
					slog.String("reservation_owner", r.UserID().String()),
				)
				return ErrNotOwnerOrAdmin
			}
		}

		if txErr = c.reservations.DeleteByIDs(ctx, in.ReservationIDs); txErr != nil {
			c.log.ErrorContext(ctx, "failed to delete reservations", slog.Any("error", txErr))
			return fmt.Errorf("delete reservations: %w", txErr)
		}

		return nil
	})

	if err != nil {
		c.log.ErrorContext(ctx, "cancel booking failed", slog.Any("error", err))
		return err
	}

	c.log.InfoContext(ctx, "booking cancelled",
		slog.String("actor_id", in.ActorID.String()),
		slog.Int("cancelled_count", len(in.ReservationIDs)),
	)

	return nil
}
