package booking

import (
	"errors"
	"time"
)

var ErrSlotOutOfRange = errors.New("booking: slot is out of range")

const SlotDuration = 5 * time.Minute

const SlotsPerDay = int(24 * time.Hour / SlotDuration)

type Slot int

func NewSlot(n int) (Slot, error) {
	if n < 0 || n >= SlotsPerDay {
		return 0, ErrSlotOutOfRange
	}
	return Slot(n), nil
}

func (s Slot) Int() int { return int(s) }
