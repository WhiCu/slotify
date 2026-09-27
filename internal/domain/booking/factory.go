package booking

import (
	"time"

	"github.com/whicu/slotify/internal/domain/space"
	"github.com/whicu/slotify/internal/domain/user"
)

func NewReservations(
	spaceID space.SpaceID,
	userID user.UserID,
	date time.Time,
	slots []Slot,
	idFn func() ReservationID,
	now time.Time,
) ([]*Reservation, error) {
	reservations := make([]*Reservation, 0, len(slots))
	for _, slot := range slots {
		res, err := NewReservation(idFn(), spaceID, userID, date, slot, now)
		if err != nil {
			return nil, err
		}
		reservations = append(reservations, res)
	}
	return reservations, nil
}
