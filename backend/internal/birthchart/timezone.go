package birthchart

import (
	"context"
	"time"
)

type IANATimezoneResolver struct{}

func (IANATimezoneResolver) Resolve(_ context.Context, localCivilTime time.Time, timezoneID string) (*TimezoneResult, error) {
	loc, err := time.LoadLocation(timezoneID)
	if err != nil {
		return nil, ErrInvalidLocation
	}
	localized := time.Date(localCivilTime.Year(), localCivilTime.Month(), localCivilTime.Day(), localCivilTime.Hour(), localCivilTime.Minute(), localCivilTime.Second(), 0, loc)
	if localized.Year() != localCivilTime.Year() || localized.Month() != localCivilTime.Month() || localized.Day() != localCivilTime.Day() || localized.Hour() != localCivilTime.Hour() || localized.Minute() != localCivilTime.Minute() {
		return nil, ErrInvalidLocalTime
	}
	if isAmbiguousWallTime(localized, localCivilTime, loc) {
		return nil, ErrAmbiguousLocalTime
	}
	_, historicalOffset := localized.Zone()
	standardOffset := historicalOffset
	if localized.IsDST() {
		if offset, ok := findStandardOffset(localized); ok {
			standardOffset = offset
		}
	}
	dstOffset := historicalOffset - standardOffset
	standardWall := localCivilTime.Add(-time.Duration(dstOffset) * time.Second)
	return &TimezoneResult{
		TimezoneID: timezoneID, LocalCivilTime: localCivilTime, BirthUTC: localized.UTC(),
		LocalStandardTime: standardWall, HistoricalUTCOffset: historicalOffset,
		StandardUTCOffset: standardOffset, DSTApplied: localized.IsDST(), DSTOffset: dstOffset,
	}, nil
}

func isAmbiguousWallTime(chosen, wall time.Time, loc *time.Location) bool {
	for _, delta := range []time.Duration{-2 * time.Hour, -time.Hour, time.Hour, 2 * time.Hour} {
		candidate := chosen.Add(delta).In(loc)
		if candidate.Year() == wall.Year() && candidate.Month() == wall.Month() && candidate.Day() == wall.Day() &&
			candidate.Hour() == wall.Hour() && candidate.Minute() == wall.Minute() && candidate.Second() == wall.Second() {
			_, chosenOffset := chosen.Zone()
			_, candidateOffset := candidate.Zone()
			if chosenOffset != candidateOffset {
				return true
			}
		}
	}
	return false
}

func findStandardOffset(t time.Time) (int, bool) {
	for days := 1; days <= 370; days++ {
		for _, direction := range []int{-1, 1} {
			candidate := t.AddDate(0, 0, days*direction)
			if !candidate.IsDST() {
				_, offset := candidate.Zone()
				return offset, true
			}
		}
	}
	return 0, false
}
