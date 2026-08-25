package birthchart

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeoNamesResolverSearchAndResolve(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/searchJSON":
			_, _ = w.Write([]byte(`{"geonames":[{"geonameId":1796236,"name":"Shanghai","countryCode":"CN","countryName":"China","adminCode1":"23","adminName1":"Shanghai","lat":"31.22222","lng":"121.45806"}]}`))
		case "/getJSON":
			_, _ = w.Write([]byte(`{"geonameId":1796236,"name":"Shanghai","countryCode":"CN","countryName":"China","adminCode1":"23","adminName1":"Shanghai","lat":"31.22222","lng":"121.45806"}`))
		case "/timezoneJSON":
			_, _ = w.Write([]byte(`{"timezoneId":"Asia/Shanghai"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resolver := NewGeoNamesResolver("test-user", server.URL, server.Client())
	items, err := resolver.Search(context.Background(), "Shanghai", "en")
	if err != nil || len(items) != 1 || items[0].PlaceID != "1796236" {
		t.Fatalf("unexpected search result: items=%+v err=%v", items, err)
	}
	resolved, err := resolver.Resolve(context.Background(), LocationInput{PlaceID: items[0].PlaceID})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if resolved.TimezoneID != "Asia/Shanghai" || resolved.City != "Shanghai" {
		t.Fatalf("unexpected resolved location: %+v", resolved)
	}
}

func TestGeoNamesResolverRejectsConflictingCoordinates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"geonameId":1796236,"name":"Shanghai","countryCode":"CN","countryName":"China","adminCode1":"23","adminName1":"Shanghai","lat":"31.22222","lng":"121.45806"}`))
	}))
	defer server.Close()
	resolver := NewGeoNamesResolver("test-user", server.URL, server.Client())
	_, err := resolver.Resolve(context.Background(), LocationInput{PlaceID: "1796236", Latitude: 40.7128, Longitude: -74.006, HasCoordinates: true})
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestGeoNamesResolverRejectsConflictingNamedRegionAndCoordinates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/findNearbyPlaceNameJSON":
			_, _ = w.Write([]byte(`{"geonames":[{"geonameId":5128581,"name":"New York City","countryCode":"US","countryName":"United States","adminCode1":"NY","adminName1":"New York"}]}`))
		case "/timezoneJSON":
			_, _ = w.Write([]byte(`{"timezoneId":"America/New_York"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	resolver := NewGeoNamesResolver("test-user", server.URL, server.Client())
	_, err := resolver.Resolve(context.Background(), LocationInput{
		CountryCode: "CN", RegionCode: "SH", Latitude: 40.7128, Longitude: -74.006, HasCoordinates: true,
	})
	if err != ErrLocationConflict {
		t.Fatalf("expected location conflict, got %v", err)
	}
}
