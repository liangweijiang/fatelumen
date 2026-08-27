package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"fatelumen/backend/internal/birthchart"
)

type consistencyFixture struct {
	Cases []struct {
		Name           string  `json:"name"`
		TimezoneID     string  `json:"timezone_id"`
		Latitude       float64 `json:"latitude"`
		Longitude      float64 `json:"longitude"`
		LocalCivilTime string  `json:"local_civil_time"`
	} `json:"cases"`
}

func loadConsistencyFixture(t *testing.T) consistencyFixture {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve consistency fixture path")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(source), "..", "..", "testdata", "birthchart", "global_acceptance_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture consistencyFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func TestGlobalFreeAndReportPathConsistency(t *testing.T) {
	engine := birthchart.NewDefaultEngine()
	freeService := NewFreeChartService(engine)
	for _, tc := range loadConsistencyFixture(t).Cases {
		t.Run(tc.Name, func(t *testing.T) {
			wall, err := time.Parse("2006-01-02 15:04", tc.LocalCivilTime)
			if err != nil {
				t.Fatal(err)
			}
			location := birthchart.LocationInput{DisplayName: tc.Name, Latitude: tc.Latitude, Longitude: tc.Longitude, TimezoneID: tc.TimezoneID, HasCoordinates: true}
			freeInput := FreeChartInput{Gender: 1, CalendarType: 0, Year: wall.Year(), Month: int(wall.Month()), Day: wall.Day(), Hour: wall.Hour(), Minute: wall.Minute(), Location: location}
			freeResult, err := freeService.Calculate(context.Background(), freeInput)
			if err != nil {
				t.Fatal(err)
			}
			reportResult, err := engine.Calculate(context.Background(), birthchart.Input{Gender: freeInput.Gender, CalendarType: freeInput.CalendarType, Year: freeInput.Year, Month: freeInput.Month, Day: freeInput.Day, Hour: freeInput.Hour, Minute: freeInput.Minute, Location: location})
			if err != nil {
				t.Fatal(err)
			}
			reportHash := BuildChartHash(freeInput.Gender, freeInput.CalendarType, freeInput.Year, freeInput.Month, freeInput.Day, freeInput.Hour, freeInput.Minute, false, reportResult)
			if freeResult.ChartHash != reportHash || !reflect.DeepEqual(freeResult.Pillars, reportResult.Chart.Pillars) || !reflect.DeepEqual(freeResult.TimeCalculation, reportResult.Chart.Meta.TimeCalculation) {
				t.Fatalf("free/report core mismatch: free_hash=%s report_hash=%s", freeResult.ChartHash, reportHash)
			}
			t.Logf("hash=%s true_solar=%s", freeResult.ChartHash, freeResult.TimeCalculation.TrueSolarTime)
		})
	}
}
