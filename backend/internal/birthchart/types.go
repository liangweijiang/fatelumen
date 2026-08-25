package birthchart

import (
	"context"
	"time"

	"fatelumen/backend/internal/model"
)

type TimeCalculationMode string
type DayBoundaryRule string

const (
	TrueSolarTime TimeCalculationMode = "TRUE_SOLAR_TIME"
	Midnight00    DayBoundaryRule     = "MIDNIGHT_00"
	LateZi23      DayBoundaryRule     = "LATE_ZI_23"
)

type LocationInput struct {
	CountryCode    string  `json:"country_code,omitempty"`
	CountryName    string  `json:"country_name,omitempty"`
	RegionCode     string  `json:"region_code,omitempty"`
	RegionName     string  `json:"region_name,omitempty"`
	City           string  `json:"city,omitempty"`
	PlaceID        string  `json:"place_id,omitempty"`
	DisplayName    string  `json:"display_name,omitempty"`
	Latitude       float64 `json:"latitude,omitempty"`
	Longitude      float64 `json:"longitude,omitempty"`
	TimezoneID     string  `json:"timezone_id,omitempty"`
	HasCoordinates bool    `json:"has_coordinates"`
}

type Input struct {
	Gender       int8
	CalendarType int8
	Year         int
	Month        int
	Day          int
	Hour         int
	Minute       int
	IsLeapMonth  bool
	Location     LocationInput
}

type NormalizedInput struct {
	Original       Input
	LocalCivilTime time.Time
}

type ResolvedLocation struct {
	CountryCode string  `json:"country_code,omitempty"`
	CountryName string  `json:"country_name,omitempty"`
	RegionCode  string  `json:"region_code,omitempty"`
	RegionName  string  `json:"region_name,omitempty"`
	City        string  `json:"city,omitempty"`
	PlaceID     string  `json:"place_id,omitempty"`
	DisplayName string  `json:"display_name,omitempty"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	TimezoneID  string  `json:"timezone_id"`
}

type LocationCandidate struct {
	CountryCode string  `json:"country_code,omitempty"`
	CountryName string  `json:"country_name,omitempty"`
	RegionCode  string  `json:"region_code,omitempty"`
	RegionName  string  `json:"region_name,omitempty"`
	City        string  `json:"city,omitempty"`
	PlaceID     string  `json:"place_id"`
	DisplayName string  `json:"display_name"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type TimezoneResult struct {
	TimezoneID          string    `json:"timezone_id"`
	LocalCivilTime      time.Time `json:"local_civil_time"`
	BirthUTC            time.Time `json:"birth_utc"`
	LocalStandardTime   time.Time `json:"local_standard_time"`
	HistoricalUTCOffset int       `json:"historical_utc_offset_seconds"`
	StandardUTCOffset   int       `json:"standard_utc_offset_seconds"`
	DSTApplied          bool      `json:"dst_applied"`
	DSTOffset           int       `json:"dst_offset_seconds"`
}

type SolarTimeResult struct {
	StandardMeridian          float64   `json:"standard_meridian"`
	LongitudeCorrectionMinute float64   `json:"longitude_correction_minutes"`
	MeanSolarTime             time.Time `json:"mean_solar_time"`
	EquationOfTimeMinute      float64   `json:"equation_of_time_minutes"`
	TrueSolarTime             time.Time `json:"true_solar_time"`
	AlgorithmVersion          string    `json:"algorithm_version"`
}

type Result struct {
	Location        ResolvedLocation
	Timezone        TimezoneResult
	SolarTime       SolarTimeResult
	Mode            TimeCalculationMode
	DayBoundaryRule DayBoundaryRule
	Chart           *model.ChartData
	EngineVersion   string
	LocationVersion string
}

type InputNormalizer interface {
	Normalize(ctx context.Context, input Input) (*NormalizedInput, error)
}

type LocationResolver interface {
	Version() string
	Search(ctx context.Context, keyword, locale string) ([]LocationCandidate, error)
	Resolve(ctx context.Context, input LocationInput) (*ResolvedLocation, error)
}

type HistoricalTimezoneResolver interface {
	Resolve(ctx context.Context, localCivilTime time.Time, timezoneID string) (*TimezoneResult, error)
}

type SolarTimeEngine interface {
	Calculate(input SolarTimeInput) (*SolarTimeResult, error)
}

type BaziCalculator interface {
	Calculate(ctx context.Context, input CalculatorInput) (*model.ChartData, error)
}

type Engine interface {
	Calculate(ctx context.Context, input Input) (*Result, error)
}

type SolarTimeInput struct {
	LocalStandardTime time.Time
	StandardUTCOffset int
	Longitude         float64
}

type CalculatorInput struct {
	Gender          int8
	TrueSolarTime   time.Time
	DayBoundaryRule DayBoundaryRule
}
