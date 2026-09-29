package http

import (
	"errors"

	"github.com/whicu/slotify/internal/domain"
	domainspace "github.com/whicu/slotify/internal/domain/space"
	domainuser "github.com/whicu/slotify/internal/domain/user"

	bookingapp "github.com/whicu/slotify/internal/application/booking"
	spaceapp "github.com/whicu/slotify/internal/application/space"
	userapp "github.com/whicu/slotify/internal/application/user"
)

type errKind int

const (
	errBadRequest errKind = iota
	errUnauthorized
	errForbidden
	errNotFound
	errConflict
	errUnprocessable
	errInternal
)

func classifyError(err error) errKind {
	switch {
	// domain validation
	case errors.Is(err, domain.ErrValidation):
		return errBadRequest
	case errors.Is(err, domainuser.ErrSameRole),
		errors.Is(err, domainuser.ErrCannotChangeOwnRole):
		return errBadRequest

	// auth / authz
	case errors.Is(err, ErrUnauthenticated):
		return errUnauthorized
	case errors.Is(err, ErrForbidden),
		errors.Is(err, userapp.ErrNotAdmin),
		errors.Is(err, spaceapp.ErrNotAdmin),
		errors.Is(err, bookingapp.ErrNotOwnerOrAdmin):
		return errForbidden

	// not found
	case errors.Is(err, userapp.ErrUserNotFound),
		errors.Is(err, spaceapp.ErrSpaceNotFound),
		errors.Is(err, spaceapp.ErrUserNotFound),
		errors.Is(err, bookingapp.ErrReservationNotFound),
		errors.Is(err, bookingapp.ErrSpaceNotFound),
		errors.Is(err, bookingapp.ErrUserNotFound):
		return errNotFound

	// conflict
	case errors.Is(err, bookingapp.ErrSlotsConflict),
		errors.Is(err, userapp.ErrCannotDeleteSelf),
		errors.Is(err, userapp.ErrCannotDeleteRoot),
		errors.Is(err, domainspace.ErrAlreadyDeactivated),
		errors.Is(err, domainspace.ErrAlreadyActive):
		return errConflict

	// unprocessable entity
	case errors.Is(err, bookingapp.ErrCapacityExceeded),
		errors.Is(err, bookingapp.ErrSpaceNotActive),
		errors.Is(err, bookingapp.ErrNoSlots),
		errors.Is(err, bookingapp.ErrCannotCancelEmpty):
		return errUnprocessable

	default:
		return errInternal
	}
}
