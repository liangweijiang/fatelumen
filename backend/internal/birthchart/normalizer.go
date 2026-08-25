package birthchart

import (
	"context"
	"errors"
	"time"

	"github.com/6tail/lunar-go/calendar"
)

var (
	ErrInvalidInput       = errors.New("invalid birth chart input")
	ErrInvalidLocation    = errors.New("invalid birth location")
	ErrInvalidLocalTime   = errors.New("invalid local civil time")
	ErrAmbiguousLocalTime = errors.New("ambiguous local civil time")
)

type DefaultInputNormalizer struct{}

func (DefaultInputNormalizer) Normalize(_ context.Context, input Input) (*NormalizedInput, error) {
	if input.Gender != 0 && input.Gender != 1 || input.CalendarType != 0 && input.CalendarType != 1 ||
		input.Year < 1 || input.Year > 9999 || input.Month < 1 || input.Month > 12 ||
		input.Day < 1 || input.Hour < 0 || input.Hour > 23 || input.Minute < 0 || input.Minute > 59 {
		return nil, ErrInvalidInput
	}

	var civil time.Time
	if input.CalendarType == 0 {
		civil = wallTime(input.Year, time.Month(input.Month), input.Day, input.Hour, input.Minute, 0)
		if civil.Year() != input.Year || int(civil.Month()) != input.Month || civil.Day() != input.Day {
			return nil, ErrInvalidInput
		}
	} else {
		if input.Day > 30 {
			return nil, ErrInvalidInput
		}
		month := input.Month
		if input.IsLeapMonth {
			month = -month
		}
		lunar := calendar.NewLunar(input.Year, month, input.Day, input.Hour, input.Minute, 0)
		solar := lunar.GetSolar()
		if solar == nil {
			return nil, ErrInvalidInput
		}
		civil = wallTime(solar.GetYear(), time.Month(solar.GetMonth()), solar.GetDay(), input.Hour, input.Minute, 0)
	}

	return &NormalizedInput{Original: input, LocalCivilTime: civil}, nil
}

func wallTime(year int, month time.Month, day, hour, minute, second int) time.Time {
	return time.Date(year, month, day, hour, minute, second, 0, time.UTC)
}
