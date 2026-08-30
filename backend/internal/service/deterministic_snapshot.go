package service

import (
	"encoding/json"
	"time"

	"fatelumen/backend/internal/bazi"
	"fatelumen/backend/internal/birthchart"
	"fatelumen/backend/internal/facts"
	"fatelumen/backend/internal/llm/prompts"
	"fatelumen/backend/internal/model"
)

// BuildDeterministicSnapshot converts one BirthChartEngine result into the
// same immutable facts contract consumed by full reports and admin archives.
func BuildDeterministicSnapshot(input birthchart.Input, locale, chartHash string, result *birthchart.Result) (model.ReportInputSnapshot, model.TimeCalculationSnapshot, model.ChartSnapshot, model.InterpretationFacts, error) {
	in := model.ReportInputSnapshot{CalendarType: int(input.CalendarType), Year: input.Year, Month: input.Month, Day: input.Day, Hour: input.Hour, Minute: input.Minute, IsLeapMonth: input.IsLeapMonth, Gender: int(input.Gender), CountryCode: input.Location.CountryCode, RegionCode: input.Location.RegionCode, PlaceID: input.Location.PlaceID, Longitude: result.Location.Longitude, Latitude: result.Location.Latitude, TimezoneID: result.Timezone.TimezoneID, Locale: locale, ProfileMode: "admin_archive", SchemaVersion: model.ReportInputSchemaVersion}
	tm := model.TimeCalculationSnapshot{LocalCivilTime: result.Timezone.LocalCivilTime.Format(time.RFC3339), TimezoneID: result.Timezone.TimezoneID, HistoricalUTCOffsetSeconds: result.Timezone.HistoricalUTCOffset, StandardUTCOffsetSeconds: result.Timezone.StandardUTCOffset, DSTApplied: result.Timezone.DSTApplied, DSTOffsetSeconds: result.Timezone.DSTOffset, StandardMeridian: result.SolarTime.StandardMeridian, LongitudeCorrectionMinutes: result.SolarTime.LongitudeCorrectionMinute, EquationOfTimeMinutes: result.SolarTime.EquationOfTimeMinute, LocalStandardTime: result.Timezone.LocalStandardTime.Format(time.RFC3339), MeanSolarTime: result.SolarTime.MeanSolarTime.Format(time.RFC3339), TrueSolarTime: result.SolarTime.TrueSolarTime.Format(time.RFC3339), CrossedDateBoundary: !sameCivilDate(result.Timezone.LocalCivilTime, result.SolarTime.TrueSolarTime), DayBoundaryRule: string(result.DayBoundaryRule), SolarAlgorithmVersion: result.SolarTime.AlgorithmVersion, LocationDatabaseVersion: result.LocationVersion, TimezoneDatabaseVersion: "iana-runtime-v1"}
	chart := model.ChartSnapshot{ChartHash: chartHash, ChartSchemaVersion: model.ChartSnapshotVersion, EngineVersion: result.EngineVersion, LunarGoVersion: bazi.CalcVersion, Data: *result.Chart}
	versions := model.InterpretationFactVersions{PromptVersion: prompts.ReportPromptVersion, SolarAlgorithmVersion: result.SolarTime.AlgorithmVersion, LocationDatabaseVersion: result.LocationVersion, TimezoneDatabaseVersion: "iana-runtime-v1", LunarGoVersion: bazi.CalcVersion}
	built, err := facts.Build(facts.BuildInput{Input: in, TimeCalculation: tm, Chart: chart, Versions: versions})
	return in, tm, chart, built, err
}

func marshalSnapshot(v any) (model.JSONRaw, error) {
	b, err := json.Marshal(v)
	return model.JSONRaw(b), err
}
