package booking

import (
	"errors"
	"fmt"
	"time"
)

var ErrInvalid = errors.New("date: invalid date")

type Date struct {
	year  int
	month time.Month
	day   int
}

func New(year int, month time.Month, day int) (Date, error) {
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

	if t.Year() != year || t.Month() != month || t.Day() != day {
		return Date{}, ErrInvalid
	}

	return Date{
		year:  year,
		month: month,
		day:   day,
	}, nil
}

func FromTime(t time.Time) Date {
	return Date{
		year:  t.Year(),
		month: t.Month(),
		day:   t.Day(),
	}
}

func (d Date) Year() int {
	return d.year
}

func (d Date) Month() time.Month {
	return d.month
}

func (d Date) Day() int {
	return d.day
}

func (d Date) Format(layout string) string {
	t := time.Date(d.year, d.month, d.day, 0, 0, 0, 0, time.UTC)
	return t.Format(layout)
}

func (d Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.year, d.month, d.day)
}
