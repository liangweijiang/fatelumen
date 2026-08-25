package service

import (
	"fatelumen/backend/internal/bazi"
	"fatelumen/backend/internal/birthchart"
	"fatelumen/backend/internal/pkg/hash"
)

func BuildChartHash(gender, calendarType int8, year, month, day, hour, minute int, isLeap bool, result *birthchart.Result) string {
	return hash.CalcChartHashV2(hash.ChartHashInput{Gender: gender, CalendarType: calendarType, Year: year, Month: month, Day: day, Hour: hour, Minute: minute, IsLeapMonth: isLeap, Latitude: result.Location.Latitude, Longitude: result.Location.Longitude, TimezoneID: result.Location.TimezoneID, HistoricalUTCOffset: result.Timezone.HistoricalUTCOffset, TimeCalculationMode: string(result.Mode), DayBoundaryRule: string(result.DayBoundaryRule), LocationVersion: result.LocationVersion, TimezoneVersion: "iana-runtime-v1", SolarAlgorithmVersion: result.SolarTime.AlgorithmVersion, CalculatorVersion: bazi.CalcVersion})
}
