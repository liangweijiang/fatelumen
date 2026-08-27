package birthchart

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

type globalAcceptanceFixture struct {
	FixedUTC string                 `json:"fixed_utc"`
	Cases    []globalAcceptanceCase `json:"cases"`
}

type globalAcceptanceCase struct {
	Name                    string  `json:"name"`
	TimezoneID              string  `json:"timezone_id"`
	Latitude                float64 `json:"latitude"`
	Longitude               float64 `json:"longitude"`
	LocalCivilTime          string  `json:"local_civil_time"`
	ExpectedStandardTime    string  `json:"expected_standard_time"`
	ExpectedTrueSolarMinute string  `json:"expected_true_solar_minute"`
	YearPillar              string  `json:"year_pillar"`
	MonthPillar             string  `json:"month_pillar"`
	DayPillar               string  `json:"day_pillar"`
	HourPillar              string  `json:"hour_pillar"`
}

func loadGlobalAcceptanceFixture(t *testing.T) globalAcceptanceFixture {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve acceptance fixture path")
	}
	path := filepath.Join(filepath.Dir(source), "..", "..", "testdata", "birthchart", "global_acceptance_cases.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fixture globalAcceptanceFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func acceptancePillar(stem, branch string) string { return stem + branch }

func TestGlobalSameInstantAcceptance(t *testing.T) {
	fixture := loadGlobalAcceptanceFixture(t)
	fixedUTC, err := time.Parse(time.RFC3339, fixture.FixedUTC)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewDefaultEngine()
	for _, tc := range fixture.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			wall, err := time.Parse("2006-01-02 15:04", tc.LocalCivilTime)
			if err != nil {
				t.Fatal(err)
			}
			result, err := engine.Calculate(context.Background(), Input{
				Gender: 1, CalendarType: 0,
				Year: wall.Year(), Month: int(wall.Month()), Day: wall.Day(), Hour: wall.Hour(), Minute: wall.Minute(),
				Location: LocationInput{DisplayName: tc.Name, Latitude: tc.Latitude, Longitude: tc.Longitude, TimezoneID: tc.TimezoneID, HasCoordinates: true},
			})
			if err != nil {
				t.Fatal(err)
			}
			if !result.Timezone.BirthUTC.Equal(fixedUTC) {
				t.Fatalf("same instant violated: got=%s want=%s", result.Timezone.BirthUTC, fixedUTC)
			}
			if got := result.Timezone.LocalStandardTime.Format("2006-01-02 15:04"); got != tc.ExpectedStandardTime {
				t.Fatalf("standard time=%s want=%s", got, tc.ExpectedStandardTime)
			}
			if got := result.SolarTime.TrueSolarTime.Format("2006-01-02 15:04"); got != tc.ExpectedTrueSolarMinute {
				t.Fatalf("true solar time=%s want=%s", got, tc.ExpectedTrueSolarMinute)
			}
			p := result.Chart.Pillars
			got := []string{acceptancePillar(p.Year.Stem, p.Year.Branch), acceptancePillar(p.Month.Stem, p.Month.Branch), acceptancePillar(p.Day.Stem, p.Day.Branch), acceptancePillar(p.Hour.Stem, p.Hour.Branch)}
			want := []string{tc.YearPillar, tc.MonthPillar, tc.DayPillar, tc.HourPillar}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("pillars=%v want=%v", got, want)
				}
			}
			t.Logf("UTC=%s civil=%s standard=%s true_solar=%s pillars=%v", fixedUTC.Format(time.RFC3339), tc.LocalCivilTime, tc.ExpectedStandardTime, result.SolarTime.TrueSolarTime.Format("2006-01-02 15:04:05"), got)
		})
	}
}

func TestGlobalAcceptanceSameInstantInvariants(t *testing.T) {
	fixture := loadGlobalAcceptanceFixture(t)
	if len(fixture.Cases) < 8 {
		t.Fatalf("global matrix is too small: %d", len(fixture.Cases))
	}
	baseYear, baseMonth := fixture.Cases[0].YearPillar, fixture.Cases[0].MonthPillar
	days := map[string]struct{}{}
	hours := map[string]struct{}{}
	for _, tc := range fixture.Cases {
		if tc.YearPillar != baseYear || tc.MonthPillar != baseMonth {
			t.Fatalf("same instant must keep year/month pillars: %s", tc.Name)
		}
		days[tc.DayPillar] = struct{}{}
		hours[tc.HourPillar] = struct{}{}
	}
	if len(days) < 2 || len(hours) < 4 {
		t.Fatalf("matrix must exercise solar-day and hour differences: days=%d hours=%d", len(days), len(hours))
	}
}
