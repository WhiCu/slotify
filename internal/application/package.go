package application

import (
	"log/slog"

	"github.com/knadh/koanf/v2"
	"github.com/samber/do/v2"
	"github.com/whicu/slotify/internal/application/booking"
	"github.com/whicu/slotify/internal/application/space"
	"github.com/whicu/slotify/internal/application/user"
	"github.com/whicu/slotify/internal/config"
)

func newConfig(i do.Injector) (Config, error) {
	k, err := do.Invoke[*koanf.Koanf](i)
	if err != nil {
		return Config{}, err
	}
	def := defaultCfg
	return config.GetConfig(k, "app", &def)
}

// =============================================================================
// Booking Use Cases
// =============================================================================

func newCancelBooking(i do.Injector) (*booking.CancelBooking, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	users, err := do.InvokeAs[booking.CancelBookingUserFinder](i)
	if err != nil {
		return nil, err
	}

	reservations, err := do.InvokeAs[booking.CancelBookingFinder](i)
	if err != nil {
		return nil, err
	}

	deleter, err := do.InvokeAs[booking.CancelBookingDeleter](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[booking.Transactor](i)
	if err != nil {
		return nil, err
	}

	return booking.NewCancelBooking(log, users, reservations, deleter, transactor), nil
}

func newCreateBooking(i do.Injector) (*booking.CreateBooking, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	ids, err := do.InvokeAs[booking.IDGenerator](i)
	if err != nil {
		return nil, err
	}
	users, err := do.InvokeAs[booking.CreateBookingUserFinder](i)
	if err != nil {
		return nil, err
	}

	spaces, err := do.InvokeAs[booking.CreateBookingSpaceFinder](i)
	if err != nil {
		return nil, err
	}

	reservations, err := do.InvokeAs[booking.CreateBookingConflictChecker](i)
	if err != nil {
		return nil, err
	}

	saver, err := do.InvokeAs[booking.CreateBookingSaver](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[booking.Transactor](i)
	if err != nil {
		return nil, err
	}

	return booking.NewCreateBooking(log, ids, users, spaces, reservations, saver, transactor), nil
}

func newGetBooking(i do.Injector) (*booking.GetBooking, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	reservations, err := do.InvokeAs[booking.GetBookingFinder](i)
	if err != nil {
		return nil, err
	}

	return booking.NewGetBooking(log, reservations), nil
}

func newListBySpaceAndDate(i do.Injector) (*booking.ListBySpaceAndDate, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	reservations, err := do.InvokeAs[booking.ListBySpaceAndDateFinder](i)
	if err != nil {
		return nil, err
	}

	return booking.NewListBySpaceAndDate(log, reservations), nil
}

func newListByUser(i do.Injector) (*booking.ListByUser, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	reservations, err := do.InvokeAs[booking.ListByUserFinder](i)
	if err != nil {
		return nil, err
	}

	return booking.NewListByUser(log, reservations), nil
}

// =============================================================================
// Space Use Cases
// =============================================================================

func newCreateSpace(i do.Injector) (*space.CreateSpace, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	ids, err := do.InvokeAs[space.IDGenerator](i)
	if err != nil {
		return nil, err
	}
	users, err := do.InvokeAs[space.CreateSpaceUserFinder](i)
	if err != nil {
		return nil, err
	}

	spaces, err := do.InvokeAs[space.CreateSpaceSaver](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[space.Transactor](i)
	if err != nil {
		return nil, err
	}

	return space.NewCreateSpace(log, ids, users, spaces, transactor), nil
}

func newGetSpace(i do.Injector) (*space.GetSpace, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	spaces, err := do.InvokeAs[space.GetSpaceFinder](i)
	if err != nil {
		return nil, err
	}

	return space.NewGetSpace(log, spaces), nil
}

func newListSpaces(i do.Injector) (*space.ListSpaces, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	spaces, err := do.InvokeAs[space.ListSpacesLister](i)
	if err != nil {
		return nil, err
	}

	return space.NewListSpaces(log, spaces), nil
}

func newUpdateSpace(i do.Injector) (*space.UpdateSpace, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	users, err := do.InvokeAs[space.UpdateSpaceUserFinder](i)
	if err != nil {
		return nil, err
	}

	spaces, err := do.InvokeAs[space.UpdateSpaceFinder](i)
	if err != nil {
		return nil, err
	}

	saver, err := do.InvokeAs[space.UpdateSpaceSaver](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[space.Transactor](i)
	if err != nil {
		return nil, err
	}

	return space.NewUpdateSpace(log, users, spaces, saver, transactor), nil
}

func newDeleteSpace(i do.Injector) (*space.DeleteSpace, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	users, err := do.InvokeAs[space.DeleteSpaceUserFinder](i)
	if err != nil {
		return nil, err
	}

	spaces, err := do.InvokeAs[space.DeleteSpaceFinder](i)
	if err != nil {
		return nil, err
	}

	deleter, err := do.InvokeAs[space.DeleteSpaceDeleter](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[space.Transactor](i)
	if err != nil {
		return nil, err
	}

	return space.NewDeleteSpace(log, users, spaces, deleter, transactor), nil
}

func newDeactivateSpace(i do.Injector) (*space.DeactivateSpace, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	users, err := do.InvokeAs[space.DeactivateSpaceUserFinder](i)
	if err != nil {
		return nil, err
	}

	spaces, err := do.InvokeAs[space.DeactivateSpaceFinder](i)
	if err != nil {
		return nil, err
	}

	saver, err := do.InvokeAs[space.DeactivateSpaceSaver](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[space.Transactor](i)
	if err != nil {
		return nil, err
	}

	return space.NewDeactivateSpace(log, users, spaces, saver, transactor), nil
}

// =============================================================================
// User Use Cases
// =============================================================================

func newCreateUser(i do.Injector) (*user.CreateUser, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, err
	}

	tokenIssuer, err := do.InvokeAs[user.TokenIssuer](i)
	if err != nil {
		return nil, err
	}

	ids, err := do.InvokeAs[user.IDGenerator](i)
	if err != nil {
		return nil, err
	}

	counter, err := do.InvokeAs[user.CreateUserCounter](i)
	if err != nil {
		return nil, err
	}

	saver, err := do.InvokeAs[user.CreateUserSaver](i)
	if err != nil {
		return nil, err
	}

	rootLocker, err := do.InvokeAs[user.CreateUserRootLocker](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[user.Transactor](i)
	if err != nil {
		return nil, err
	}

	return user.NewCreateUser(log, ids, counter, saver, rootLocker, transactor, tokenIssuer, cfg.User.TTL), nil
}

func newGetUser(i do.Injector) (*user.GetUser, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	users, err := do.InvokeAs[user.GetUserFinder](i)
	if err != nil {
		return nil, err
	}

	return user.NewGetUser(log, users), nil
}

func newListUsers(i do.Injector) (*user.ListUsers, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	users, err := do.InvokeAs[user.ListUsersFinder](i)
	if err != nil {
		return nil, err
	}

	lister, err := do.InvokeAs[user.ListUsersLister](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[user.Transactor](i)
	if err != nil {
		return nil, err
	}

	return user.NewListUsers(log, users, lister, transactor), nil
}

func newUpdateRole(i do.Injector) (*user.UpdateRole, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	users, err := do.InvokeAs[user.UpdateRoleFinder](i)
	if err != nil {
		return nil, err
	}

	saver, err := do.InvokeAs[user.UpdateRoleSaver](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[user.Transactor](i)
	if err != nil {
		return nil, err
	}

	return user.NewUpdateRole(log, users, saver, transactor), nil
}

func newDeleteUser(i do.Injector) (*user.DeleteUser, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	users, err := do.InvokeAs[user.DeleteUserFinder](i)
	if err != nil {
		return nil, err
	}

	deleter, err := do.InvokeAs[user.DeleteUserDeleter](i)
	if err != nil {
		return nil, err
	}

	reservations, err := do.InvokeAs[user.DeleteUserReservationCanceller](i)
	if err != nil {
		return nil, err
	}

	transactor, err := do.InvokeAs[user.Transactor](i)
	if err != nil {
		return nil, err
	}

	return user.NewDeleteUser(log, users, deleter, reservations, transactor), nil
}

var Package = do.Package(
	do.Lazy(newConfig),

	// booking
	do.Lazy(newCancelBooking),
	do.Lazy(newCreateBooking),
	do.Lazy(newGetBooking),
	do.Lazy(newListBySpaceAndDate),
	do.Lazy(newListByUser),

	// space
	do.Lazy(newCreateSpace),
	do.Lazy(newGetSpace),
	do.Lazy(newListSpaces),
	do.Lazy(newUpdateSpace),
	do.Lazy(newDeleteSpace),
	do.Lazy(newDeactivateSpace),

	// user
	do.Lazy(newCreateUser),
	do.Lazy(newGetUser),
	do.Lazy(newListUsers),
	do.Lazy(newUpdateRole),
	do.Lazy(newDeleteUser),
)
