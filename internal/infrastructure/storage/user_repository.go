package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/whicu/slotify/internal/domain"
	domainuser "github.com/whicu/slotify/internal/domain/user"
	"github.com/whicu/slotify/internal/infrastructure/storage/pg"
)

type UserRepository struct {
	storage *Storage
}

func NewUserRepository(storage *Storage) *UserRepository {
	return &UserRepository{
		storage: storage,
	}
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	return r.storage.GetQueries(ctx).CountUsers(ctx)
}

func (r *UserRepository) LockRootCreation(ctx context.Context) error {
	return r.storage.GetQueries(ctx).LockRootCreation(ctx)
}

func (r *UserRepository) Save(
	ctx context.Context,
	u *domainuser.User,
) error {
	return r.storage.GetQueries(ctx).SaveUser(
		ctx,
		pg.SaveUserParams{
			ID:        u.ID(),
			Role:      pg.UserRole(u.Role().String()),
			CreatedAt: u.CreatedAt(),
		},
	)
}

func (r *UserRepository) FindByID(
	ctx context.Context,
	id domainuser.UserID,
) (*domainuser.User, error) {
	row, err := r.storage.GetQueries(ctx).FindUserByID(
		ctx,
		id,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}

		return nil, err
	}

	return rowToUser(row)
}

func (r *UserRepository) FindByIDForUpdate(
	ctx context.Context,
	id domainuser.UserID,
) (*domainuser.User, error) {
	row, err := r.storage.GetQueries(ctx).FindUserByIDForUpdate(
		ctx,
		id,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}

		return nil, err
	}

	return rowToUser(row)
}

func (r *UserRepository) ListAll(
	ctx context.Context,
) ([]*domainuser.User, error) {
	rows, err := r.storage.GetQueries(ctx).ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	users := make([]*domainuser.User, 0, len(rows))

	for _, row := range rows {
		u, errTo := rowToUser(row)
		if errTo != nil {
			return nil, errTo
		}

		users = append(users, u)
	}

	return users, nil
}

func (r *UserRepository) Delete(
	ctx context.Context,
	id domainuser.UserID,
) error {
	result, err := r.storage.GetQueries(ctx).DeleteUser(ctx, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() != 1 {
		return domain.ErrNotFound
	}

	return nil
}

func rowToUser(row pg.User) (*domainuser.User, error) {
	role, err := domainuser.RoleFromString(
		string(row.Role),
	)
	if err != nil {
		return nil, err
	}

	return domainuser.Reconstruct(
		row.ID,
		role,
		row.CreatedAt,
	), nil
}
