package birthchart

import (
	"context"
	"fmt"
	"time"

	"fatelumen/backend/internal/bazi"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
)

const EngineVersion = "birth-chart-engine-v2"

type DefaultEngine struct {
	normalizer InputNormalizer
	locations  LocationResolver
	timezones  HistoricalTimezoneResolver
	solar      SolarTimeEngine
	calculator BaziCalculator
}

func NewEngine(normalizer InputNormalizer, locations LocationResolver, timezones HistoricalTimezoneResolver, solar SolarTimeEngine, calculator BaziCalculator) *DefaultEngine {
	return &DefaultEngine{normalizer: normalizer, locations: locations, timezones: timezones, solar: solar, calculator: calculator}
}

func NewDefaultEngine() *DefaultEngine {
	return NewEngine(DefaultInputNormalizer{}, ProvidedLocationResolver{}, IANATimezoneResolver{}, DefaultSolarTimeEngine{}, LunarGoCalculator{})
}

func (e *DefaultEngine) Calculate(ctx context.Context, input Input) (*Result, error) {
	normalized, err := e.normalizer.Normalize(ctx, input)
	if err != nil {
		logger.FromCtx(ctx).Warn("normalize birth chart input failed", "err", err)
		return nil, fmt.Errorf("normalize birth input: %w", err)
	}
	location, err := e.locations.Resolve(ctx, input.Location)
	if err != nil {
		logger.FromCtx(ctx).Warn("resolve birth location failed", "err", err)
		return nil, fmt.Errorf("resolve birth location: %w", err)
	}
	timezone, err := e.timezones.Resolve(ctx, normalized.LocalCivilTime, location.TimezoneID)
	if err != nil {
		logger.FromCtx(ctx).Error("resolve historical timezone failed", "timezone_id", location.TimezoneID, "err", err)
		return nil, fmt.Errorf("resolve historical timezone: %w", err)
	}
	solar, err := e.solar.Calculate(SolarTimeInput{LocalStandardTime: timezone.LocalStandardTime, StandardUTCOffset: timezone.StandardUTCOffset, Longitude: location.Longitude})
	if err != nil {
		logger.FromCtx(ctx).Error("calculate true solar time failed", "timezone_id", location.TimezoneID, "err", err)
		return nil, fmt.Errorf("calculate true solar time: %w", err)
	}
	chart, err := e.calculator.Calculate(ctx, CalculatorInput{Gender: input.Gender, TrueSolarTime: solar.TrueSolarTime, DayBoundaryRule: Midnight00})
	if err != nil {
		logger.FromCtx(ctx).Error("calculate lunar-go chart failed", "err", err)
		return nil, fmt.Errorf("calculate chart: %w", err)
	}
	chart.Meta.TimeCalculation = model.TimeCalculationMeta{
		TimezoneID: location.TimezoneID, HistoricalUTCOffset: timezone.HistoricalUTCOffset,
		StandardUTCOffset: timezone.StandardUTCOffset, DSTApplied: timezone.DSTApplied, DSTOffset: timezone.DSTOffset,
		Longitude: location.Longitude, Latitude: location.Latitude,
		StandardMeridian: solar.StandardMeridian, LongitudeCorrectionMinute: solar.LongitudeCorrectionMinute,
		EquationOfTimeMinute: solar.EquationOfTimeMinute,
		LocalCivilTime:       timezone.LocalCivilTime.Format("2006-01-02 15:04:05"),
		LocalStandardTime:    timezone.LocalStandardTime.Format("2006-01-02 15:04:05"),
		MeanSolarTime:        solar.MeanSolarTime.Format("2006-01-02 15:04:05"),
		TrueSolarTime:        solar.TrueSolarTime.Format("2006-01-02 15:04:05"),
		Mode:                 string(TrueSolarTime), DayBoundaryRule: string(Midnight00),
		SolarAlgorithmVersion: solar.AlgorithmVersion, EngineVersion: EngineVersion,
	}
	return &Result{Location: *location, Timezone: *timezone, SolarTime: *solar, Mode: TrueSolarTime, DayBoundaryRule: Midnight00, Chart: chart, EngineVersion: EngineVersion, LocationVersion: e.locations.Version()}, nil
}

type LunarGoCalculator struct{ AnnualCalendar AnnualCalendarProvider }

func (c LunarGoCalculator) Calculate(ctx context.Context, input CalculatorInput) (*model.ChartData, error) {
	rule := string(input.DayBoundaryRule)
	var years []model.AnnualCalendarYear
	if c.AnnualCalendar != nil {
		var err error
		years, err = c.AnnualCalendar.Range(ctx, time.Now().Year(), 10)
		if err != nil {
			return nil, fmt.Errorf("load annual calendar: %w", err)
		}
		if len(years) != 10 {
			return nil, fmt.Errorf("annual calendar incomplete: expected 10 years, got %d", len(years))
		}
	}
	return bazi.Calculate(bazi.BirthInput{
		Gender: input.Gender, CalendarType: 0,
		Year: input.TrueSolarTime.Year(), Month: int(input.TrueSolarTime.Month()), Day: input.TrueSolarTime.Day(),
		Hour: input.TrueSolarTime.Hour(), Minute: input.TrueSolarTime.Minute(),
		NormalizedSolar: true, DayBoundaryRule: rule, AnnualCalendar: years,
	})
}
