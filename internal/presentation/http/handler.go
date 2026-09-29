package http

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	api "github.com/whicu/slotify/api/http"
	bookingapp "github.com/whicu/slotify/internal/application/booking"
	spaceapp "github.com/whicu/slotify/internal/application/space"
	userapp "github.com/whicu/slotify/internal/application/user"
	domainbooking "github.com/whicu/slotify/internal/domain/booking"
)

type Handler struct {
	log *slog.Logger

	// user use cases
	createUser *userapp.CreateUser
	getUser    *userapp.GetUser
	listUsers  *userapp.ListUsers
	updateRole *userapp.UpdateRole
	deleteUser *userapp.DeleteUser

	// space use cases
	createSpace     *spaceapp.CreateSpace
	getSpace        *spaceapp.GetSpace
	listSpaces      *spaceapp.ListSpaces
	updateSpace     *spaceapp.UpdateSpace
	deleteSpace     *spaceapp.DeleteSpace
	deactivateSpace *spaceapp.DeactivateSpace

	// booking use cases
	createBooking   *bookingapp.CreateBooking
	cancelBooking   *bookingapp.CancelBooking
	getBooking      *bookingapp.GetBooking
	listBySpaceDate *bookingapp.ListBySpaceAndDate
	listByUser      *bookingapp.ListByUser
}

func NewHandler(
	log *slog.Logger,
	createUser *userapp.CreateUser,
	getUser *userapp.GetUser,
	listUsers *userapp.ListUsers,
	updateRole *userapp.UpdateRole,
	deleteUser *userapp.DeleteUser,
	createSpace *spaceapp.CreateSpace,
	getSpace *spaceapp.GetSpace,
	listSpaces *spaceapp.ListSpaces,
	updateSpace *spaceapp.UpdateSpace,
	deleteSpace *spaceapp.DeleteSpace,
	deactivateSpace *spaceapp.DeactivateSpace,
	createBooking *bookingapp.CreateBooking,
	cancelBooking *bookingapp.CancelBooking,
	getBooking *bookingapp.GetBooking,
	listBySpaceDate *bookingapp.ListBySpaceAndDate,
	listByUser *bookingapp.ListByUser,
) *Handler {
	return &Handler{
		log:             log,
		createUser:      createUser,
		getUser:         getUser,
		listUsers:       listUsers,
		updateRole:      updateRole,
		deleteUser:      deleteUser,
		createSpace:     createSpace,
		getSpace:        getSpace,
		listSpaces:      listSpaces,
		updateSpace:     updateSpace,
		deleteSpace:     deleteSpace,
		deactivateSpace: deactivateSpace,
		createBooking:   createBooking,
		cancelBooking:   cancelBooking,
		getBooking:      getBooking,
		listBySpaceDate: listBySpaceDate,
		listByUser:      listByUser,
	}
}

var _ api.Handler = (*Handler)(nil)

// HealthCheck implements [api.Handler].
func (h *Handler) HealthCheck(_ context.Context) error {
	return nil
}

// =============================================================================
// Users
// =============================================================================

// CreateUser implements [api.Handler].
func (h *Handler) CreateUser(ctx context.Context, req *api.CreateUserRequest) (api.CreateUserRes, error) {
	out, err := h.createUser.Execute(ctx, userapp.CreateUserInput{
		Role: string(req.Role),
	})
	if err != nil {
		return h.userError(err, "create user")
	}

	return &api.User{
		ID:        out.ID,
		Role:      api.Role(out.Role),
		CreatedAt: out.CreatedAt,
		Token:     out.Token,
	}, nil
}

// GetUser implements [api.Handler].
func (h *Handler) GetUser(ctx context.Context, params api.GetUserParams) (api.GetUserRes, error) {
	out, err := h.getUser.Execute(ctx, params.ID)
	if err != nil {
		return h.getUserError(err, "get user")
	}

	return &api.User{
		ID:        out.ID,
		Role:      api.Role(out.Role),
		CreatedAt: out.CreatedAt,
	}, nil
}

