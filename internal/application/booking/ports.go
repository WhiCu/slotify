package booking

import (
	"context"
	"errors"
	"time"

	domainbooking "github.com/whicu/slotify/internal/domain/booking"
	domainspace "github.com/whicu/slotify/internal/domain/space"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

var (
	ErrSpaceNotFound       = errors.New("application/booking: space not found")
	ErrSpaceNotActive      = errors.New("application/booking: space is not active")
	ErrCapacityExceeded    = errors.New("application/booking: space cannot host requested number of people")
	ErrSlotsConflict       = errors.New("application/booking: one or more slots are already booked")
	ErrReservationNotFound = errors.New("application/booking: reservation not found")
	ErrNotOwnerOrAdmin     = errors.New("application/booking: only owner or admin can perform this action")
	ErrNoSlots             = errors.New("application/booking: at least one slot is required")
)

type IDGenerator interface {
	NewID() domainbooking.ReservationID
}

type ReservationRepository interface {
	SaveAll(ctx context.Context, reservations []*domainbooking.Reservation) error
	FindByID(ctx context.Context, id domainbooking.ReservationID) (*domainbooking.Reservation, error)
	ListBySpaceAndDate(ctx context.Context, spaceID domainspace.SpaceID, date time.Time) ([]*domainbooking.Reservation, error)
	ListByUserID(ctx context.Context, userID domainuser.UserID) ([]*domainbooking.Reservation, error)
	DeleteByIDs(ctx context.Context, ids []domainbooking.ReservationID) error
	HasConflict(ctx context.Context, spaceID domainspace.SpaceID, date time.Time, slots []domainbooking.Slot) (bool, error)
}

type SpaceRepository interface {
	FindByID(ctx context.Context, id domainspace.SpaceID) (*domainspace.Space, error)
}

type UserRepository interface {
	FindByID(ctx context.Context, id domainuser.UserID) (*domainuser.User, error)
}

type Transactor interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
