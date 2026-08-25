package repository

import (
	"context"
	"errors"
	"testing"

	"fatelumen/backend/internal/birthchart"
	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
)

type fakeGeoLocationStore struct {
	city *model.GeoCity
}

func (f fakeGeoLocationStore) SearchCities(context.Context, string, string, int) ([]model.GeoCity, error) {
	return []model.GeoCity{*f.city}, nil
}
func (f fakeGeoLocationStore) CityContext(_ context.Context, id uint64) (*model.GeoCity, error) {
	if f.city == nil || f.city.GeoNameID != id {
		return nil, gorm.ErrRecordNotFound
	}
	copy := *f.city
	return &copy, nil
}
func (fakeGeoLocationStore) CountryName(context.Context, string, string) (string, error) {
	return "China", nil
}

func testShanghai() *model.GeoCity {
	return &model.GeoCity{GeoNameID: 1796236, CountryCode: "CN", Admin1Code: "23", NameEN: "Shanghai", NameZH: "上海", Latitude: 31.22222, Longitude: 121.45806, TimezoneID: "Asia/Shanghai"}
}

func TestGeoLocationResolverUsesFixedCity(t *testing.T) {
	resolver := NewGeoLocationResolver(fakeGeoLocationStore{city: testShanghai()})
	got, err := resolver.Resolve(context.Background(), birthchart.LocationInput{CountryCode: "CN", PlaceID: "1796236"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.Longitude != 121.45806 || got.Latitude != 31.22222 || got.TimezoneID != "Asia/Shanghai" {
		t.Fatalf("Resolve() = %#v", got)
	}
}

func TestGeoLocationResolverAcceptsMatchingCoordinates(t *testing.T) {
	resolver := NewGeoLocationResolver(fakeGeoLocationStore{city: testShanghai()})
	_, err := resolver.Resolve(context.Background(), birthchart.LocationInput{CountryCode: "CN", PlaceID: "1796236", HasCoordinates: true, Latitude: 31.22222, Longitude: 121.45806})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
}

func TestGeoLocationResolverAllowsExplicitValidTimezoneOverride(t *testing.T) {
	resolver := NewGeoLocationResolver(fakeGeoLocationStore{city: testShanghai()})
	got, err := resolver.Resolve(context.Background(), birthchart.LocationInput{CountryCode: "CN", PlaceID: "1796236", TimezoneID: "Asia/Urumqi"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.TimezoneID != "Asia/Urumqi" || got.Longitude != 121.45806 {
		t.Fatalf("Resolve() = %#v", got)
	}
}

func TestGeoLocationResolverRejectsInvalidTimezoneOverride(t *testing.T) {
	resolver := NewGeoLocationResolver(fakeGeoLocationStore{city: testShanghai()})
	_, err := resolver.Resolve(context.Background(), birthchart.LocationInput{CountryCode: "CN", PlaceID: "1796236", TimezoneID: "Mars/Olympus"})
	if !errors.Is(err, birthchart.ErrInvalidLocation) {
		t.Fatalf("Resolve() error = %v, want ErrInvalidLocation", err)
	}
}

func TestGeoLocationResolverRejectsConflictingCoordinates(t *testing.T) {
	resolver := NewGeoLocationResolver(fakeGeoLocationStore{city: testShanghai()})
	_, err := resolver.Resolve(context.Background(), birthchart.LocationInput{CountryCode: "CN", PlaceID: "1796236", HasCoordinates: true, Latitude: 40, Longitude: 116})
	if !errors.Is(err, birthchart.ErrLocationConflict) {
		t.Fatalf("Resolve() error = %v, want ErrLocationConflict", err)
	}
}
