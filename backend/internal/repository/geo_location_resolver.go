package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"fatelumen/backend/internal/birthchart"
	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
)

const GeoLocationVersion = "geonames-fixed-2026-08-24"
const coordinateTolerance = 0.0001

type geoLocationStore interface {
	SearchCities(ctx context.Context, query, locale string, limit int) ([]model.GeoCity, error)
	CityContext(ctx context.Context, id uint64) (*model.GeoCity, error)
	CountryName(ctx context.Context, code, locale string) (string, error)
}

type GeoLocationResolver struct{ repo geoLocationStore }

func NewGeoLocationResolver(repo geoLocationStore) *GeoLocationResolver {
	return &GeoLocationResolver{repo: repo}
}

func (*GeoLocationResolver) Version() string { return GeoLocationVersion }

func (r *GeoLocationResolver) Search(ctx context.Context, keyword, locale string) ([]birthchart.LocationCandidate, error) {
	rows, err := r.repo.SearchCities(ctx, keyword, locale, 20)
	if err != nil {
		return nil, err
	}
	items := make([]birthchart.LocationCandidate, 0, len(rows))
	for i := range rows {
		items = append(items, mapGeoCandidate(&rows[i], locale))
	}
	return items, nil
}

func (r *GeoLocationResolver) Resolve(ctx context.Context, input birthchart.LocationInput) (*birthchart.ResolvedLocation, error) {
	placeID := strings.TrimSpace(input.PlaceID)
	if placeID == "" {
		return birthchart.ProvidedLocationResolver{}.Resolve(ctx, input)
	}
	id, err := strconv.ParseUint(placeID, 10, 64)
	if err != nil {
		return nil, birthchart.ErrInvalidLocation
	}
	row, err := r.repo.CityContext(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, birthchart.ErrInvalidLocation
	}
	if err != nil {
		return nil, fmt.Errorf("load fixed geo city: %w", err)
	}
	if input.CountryCode != "" && !strings.EqualFold(input.CountryCode, row.CountryCode) {
		return nil, birthchart.ErrLocationConflict
	}
	if input.HasCoordinates && (math.Abs(input.Latitude-row.Latitude) > coordinateTolerance || math.Abs(input.Longitude-row.Longitude) > coordinateTolerance) {
		return nil, birthchart.ErrLocationConflict
	}
	countryName, err := r.repo.CountryName(ctx, row.CountryCode, "en")
	if err != nil {
		return nil, fmt.Errorf("load fixed geo country: %w", err)
	}
	timezoneID := row.TimezoneID
	if requestedTimezone := strings.TrimSpace(input.TimezoneID); requestedTimezone != "" {
		if _, err := time.LoadLocation(requestedTimezone); err != nil {
			return nil, birthchart.ErrInvalidLocation
		}
		timezoneID = requestedTimezone
	}
	return &birthchart.ResolvedLocation{
		CountryCode: row.CountryCode, CountryName: countryName,
		RegionCode: row.Admin1Code, RegionName: row.NameEN, City: row.NameEN,
		PlaceID: strconv.FormatUint(row.GeoNameID, 10), DisplayName: row.NameEN + ", " + countryName,
		Latitude: row.Latitude, Longitude: row.Longitude, TimezoneID: timezoneID,
	}, nil
}

func mapGeoCandidate(row *model.GeoCity, locale string) birthchart.LocationCandidate {
	name := geoCityName(row, locale)
	return birthchart.LocationCandidate{
		CountryCode: row.CountryCode, RegionCode: row.Admin1Code, RegionName: row.Admin1Name,
		City: name, PlaceID: strconv.FormatUint(row.GeoNameID, 10), DisplayName: name,
		Latitude: row.Latitude, Longitude: row.Longitude,
	}
}

func geoCityName(row *model.GeoCity, locale string) string {
	switch strings.ToLower(locale) {
	case "zh":
		return row.NameZH
	case "ja":
		return row.NameJA
	case "ko":
		return row.NameKO
	default:
		return row.NameEN
	}
}
