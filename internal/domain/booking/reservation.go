package booking

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/whicu/slotify/internal/domain"
	"github.com/whicu/slotify/internal/domain/space"
	"github.com/whicu/slotify/internal/domain/user"
)

var ErrIDRequired = errors.New("reservation: id is required")

type ReservationID = uuid.UUID

type Reservation struct {
	id        ReservationID
	spaceID   space.SpaceID
	userID    user.UserID
	date      Date
	slot      Slot
	createdAt time.Time
}

func NewReservation(
	id ReservationID,
	spaceID space.SpaceID,
	userID user.UserID,
	date Date,
	slot Slot,
	createdAt time.Time,
) (r *Reservation, err error) {
	defer func() {
		if err != nil {
			err = domain.ErrInvalidArgument(err)
		}
	}()
	if id == uuid.Nil {
		return nil, ErrIDRequired
	}
	if spaceID == uuid.Nil {
		return nil, space.ErrIDRequired
	}
	if userID == uuid.Nil {
		return nil, user.ErrIDRequired
	}
	return &Reservation{
		id: id, spaceID: spaceID, userID: userID,
		date: date, slot: slot, createdAt: createdAt,
	}, nil
}

func (r *Reservation) ID() ReservationID      { return r.id }
func (r *Reservation) SpaceID() space.SpaceID { return r.spaceID }
func (r *Reservation) UserID() user.UserID    { return r.userID }
func (r *Reservation) Date() Date             { return r.date }
func (r *Reservation) Slot() Slot             { return r.slot }
func (r *Reservation) CreatedAt() time.Time   { return r.createdAt }

func Reconstruct(id ReservationID, spaceID space.SpaceID, userID user.UserID, date Date, slot Slot, createdAt time.Time) *Reservation {
	return &Reservation{id: id, spaceID: spaceID, userID: userID, date: date, slot: slot, createdAt: createdAt}
}
