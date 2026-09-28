package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/whicu/slotify/internal/domain"
	"github.com/whicu/slotify/internal/infrastructure/storage/pg"

	domainbooking "github.com/whicu/slotify/internal/domain/booking"
	domainspace "github.com/whicu/slotify/internal/domain/space"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

type ReservationRepository struct {
	storage *Storage
}

func NewReservationRepository(storage *Storage) *ReservationRepository {
	return &ReservationRepository{
		storage: storage,
	}
}

func (r *ReservationRepository) SaveAll(
	ctx context.Context,
	reservations []*domainbooking.Reservation,
) error {
	if len(reservations) == 0 {
		return nil
	}

	params := make(
		[]pg.SaveReservationsParams,
		0,
		len(reservations),
	)

	for _, reservation := range reservations {
		params = append(
			params,
			pg.SaveReservationsParams{
				ID:        reservation.ID(),
				SpaceID:   reservation.SpaceID(),
				UserID:    reservation.UserID(),
				Date:      dateToTime(reservation.Date()),
				Slot:      reservation.Slot().Int(),
				CreatedAt: reservation.CreatedAt(),
			},
		)
	}

	batch := r.storage.GetQueries(ctx).SaveReservations(
		ctx,
		params,
	)

	var execErr error

	batch.Exec(func(_ int, err error) {
		if err != nil && execErr == nil {
			execErr = err
		}
	})

	closeErr := batch.Close()

	if execErr != nil {
		return execErr
	}

	return closeErr
}

func (r *ReservationRepository) FindByID(
	ctx context.Context,
	id domainbooking.ReservationID,
) (*domainbooking.Reservation, error) {
	row, err := r.storage.GetQueries(ctx).FindReservationByID(
		ctx,
		id,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}

		return nil, err
	}

	return rowToReservation(row)
}

func (r *ReservationRepository) FindByIDForUpdate(
	ctx context.Context,
	id domainbooking.ReservationID,
) (*domainbooking.Reservation, error) {
	row, err := r.storage.GetQueries(ctx).FindReservationByIDForUpdate(
		ctx,
		id,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}

		return nil, err
	}

	return rowToReservation(row)
}

func (r *ReservationRepository) HasConflict(
	ctx context.Context,
	spaceID domainspace.SpaceID,
	date domainbooking.Date,
	slots []domainbooking.Slot,
) (bool, error) {
	dbSlots := make([]int, 0, len(slots))

	for _, slot := range slots {
		dbSlots = append(
			dbSlots,
			slot.Int(),
		)
	}

	conflict, err := r.storage.GetQueries(ctx).HasReservationConflict(
		ctx,
		pg.HasReservationConflictParams{
			SpaceID: spaceID,
			Date:    dateToTime(date),
			Slots:   dbSlots,
		},
	)
	if err != nil {
		return false, err
	}

	return conflict, nil
}

func (r *ReservationRepository) ListBySpaceAndDate(
	ctx context.Context,
	spaceID domainspace.SpaceID,
	date domainbooking.Date,
) ([]*domainbooking.Reservation, error) {
	rows, err := r.storage.GetQueries(ctx).ListReservationsBySpaceAndDate(
		ctx,
		pg.ListReservationsBySpaceAndDateParams{
			SpaceID: spaceID,
			Date:    dateToTime(date),
		},
	)
	if err != nil {
		return nil, err
	}

	reservations := make(
		[]*domainbooking.Reservation,
		0,
		len(rows),
	)

	for _, row := range rows {
		reservation, errTo := rowToReservation(row)
		if errTo != nil {
			return nil, errTo
		}

		reservations = append(
			reservations,
			reservation,
		)
	}

	return reservations, nil
}

func (r *ReservationRepository) ListByUserID(
	ctx context.Context,
	userID domainuser.UserID,
) ([]*domainbooking.Reservation, error) {
	rows, err := r.storage.GetQueries(ctx).ListReservationsByUserID(
		ctx,
		userID,
	)
	if err != nil {
		return nil, err
	}

	reservations := make(
		[]*domainbooking.Reservation,
		0,
		len(rows),
	)

	for _, row := range rows {
		reservation, errTo := rowToReservation(row)
		if errTo != nil {
			return nil, errTo
		}

		reservations = append(
			reservations,
			reservation,
		)
	}

	return reservations, nil
}

func (r *ReservationRepository) DeleteByIDs(
	ctx context.Context,
	ids []domainbooking.ReservationID,
) error {
	if len(ids) == 0 {
		return nil
	}

	return r.storage.GetQueries(ctx).DeleteReservationsByIDs(
		ctx,
		ids,
	)
}

func (r *ReservationRepository) DeleteByUserID(
	ctx context.Context,
	userID domainuser.UserID,
) error {
	return r.storage.GetQueries(ctx).DeleteReservationsByUserID(
		ctx,
		userID,
	)
}

func rowToReservation(
	row pg.Reservation,
) (*domainbooking.Reservation, error) {
	slot, err := domainbooking.NewSlot(
		row.Slot,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid reservation slot %d: %w",
			row.Slot,
			err,
		)
	}

	date := domainbooking.FromTime(row.Date)

	return domainbooking.Reconstruct(
		row.ID,
		row.SpaceID,
		row.UserID,
		date,
		slot,
		row.CreatedAt,
	), nil
}

func dateToTime(d domainbooking.Date) time.Time {
	return time.Date(
		d.Year(),
		d.Month(),
		d.Day(),
		0,
		0,
		0,
		0,
		time.UTC,
	)
}
