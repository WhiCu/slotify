package space

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/whicu/slotify/internal/domain"
)

var (
	ErrIDRequired         = errors.New("space: id is required")
	ErrNameRequired       = errors.New("space: name is required")
	ErrTypeRequired       = errors.New("space: type is required")
	ErrCapacityInvalid    = errors.New("space: capacity must be positive")
	ErrAlreadyDeactivated = errors.New("space: already deactivated")
	ErrAlreadyActive      = errors.New("space: already active")
)

type SpaceID = uuid.UUID

func NewSpaceID(bytes []byte) (SpaceID, error) {
	id, err := uuid.FromBytes(bytes)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

type Space struct {
	id        SpaceID
	name      string
	spaceType Type
	capacity  int
	active    bool
	createdAt time.Time
}

func New(id SpaceID, name string, spaceType Type, capacity int, createdAt time.Time) (s *Space, err error) {
	defer func() {
		if err != nil {
			err = domain.ErrInvalidArgument(err)
		}
	}()
	if id == uuid.Nil {
		return nil, ErrIDRequired
	}
	if name == "" {
		return nil, ErrNameRequired
	}
	if spaceType == Unknown {
		return nil, ErrTypeRequired
	}
	if capacity <= 0 {
		return nil, ErrCapacityInvalid
	}
	return &Space{
		id:        id,
		name:      name,
		spaceType: spaceType,
		capacity:  capacity,
		active:    true,
		createdAt: createdAt,
	}, nil
}

func (s *Space) ID() SpaceID          { return s.id }
func (s *Space) Name() string         { return s.name }
func (s *Space) Type() Type           { return s.spaceType }
func (s *Space) Capacity() int        { return s.capacity }
func (s *Space) IsActive() bool       { return s.active }
func (s *Space) CreatedAt() time.Time { return s.createdAt }

func (s *Space) Rename(name string) error {
	if name == "" {
		return domain.ErrInvalidArgument(ErrNameRequired)
	}
	s.name = name
	return nil
}

func (s *Space) ChangeCapacity(capacity int) error {
	if capacity <= 0 {
		return domain.ErrInvalidArgument(ErrCapacityInvalid)
	}
	s.capacity = capacity
	return nil
}

func (s *Space) CanHost(people int) bool {
	return s.active && people <= s.capacity
}

func (s *Space) Deactivate() error {
	if !s.active {
		return ErrAlreadyDeactivated
	}
	s.active = false
	return nil
}

func (s *Space) Activate() error {
	if s.active {
		return ErrAlreadyActive
	}
	s.active = true
	return nil
}

func Reconstruct(id SpaceID, name string, spaceType Type, capacity int, active bool, createdAt time.Time) *Space {
	return &Space{id: id, name: name, spaceType: spaceType, capacity: capacity, active: active, createdAt: createdAt}
}
