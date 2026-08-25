package birthchart

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrLocationSearchUnavailable = errors.New("location search is not configured")
var ErrLocationConflict = errors.New("birth location and coordinates conflict")

// ProvidedLocationResolver validates already resolved location data. A geocoding
// provider can replace it through dependency injection without changing the engine.
type ProvidedLocationResolver struct{}

func (ProvidedLocationResolver) Version() string { return "provided-location-v1" }

func (ProvidedLocationResolver) Search(_ context.Context, _, _ string) ([]LocationCandidate, error) {
	return nil, ErrLocationSearchUnavailable
}

func (ProvidedLocationResolver) Resolve(_ context.Context, input LocationInput) (*ResolvedLocation, error) {
	if !input.HasCoordinates || input.Longitude < -180 || input.Longitude > 180 || input.Latitude < -90 || input.Latitude > 90 {
		return nil, ErrInvalidLocation
	}
	timezoneID := strings.TrimSpace(input.TimezoneID)
	if timezoneID == "" {
		return nil, ErrInvalidLocation
	}
	if _, err := time.LoadLocation(timezoneID); err != nil {
		return nil, ErrInvalidLocation
	}
	return &ResolvedLocation{
		CountryCode: input.CountryCode, CountryName: input.CountryName,
		RegionCode: input.RegionCode, RegionName: input.RegionName, City: input.City,
		PlaceID: input.PlaceID, DisplayName: input.DisplayName,
		Latitude: input.Latitude, Longitude: input.Longitude, TimezoneID: timezoneID,
	}, nil
}
