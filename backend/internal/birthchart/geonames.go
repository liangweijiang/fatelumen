package birthchart

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"fatelumen/backend/internal/pkg/logger"
)

const GeoNamesLocationVersion = "geonames-webservice-v1"

type GeoNamesResolver struct {
	username string
	baseURL  string
	client   *http.Client
}

func (*GeoNamesResolver) Version() string { return GeoNamesLocationVersion }

func NewGeoNamesResolver(username, baseURL string, client *http.Client) *GeoNamesResolver {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://secure.geonames.org"
	}
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	return &GeoNamesResolver{username: strings.TrimSpace(username), baseURL: strings.TrimRight(baseURL, "/"), client: client}
}

type geoNamesSearchResponse struct {
	GeoNames []struct {
		ID          int64  `json:"geonameId"`
		Name        string `json:"name"`
		ToponymName string `json:"toponymName"`
		CountryCode string `json:"countryCode"`
		CountryName string `json:"countryName"`
		AdminCode1  string `json:"adminCode1"`
		AdminName1  string `json:"adminName1"`
		Latitude    string `json:"lat"`
		Longitude   string `json:"lng"`
	} `json:"geonames"`
	Status *struct {
		Message string `json:"message"`
	} `json:"status,omitempty"`
}

func (r *GeoNamesResolver) Search(ctx context.Context, keyword, locale string) ([]LocationCandidate, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" || r.username == "" {
		return nil, ErrLocationSearchUnavailable
	}
	values := url.Values{"q": {keyword}, "maxRows": {"12"}, "featureClass": {"P"}, "style": {"FULL"}, "username": {r.username}}
	if locale != "" {
		values.Set("lang", locale)
	}
	var payload geoNamesSearchResponse
	if err := r.getJSON(ctx, "/searchJSON", values, &payload); err != nil {
		return nil, err
	}
	if payload.Status != nil {
		return nil, fmt.Errorf("geonames search: %s", payload.Status.Message)
	}
	items := make([]LocationCandidate, 0, len(payload.GeoNames))
	for _, item := range payload.GeoNames {
		lat, latErr := strconv.ParseFloat(item.Latitude, 64)
		lng, lngErr := strconv.ParseFloat(item.Longitude, 64)
		if latErr != nil || lngErr != nil {
			continue
		}
		city := item.Name
		if city == "" {
			city = item.ToponymName
		}
		display := strings.Join(nonEmpty(city, item.AdminName1, item.CountryName), ", ")
		items = append(items, LocationCandidate{CountryCode: item.CountryCode, CountryName: item.CountryName, RegionCode: item.AdminCode1, RegionName: item.AdminName1, City: city, PlaceID: strconv.FormatInt(item.ID, 10), DisplayName: display, Latitude: lat, Longitude: lng})
	}
	return items, nil
}

func (r *GeoNamesResolver) Resolve(ctx context.Context, input LocationInput) (*ResolvedLocation, error) {
	if r.username == "" {
		return ProvidedLocationResolver{}.Resolve(ctx, input)
	}
	providedCoords := input.HasCoordinates
	if providedCoords && (input.Longitude < -180 || input.Longitude > 180 || input.Latitude < -90 || input.Latitude > 90) {
		return nil, ErrInvalidLocation
	}

	resolved := ResolvedLocation{CountryCode: input.CountryCode, CountryName: input.CountryName, RegionCode: input.RegionCode, RegionName: input.RegionName, City: input.City, PlaceID: input.PlaceID, DisplayName: input.DisplayName, Latitude: input.Latitude, Longitude: input.Longitude}
	if strings.TrimSpace(input.PlaceID) != "" {
		place, err := r.placeByID(ctx, input.PlaceID)
		if err != nil {
			return nil, err
		}
		if providedCoords && distanceKM(input.Latitude, input.Longitude, place.Latitude, place.Longitude) > 100 {
			return nil, ErrLocationConflict
		}
		if !providedCoords {
			resolved = *place
		} else {
			mergeLocationNames(&resolved, place)
		}
	} else if !providedCoords {
		query := strings.Join(nonEmpty(input.City, input.RegionName, input.CountryName), ", ")
		items, err := r.Search(ctx, query, "en")
		if err != nil || len(items) == 0 {
			return nil, ErrInvalidLocation
		}
		item := items[0]
		resolved = ResolvedLocation{CountryCode: item.CountryCode, CountryName: item.CountryName, RegionCode: item.RegionCode, RegionName: item.RegionName, City: item.City, PlaceID: item.PlaceID, DisplayName: item.DisplayName, Latitude: item.Latitude, Longitude: item.Longitude}
	} else if hasLocationNames(input) {
		place, err := r.nearby(ctx, input.Latitude, input.Longitude)
		if err != nil {
			return nil, err
		}
		if locationNamesConflict(input, place) {
			return nil, ErrLocationConflict
		}
		mergeLocationNames(&resolved, place)
	}
	if providedCoords && resolved.DisplayName == "" {
		if place, err := r.nearby(ctx, input.Latitude, input.Longitude); err == nil {
			mergeLocationNames(&resolved, place)
		}
	}
	tz, err := r.timezone(ctx, resolved.Latitude, resolved.Longitude)
	if err != nil {
		return nil, err
	}
	resolved.TimezoneID = tz
	return &resolved, nil
}

func hasLocationNames(input LocationInput) bool {
	return strings.TrimSpace(input.CountryCode) != "" || strings.TrimSpace(input.CountryName) != "" ||
		strings.TrimSpace(input.RegionCode) != "" || strings.TrimSpace(input.RegionName) != "" || strings.TrimSpace(input.City) != ""
}