// ListUsers implements [api.Handler].
func (h *Handler) ListUsers(ctx context.Context) (api.ListUsersRes, error) {
	actorID, ok := userIDFromContext(ctx)
	if !ok {
		return &api.ListUsersUnauthorized{Error: "unauthenticated"}, nil
	}

	out, err := h.listUsers.Execute(ctx, actorID)
	if err != nil {
		return h.listUsersError(err, "list users")
	}

	users := make(api.ListUsersOKApplicationJSON, 0, len(out.Users))
	for _, u := range out.Users {
		users = append(users, api.User{
			ID:        u.ID,
			Role:      api.Role(u.Role),
			CreatedAt: u.CreatedAt,
		})
	}

	return &users, nil
}

// UpdateUserRole implements [api.Handler].
func (h *Handler) UpdateUserRole(ctx context.Context, req *api.UpdateRoleRequest, params api.UpdateUserRoleParams) (api.UpdateUserRoleRes, error) {
	actorID, ok := userIDFromContext(ctx)
	if !ok {
		return &api.UpdateUserRoleUnauthorized{Error: "unauthenticated"}, nil
	}

	out, err := h.updateRole.Execute(ctx, userapp.UpdateRoleInput{
		TargetUserID: params.ID,
		NewRole:      string(req.Role),
		ActorID:      actorID,
	})
	if err != nil {
		return h.updateUserRoleError(err, "update user role")
	}

	return &api.User{
		ID:        out.ID,
		Role:      api.Role(out.NewRole),
		CreatedAt: out.CreatedAt,
	}, nil
}

// DeleteUser implements [api.Handler].
func (h *Handler) DeleteUser(ctx context.Context, params api.DeleteUserParams) (api.DeleteUserRes, error) {
	actorID, ok := userIDFromContext(ctx)
	if !ok {
		return &api.DeleteUserUnauthorized{Error: "unauthenticated"}, nil
	}

	err := h.deleteUser.Execute(ctx, userapp.DeleteUserInput{
		TargetUserID: params.ID,
		ActorID:      actorID,
	})
	if err != nil {
		return h.deleteUserError(err, "delete user")
	}

	return &api.DeleteUserNoContent{}, nil
}

// =============================================================================
// Spaces
// =============================================================================

// CreateSpace implements [api.Handler].
func (h *Handler) CreateSpace(ctx context.Context, req *api.CreateSpaceRequest) (api.CreateSpaceRes, error) {
	actorID, ok := userIDFromContext(ctx)
	if !ok {
		return &api.CreateSpaceUnauthorized{Error: "unauthenticated"}, nil
	}

	out, err := h.createSpace.Execute(ctx, spaceapp.CreateSpaceInput{
		Name:     req.Name,
		Type:     string(req.Type),
		Capacity: req.Capacity,
		ActorID:  actorID,
	})
	if err != nil {
		return h.createSpaceError(err, "create space")
	}

	return &api.Space{
		ID:        out.ID,
		Name:      out.Name,
		Type:      api.SpaceType(out.Type),
		Capacity:  out.Capacity,
		Active:    out.Active,
		CreatedAt: out.CreatedAt,
	}, nil
}

// GetSpace implements [api.Handler].
func (h *Handler) GetSpace(ctx context.Context, params api.GetSpaceParams) (api.GetSpaceRes, error) {
	out, err := h.getSpace.Execute(ctx, params.ID)
	if err != nil {
		return h.getSpaceError(err, "get space")
	}

	return &api.Space{
		ID:        out.ID,
		Name:      out.Name,
		Type:      api.SpaceType(out.Type),
		Capacity:  out.Capacity,
		Active:    out.Active,
		CreatedAt: out.CreatedAt,
	}, nil
}

