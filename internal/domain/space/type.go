package space

import "errors"

var ErrUnknownType = errors.New("space: unknown type")

type Type struct {
	slug string
}

func (t Type) String() string { return t.slug }

var (
	Unknown     = Type{""}
	MeetingRoom = Type{"meeting_room"}
	Desk        = Type{"desk"}
)

func TypeFromString(s string) (Type, error) {
	switch s {
	case MeetingRoom.slug:
		return MeetingRoom, nil
	case Desk.slug:
		return Desk, nil
	}
	return Unknown, ErrUnknownType
}