func locationNamesConflict(input LocationInput, resolved *ResolvedLocation) bool {
	if strings.TrimSpace(input.CountryCode) != "" && resolved.CountryCode != "" && !strings.EqualFold(input.CountryCode, resolved.CountryCode) {
		return true
	}
	if strings.TrimSpace(input.CountryName) != "" && resolved.CountryName != "" && !strings.EqualFold(input.CountryName, resolved.CountryName) {
		return true
	}
	if strings.TrimSpace(input.RegionCode) != "" && resolved.RegionCode != "" && !strings.EqualFold(input.RegionCode, resolved.RegionCode) {
		return true
	}
	return strings.TrimSpace(input.RegionName) != "" && resolved.RegionName != "" && !strings.EqualFold(input.RegionName, resolved.RegionName)
}

func (r *GeoNamesResolver) placeByID(ctx context.Context, id string) (*ResolvedLocation, error) {
	var item struct {
		ID                                                               int64 `json:"geonameId"`
		Name, CountryCode, CountryName, AdminCode1, AdminName1, Lat, Lng string
	}
	if err := r.getJSON(ctx, "/getJSON", url.Values{"geonameId": {id}, "username": {r.username}}, &item); err != nil {
		return nil, err
	}
	lat, e1 := strconv.ParseFloat(item.Lat, 64)
	lng, e2 := strconv.ParseFloat(item.Lng, 64)
	if e1 != nil || e2 != nil {
		return nil, ErrInvalidLocation
	}
	return &ResolvedLocation{CountryCode: item.CountryCode, CountryName: item.CountryName, RegionCode: item.AdminCode1, RegionName: item.AdminName1, City: item.Name, PlaceID: strconv.FormatInt(item.ID, 10), DisplayName: strings.Join(nonEmpty(item.Name, item.AdminName1, item.CountryName), ", "), Latitude: lat, Longitude: lng}, nil
}

func (r *GeoNamesResolver) nearby(ctx context.Context, lat, lng float64) (*ResolvedLocation, error) {
	var payload geoNamesSearchResponse
	values := url.Values{"lat": {strconv.FormatFloat(lat, 'f', 6, 64)}, "lng": {strconv.FormatFloat(lng, 'f', 6, 64)}, "username": {r.username}}
	if err := r.getJSON(ctx, "/findNearbyPlaceNameJSON", values, &payload); err != nil {
		return nil, err
	}
	if len(payload.GeoNames) == 0 {
		return nil, ErrInvalidLocation
	}
	i := payload.GeoNames[0]
	return &ResolvedLocation{CountryCode: i.CountryCode, CountryName: i.CountryName, RegionCode: i.AdminCode1, RegionName: i.AdminName1, City: i.Name, PlaceID: strconv.FormatInt(i.ID, 10), DisplayName: strings.Join(nonEmpty(i.Name, i.AdminName1, i.CountryName), ", ")}, nil
}

func (r *GeoNamesResolver) timezone(ctx context.Context, lat, lng float64) (string, error) {
	var payload struct {
		TimezoneID string `json:"timezoneId"`
		Status     *struct {
			Message string `json:"message"`
		} `json:"status,omitempty"`
	}
	v := url.Values{"lat": {strconv.FormatFloat(lat, 'f', 6, 64)}, "lng": {strconv.FormatFloat(lng, 'f', 6, 64)}, "username": {r.username}}
	if err := r.getJSON(ctx, "/timezoneJSON", v, &payload); err != nil {
		return "", err
	}
	if payload.Status != nil || payload.TimezoneID == "" {
		return "", ErrInvalidLocation
	}
	if _, err := time.LoadLocation(payload.TimezoneID); err != nil {
		return "", ErrInvalidLocation
	}
	return payload.TimezoneID, nil
}

func (r *GeoNamesResolver) getJSON(ctx context.Context, path string, values url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+path+"?"+values.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		logger.FromCtx(ctx).Error("geonames request failed", "err", err, "operation", path)
		return fmt.Errorf("geonames request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.FromCtx(ctx).Error("geonames response failed", "operation", path, "status", resp.StatusCode)
		return fmt.Errorf("geonames response status: %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		logger.FromCtx(ctx).Error("decode geonames response failed", "operation", path, "err", err)
		return fmt.Errorf("decode geonames response: %w", err)
	}
	return nil
}

func nonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}
func mergeLocationNames(dst *ResolvedLocation, src *ResolvedLocation) {
	if dst.CountryCode == "" {
		dst.CountryCode = src.CountryCode
	}
	if dst.CountryName == "" {
		dst.CountryName = src.CountryName
	}
	if dst.RegionCode == "" {
		dst.RegionCode = src.RegionCode
	}
	if dst.RegionName == "" {
		dst.RegionName = src.RegionName
	}
	if dst.City == "" {
		dst.City = src.City
	}
	if dst.PlaceID == "" {
		dst.PlaceID = src.PlaceID
	}
	if dst.DisplayName == "" {
		dst.DisplayName = src.DisplayName
	}
}
func distanceKM(aLat, aLng, bLat, bLng float64) float64 {
	const radius = 6371.0
	dLat := (bLat - aLat) * math.Pi / 180
	dLng := (bLng - aLng) * math.Pi / 180
	x := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(aLat*math.Pi/180)*math.Cos(bLat*math.Pi/180)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return radius * 2 * math.Atan2(math.Sqrt(x), math.Sqrt(1-x))
}