// ListSpaces implements [api.Handler].
func (h *Handler) ListSpaces(
	ctx context.Context,
	params api.ListSpacesParams,
) (api.ListSpacesRes, error) {
	onlyActive := params.OnlyActive.Or(true)

	out, err := h.listSpaces.Execute(ctx, spaceapp.ListSpacesInput{
		OnlyActive: onlyActive,
	})
	if err != nil {
		h.log.ErrorContext(
			ctx,
			"list spaces failed",
			slog.Any("error", err),
		)
		return &api.Error{Error: "internal error"}, nil
	}

	spaces := make(
		api.ListSpacesOKApplicationJSON,
		0,
		len(out.Spaces),
	)

	for _, s := range out.Spaces {
		spaces = append(spaces, api.Space{
			ID:        s.ID,
			Name:      s.Name,
			Type:      api.SpaceType(s.Type),
			Capacity:  s.Capacity,
			Active:    s.Active,
			CreatedAt: s.CreatedAt,
		})
	}

	return &spaces, nil
}

// UpdateSpace implements [api.Handler].
func (h *Handler) UpdateSpace(ctx context.Context, req *api.UpdateSpaceRequest, params api.UpdateSpaceParams) (api.UpdateSpaceRes, error) {
	actorID, ok := userIDFromContext(ctx)
	if !ok {
		return &api.UpdateSpaceUnauthorized{Error: "unauthenticated"}, nil
	}

	input := spaceapp.UpdateSpaceInput{
		SpaceID: params.ID,
		ActorID: actorID,
	}

	if name, nameSet := req.Name.Get(); nameSet {
		input.Name = &name
	}

	if capacity, capSet := req.Capacity.Get(); capSet {
		input.Capacity = &capacity
	}

	out, err := h.updateSpace.Execute(ctx, input)
	if err != nil {
		return h.updateSpaceError(err, "update space")
	}

	return &api.Space{
		ID:        out.ID,
		Name:      out.Name,
		Type:      api.SpaceType(out.Type),
		Capacity:  out.Capacity,
		Active:    out.Active,
		CreatedAt: out.CreatedAt,
	}, nil
}

// DeleteSpace implements [api.Handler].
func (h *Handler) DeleteSpace(ctx context.Context, params api.DeleteSpaceParams) (api.DeleteSpaceRes, error) {
	actorID, ok := userIDFromContext(ctx)
	if !ok {
		return &api.DeleteSpaceUnauthorized{Error: "unauthenticated"}, nil
	}

	err := h.deleteSpace.Execute(ctx, spaceapp.DeleteSpaceInput{
		SpaceID: params.ID,
		ActorID: actorID,
	})
	if err != nil {
		return h.deleteSpaceError(err, "delete space")
	}

	return &api.DeleteSpaceNoContent{}, nil
}

// DeactivateSpace implements [api.Handler].
func (h *Handler) DeactivateSpace(ctx context.Context, params api.DeactivateSpaceParams) (api.DeactivateSpaceRes, error) {
	actorID, ok := userIDFromContext(ctx)
	if !ok {
		return &api.DeactivateSpaceUnauthorized{Error: "unauthenticated"}, nil
	}

	err := h.deactivateSpace.Execute(ctx, spaceapp.DeactivateSpaceInput{
		SpaceID: params.ID,
		ActorID: actorID,
	})
	if err != nil {
		return h.deactivateSpaceError(err, "deactivate space")
	}

	return &api.DeactivateSpaceNoContent{}, nil
}

// =============================================================================
// Bookings
// =============================================================================

