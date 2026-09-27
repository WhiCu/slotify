package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/whicu/slotify/internal/domain"
)

var (
	ErrIDRequired   = errors.New("user: id is required")
	ErrRoleRequired = errors.New("user: role is required")
)

type UserID = uuid.UUID

func NewUserID(bytes []byte) (UserID, error) {
	id, err := uuid.FromBytes(bytes)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

type User struct {
	id        UserID
	role      Role
	createdAt time.Time
}

func New(id UserID, role Role, createdAt time.Time) (u *User, err error) {
	defer func() {
		if err != nil {
			err = domain.ErrInvalidArgument(err)
		}
	}()
	if id == uuid.Nil {
		return nil, ErrIDRequired
	}
	if role == Unknown {
		return nil, ErrRoleRequired
	}
	return &User{
		id:        id,
		role:      role,
		createdAt: createdAt,
	}, nil
}

func NewRoot(id UserID, createdAt time.Time) (u *User, err error) {
	defer func() {
		if err != nil {
			err = domain.ErrInvalidArgument(err)
		}
	}()
	if id == uuid.Nil {
		return nil, ErrIDRequired
	}
	return &User{id: id, role: Admin, createdAt: createdAt}, nil
}

func (u *User) ID() UserID           { return u.id }
func (u *User) Role() Role           { return u.role }
func (u *User) CreatedAt() time.Time { return u.createdAt }

func Reconstruct(id UserID, role Role, createdAt time.Time) *User {
	return &User{id: id, role: role, createdAt: createdAt}
}
