package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/whicu/slotify/internal/domain"
	"github.com/whicu/slotify/internal/infrastructure/storage/pg"

	domainspace "github.com/whicu/slotify/internal/domain/space"
)

type SpaceRepository struct {
	storage *Storage
}

func NewSpaceRepository(storage *Storage) *SpaceRepository {
	return &SpaceRepository{
		storage: storage,
	}
}

func (r *SpaceRepository) Save(
	ctx context.Context,
	s *domainspace.Space,
) error {
	return r.storage.GetQueries(ctx).SaveSpace(
		ctx,
		pg.SaveSpaceParams{
			ID:        s.ID(),
			Name:      s.Name(),
			Type:      pg.SpaceType(s.Type().String()),
			Capacity:  s.Capacity(),
			Active:    s.IsActive(),
			CreatedAt: s.CreatedAt(),
		},
	)
}

func (r *SpaceRepository) FindByID(
	ctx context.Context,
	id domainspace.SpaceID,
) (*domainspace.Space, error) {
	row, err := r.storage.GetQueries(ctx).FindSpaceByID(
		ctx,
		id,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}

		return nil, err
	}

	return rowToSpace(row)
}

func (r *SpaceRepository) FindByIDForUpdate(
	ctx context.Context,
	id domainspace.SpaceID,
) (*domainspace.Space, error) {
	row, err := r.storage.GetQueries(ctx).FindSpaceByIDForUpdate(
		ctx,
		id,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}

		return nil, err
	}

	return rowToSpace(row)
}

func (r *SpaceRepository) List(
	ctx context.Context,
	onlyActive bool,
) ([]*domainspace.Space, error) {
	rows, err := r.storage.GetQueries(ctx).ListSpaces(
		ctx,
		onlyActive,
	)
	if err != nil {
		return nil, err
	}

	spaces := make([]*domainspace.Space, 0, len(rows))

	for _, row := range rows {
		s, errTo := rowToSpace(row)
		if errTo != nil {
			return nil, errTo
		}

		spaces = append(spaces, s)
	}

	return spaces, nil
}

func (r *SpaceRepository) Delete(
	ctx context.Context,
	id domainspace.SpaceID,
) error {
	return r.storage.GetQueries(ctx).DeleteSpace(
		ctx,
		id,
	)
}

func rowToSpace(row pg.Space) (*domainspace.Space, error) {
	spaceType, err := domainspace.TypeFromString(
		string(row.Type),
	)
	if err != nil {
		return nil, err
	}

	return domainspace.Reconstruct(
		row.ID,
		row.Name,
		spaceType,
		row.Capacity,
		row.Active,
		row.CreatedAt,
	), nil
}