// CreateBooking implements [api.Handler].
func (h *Handler) CreateBooking(ctx context.Context, req *api.CreateBookingRequest) (api.CreateBookingRes, error) {
	actorID, ok := userIDFromContext(ctx)
	if !ok {
		return &api.CreateBookingUnauthorized{Error: "unauthenticated"}, nil
	}

	date := domainbooking.FromTime(req.Date)

	people := req.People.Or(1)

	out, err := h.createBooking.Execute(ctx, bookingapp.CreateBookingInput{
		SpaceID:   req.SpaceID,
		UserID:    actorID,
		Date:      date,
		StartSlot: int(req.StartSlot),
		EndSlot:   int(req.EndSlot),
		People:    people,
	})
	if err != nil {
		return h.createBookingError(err, "create booking")
	}

	ids := make([]uuid.UUID, 0, len(out.ReservationIDs))
	ids = append(ids, out.ReservationIDs...)

	return &api.Booking{
		SpaceID:        out.SpaceID,
		Date:           out.Date.Time(),
		StartSlot:      api.Slot(out.StartSlot),
		EndSlot:        api.Slot(out.EndSlot),
		ReservationIds: ids,
	}, nil
}

// CancelBooking implements [api.Handler].
func (h *Handler) CancelBooking(ctx context.Context, req *api.CancelBookingRequest) (api.CancelBookingRes, error) {
	actorID, ok := userIDFromContext(ctx)
	if !ok {
		return &api.CancelBookingUnauthorized{Error: "unauthenticated"}, nil
	}

	err := h.cancelBooking.Execute(ctx, bookingapp.CancelBookingInput{
		ReservationIDs: req.ReservationIds,
		ActorID:        actorID,
	})
	if err != nil {
		return h.cancelBookingError(err, "cancel booking")
	}

	return &api.CancelBookingNoContent{}, nil
}

// ListMyBookings implements [api.Handler].
func (h *Handler) ListMyBookings(ctx context.Context) (api.ListMyBookingsRes, error) {
	actorID, ok := userIDFromContext(ctx)
	if !ok {
		return &api.Error{Error: "unauthenticated"}, nil
	}

	out, err := h.listByUser.Execute(ctx, actorID)
	if err != nil {
		h.log.ErrorContext(ctx, "list my bookings failed", slog.Any("error", err))
		return &api.Error{Error: "internal error"}, nil
	}

	bookings := make(api.ListMyBookingsOKApplicationJSON, 0, len(out.Bookings))
	for _, g := range out.Bookings {
		ids := make([]uuid.UUID, 0, len(g.IDs))
		ids = append(ids, g.IDs...)

		bookings = append(bookings, api.Booking{
			SpaceID:        g.SpaceID,
			Date:           time.Date(g.Date.Year(), g.Date.Month(), g.Date.Day(), 0, 0, 0, 0, time.UTC),
			StartSlot:      api.Slot(g.StartSlot.Int()),
			EndSlot:        api.Slot(g.EndSlot.Int()),
			ReservationIds: ids,
		})
	}

	return &bookings, nil
}

// GetOccupancy implements [api.Handler].
func (h *Handler) GetOccupancy(ctx context.Context, params api.GetOccupancyParams) (api.GetOccupancyRes, error) {
	date := domainbooking.FromTime(params.Date)

	out, err := h.listBySpaceDate.Execute(ctx, params.SpaceID, date)
	if err != nil {
		h.log.ErrorContext(ctx, "get occupancy failed", slog.Any("error", err))
		return &api.GetOccupancyBadRequest{Error: err.Error()}, nil
	}

	occupied := make([]api.SlotOccupancy, 0, len(out.Occupied))
	for _, o := range out.Occupied {
		occupied = append(occupied, api.SlotOccupancy{
			Slot:   api.Slot(o.Slot.Int()),
			UserID: o.UserID,
		})
	}

	return &api.Occupancy{
		SpaceID:  out.SpaceID,
		Date:     out.Date.Time(),
		Occupied: occupied,
	}, nil
}

// GetReservation implements [api.Handler].
func (h *Handler) GetReservation(ctx context.Context, params api.GetReservationParams) (api.GetReservationRes, error) {
	out, err := h.getBooking.Execute(ctx, params.ID)
	if err != nil {
		return h.getReservationError(err, "get reservation")
	}

	return &api.Reservation{
		ID:        out.ID,
		SpaceID:   out.SpaceID,
		UserID:    out.UserID,
		Date:      out.Date.Time(),
		Slot:      api.Slot(out.Slot.Int()),
		CreatedAt: out.CreatedAt,
	}, nil
}

// =============================================================================
// Error mappers — each method maps errKind to the correct ogen response type
// =============================================================================

func (h *Handler) userError(err error, op string) (api.CreateUserRes, error) {
	msg := err.Error()
	switch classifyError(err) {
	case errBadRequest:
		return &api.CreateUserBadRequest{Error: msg}, nil
	case errForbidden:
		return &api.CreateUserForbidden{Error: msg}, nil
	case errUnauthorized:
		return &api.CreateUserUnauthorized{Error: msg}, nil
	default:
		h.log.Error(op+" failed", slog.Any("error", err))
		return nil, err
	}
}

func (h *Handler) getUserError(err error, op string) (api.GetUserRes, error) {
	msg := err.Error()
	switch classifyError(err) {
	case errNotFound:
		return &api.GetUserNotFound{Error: msg}, nil
	case errUnauthorized:
		return &api.GetUserUnauthorized{Error: msg}, nil
	default:
		h.log.Error(op+" failed", slog.Any("error", err))
		return nil, err
	}
}

func (h *Handler) listUsersError(err error, op string) (api.ListUsersRes, error) {
	msg := err.Error()
	switch classifyError(err) {
	case errForbidden:
		return &api.ListUsersForbidden{Error: msg}, nil
	case errUnauthorized:
		return &api.ListUsersUnauthorized{Error: msg}, nil
	default:
		h.log.Error(op+" failed", slog.Any("error", err))
		return nil, err
	}
}

func (h *Handler) updateUserRoleError(err error, op string) (api.UpdateUserRoleRes, error) {
	msg := err.Error()
	switch classifyError(err) {
	case errBadRequest:
		return &api.UpdateUserRoleBadRequest{Error: msg}, nil
	case errForbidden:
		return &api.UpdateUserRoleForbidden{Error: msg}, nil
	case errNotFound:
		return &api.UpdateUserRoleNotFound{Error: msg}, nil
	case errUnauthorized:
		return &api.UpdateUserRoleUnauthorized{Error: msg}, nil
	default:
		h.log.Error(op+" failed", slog.Any("error", err))
		return nil, err
	}
}

func (h *Handler) deleteUserError(err error, op string) (api.DeleteUserRes, error) {
	msg := err.Error()
	switch classifyError(err) {
	case errForbidden:
		return &api.DeleteUserForbidden{Error: msg}, nil
	case errNotFound:
		return &api.DeleteUserNotFound{Error: msg}, nil
	case errConflict:
		return &api.DeleteUserConflict{Error: msg}, nil
	case errUnauthorized:
		return &api.DeleteUserUnauthorized{Error: msg}, nil
	default:
		h.log.Error(op+" failed", slog.Any("error", err))
		return nil, err
	}
}

func (h *Handler) createSpaceError(err error, op string) (api.CreateSpaceRes, error) {
	msg := err.Error()
	switch classifyError(err) {
	case errBadRequest:
		return &api.CreateSpaceBadRequest{Error: msg}, nil
	case errForbidden:
		return &api.CreateSpaceForbidden{Error: msg}, nil
	case errUnauthorized:
		return &api.CreateSpaceUnauthorized{Error: msg}, nil
	default:
		h.log.Error(op+" failed", slog.Any("error", err))
		return nil, err
	}
}

func (h *Handler) getSpaceError(err error, op string) (api.GetSpaceRes, error) {
	msg := err.Error()
	switch classifyError(err) {
	case errNotFound:
		return &api.GetSpaceNotFound{Error: msg}, nil
	case errUnauthorized:
		return &api.GetSpaceUnauthorized{Error: msg}, nil
	default:
		h.log.Error(op+" failed", slog.Any("error", err))
		return nil, err
	}
}

func (h *Handler) updateSpaceError(err error, op string) (api.UpdateSpaceRes, error) {
	msg := err.Error()
	switch classifyError(err) {
	case errBadRequest:
		return &api.UpdateSpaceBadRequest{Error: msg}, nil
	case errForbidden:
		return &api.UpdateSpaceForbidden{Error: msg}, nil
	case errNotFound:
		return &api.UpdateSpaceNotFound{Error: msg}, nil
	case errUnauthorized:
		return &api.UpdateSpaceUnauthorized{Error: msg}, nil
	default:
		h.log.Error(op+" failed", slog.Any("error", err))
		return nil, err
	}
}

func (h *Handler) deleteSpaceError(err error, op string) (api.DeleteSpaceRes, error) {
	msg := err.Error()
	switch classifyError(err) {
	case errForbidden:
		return &api.DeleteSpaceForbidden{Error: msg}, nil
	case errNotFound:
		return &api.DeleteSpaceNotFound{Error: msg}, nil
	case errConflict:
		return &api.DeleteSpaceConflict{Error: msg}, nil
	case errUnauthorized:
		return &api.DeleteSpaceUnauthorized{Error: msg}, nil
	default:
		h.log.Error(op+" failed", slog.Any("error", err))
		return nil, err
	}
}

func (h *Handler) deactivateSpaceError(err error, op string) (api.DeactivateSpaceRes, error) {
	msg := err.Error()
	switch classifyError(err) {
	case errForbidden:
		return &api.DeactivateSpaceForbidden{Error: msg}, nil
	case errNotFound:
		return &api.DeactivateSpaceNotFound{Error: msg}, nil
	case errConflict:
		return &api.DeactivateSpaceConflict{Error: msg}, nil
	case errUnauthorized:
		return &api.DeactivateSpaceUnauthorized{Error: msg}, nil
	default:
		h.log.Error(op+" failed", slog.Any("error", err))
		return nil, err
	}
}

func (h *Handler) createBookingError(err error, op string) (api.CreateBookingRes, error) {
	msg := err.Error()
	switch classifyError(err) {
	case errBadRequest:
		return &api.CreateBookingBadRequest{Error: msg}, nil
	case errNotFound:
		return &api.CreateBookingNotFound{Error: msg}, nil
	case errConflict:
		return &api.CreateBookingConflict{Error: msg}, nil
	case errUnprocessable:
		return &api.CreateBookingUnprocessableEntity{Error: msg}, nil
	case errUnauthorized:
		return &api.CreateBookingUnauthorized{Error: msg}, nil
	default:
		h.log.Error(op+" failed", slog.Any("error", err))
		return nil, err
	}
}

func (h *Handler) cancelBookingError(err error, op string) (api.CancelBookingRes, error) {
	msg := err.Error()
	switch classifyError(err) {
	case errBadRequest:
		return &api.CancelBookingBadRequest{Error: msg}, nil
	case errForbidden:
		return &api.CancelBookingForbidden{Error: msg}, nil
	case errNotFound:
		return &api.CancelBookingNotFound{Error: msg}, nil
	case errUnauthorized:
		return &api.CancelBookingUnauthorized{Error: msg}, nil
	default:
		h.log.Error(op+" failed", slog.Any("error", err))
		return nil, err
	}
}

func (h *Handler) getReservationError(err error, op string) (api.GetReservationRes, error) {
	msg := err.Error()
	switch classifyError(err) {
	case errNotFound:
		return &api.GetReservationNotFound{Error: msg}, nil
	case errUnauthorized:
		return &api.GetReservationUnauthorized{Error: msg}, nil
	default:
		h.log.Error(op+" failed", slog.Any("error", err))
		return nil, err
	}
}
