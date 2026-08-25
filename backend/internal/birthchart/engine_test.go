package birthchart

import (
	"context"
	"testing"
	"time"

	"fatelumen/backend/internal/model"
)

type fixedLocationResolver struct{ location ResolvedLocation }

func (r fixedLocationResolver) Version() string { return "test-location-v1" }
func (r fixedLocationResolver) Search(context.Context, string, string) ([]LocationCandidate, error) {
	return nil, nil
}
func (r fixedLocationResolver) Resolve(context.Context, LocationInput) (*ResolvedLocation, error) {
	result := r.location
	return &result, nil
}

type recordingCalculator struct{ input CalculatorInput }

func (r *recordingCalculator) Calculate(_ context.Context, input CalculatorInput) (*model.ChartData, error) {
	r.input = input
	return &model.ChartData{}, nil
}

func TestIANATimezoneResolverHistoricalDST(t *testing.T) {
	resolver := IANATimezoneResolver{}
	summer, err := resolver.Resolve(context.Background(), wallTime(2020, 7, 1, 12, 0, 0), "America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	if !summer.DSTApplied || summer.HistoricalUTCOffset != -4*3600 || summer.StandardUTCOffset != -5*3600 || summer.DSTOffset != 3600 {
		t.Fatalf("unexpected summer offsets: %+v", summer)
	}
	winter, err := resolver.Resolve(context.Background(), wallTime(2020, 1, 1, 12, 0, 0), "America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	if winter.DSTApplied || winter.HistoricalUTCOffset != -5*3600 || winter.StandardUTCOffset != -5*3600 || winter.DSTOffset != 0 {
		t.Fatalf("unexpected winter offsets: %+v", winter)
	}
}

func TestIANATimezoneResolverNonHourOffsets(t *testing.T) {
	resolver := IANATimezoneResolver{}
	for _, tc := range []struct {
		zone string
		want int
	}{
		{zone: "Asia/Kolkata", want: 5*3600 + 30*60},
		{zone: "Asia/Kathmandu", want: 5*3600 + 45*60},
		{zone: "America/St_Johns", want: -(3*3600 + 30*60)},
	} {
		result, err := resolver.Resolve(context.Background(), wallTime(2020, 1, 15, 12, 0, 0), tc.zone)
		if err != nil {
			t.Fatalf("%s: %v", tc.zone, err)
		}
		if result.HistoricalUTCOffset != tc.want {
			t.Fatalf("%s offset=%d want=%d", tc.zone, result.HistoricalUTCOffset, tc.want)
		}
	}
}

func TestInputNormalizerLunarAndLeapMonthRegression(t *testing.T) {
	normalizer := DefaultInputNormalizer{}
	regular, err := normalizer.Normalize(context.Background(), Input{Gender: 1, CalendarType: 1, Year: 2020, Month: 1, Day: 1, Hour: 12})
	if err != nil {
		t.Fatal(err)
	}
	if got := regular.LocalCivilTime.Format("2006-01-02"); got != "2020-01-25" {
		t.Fatalf("lunar new year conversion=%s", got)
	}
	leap, err := normalizer.Normalize(context.Background(), Input{Gender: 1, CalendarType: 1, Year: 2020, Month: 4, Day: 1, Hour: 12, IsLeapMonth: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := leap.LocalCivilTime.Format("2006-01-02"); got != "2020-05-23" {
		t.Fatalf("leap month conversion=%s", got)
	}
}

func TestIANATimezoneResolverRejectsDSTGapAndOverlap(t *testing.T) {
	resolver := IANATimezoneResolver{}
	if _, err := resolver.Resolve(context.Background(), wallTime(2020, 3, 8, 2, 30, 0), "America/New_York"); err != ErrInvalidLocalTime {
		t.Fatalf("expected invalid DST gap, got %v", err)
	}
	if _, err := resolver.Resolve(context.Background(), wallTime(2020, 11, 1, 1, 30, 0), "America/New_York"); err != ErrAmbiguousLocalTime {
		t.Fatalf("expected ambiguous DST overlap, got %v", err)
	}
}

func TestSolarTimeLongitudeCorrectionDirection(t *testing.T) {
	engine := DefaultSolarTimeEngine{}
	local := wallTime(2020, 6, 1, 12, 0, 0)
	east, err := engine.Calculate(SolarTimeInput{LocalStandardTime: local, StandardUTCOffset: 8 * 3600, Longitude: 121})
	if err != nil {
		t.Fatal(err)
	}
	west, err := engine.Calculate(SolarTimeInput{LocalStandardTime: local, StandardUTCOffset: 8 * 3600, Longitude: 119})
	if err != nil {
		t.Fatal(err)
	}
	if east.LongitudeCorrectionMinute != 4 || west.LongitudeCorrectionMinute != -4 || !east.TrueSolarTime.After(west.TrueSolarTime) {
		t.Fatalf("unexpected longitude correction: east=%+v west=%+v", east, west)
	}
	if east.AlgorithmVersion != SolarAlgorithmVersion || east.EquationOfTimeMinute > 5 || east.EquationOfTimeMinute < 0 {
		t.Fatalf("unexpected equation of time regression: %+v", east)
	}
}

func TestEngineAppliesTrueSolarCrossDayBeforeMidnightRule(t *testing.T) {
	cases := []struct {
		name      string
		hour      int
		longitude float64
		wantDay   int
	}{
		{name: "crosses to previous day", hour: 1, longitude: 75, wantDay: 31},
		{name: "crosses to next day", hour: 22, longitude: 180, wantDay: 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calculator := &recordingCalculator{}
			engine := NewEngine(DefaultInputNormalizer{}, fixedLocationResolver{ResolvedLocation{Latitude: 0, Longitude: tc.longitude, TimezoneID: "Asia/Shanghai"}}, IANATimezoneResolver{}, DefaultSolarTimeEngine{}, calculator)
			_, err := engine.Calculate(context.Background(), Input{Gender: 1, CalendarType: 0, Year: 2020, Month: 1, Day: 1, Hour: tc.hour, Location: LocationInput{HasCoordinates: true}})
			if err != nil {
				t.Fatal(err)
			}
			if calculator.input.TrueSolarTime.Day() != tc.wantDay || calculator.input.DayBoundaryRule != Midnight00 {
				t.Fatalf("unexpected final calculator input: %+v", calculator.input)
			}
		})
	}
}

func TestEngineMidnightBoundaryRegression(t *testing.T) {
	for _, minute := range []int{-1, 0, 1} {
		calculator := &recordingCalculator{}
		engine := NewEngine(DefaultInputNormalizer{}, fixedLocationResolver{ResolvedLocation{Longitude: 120, TimezoneID: "Asia/Shanghai"}}, IANATimezoneResolver{}, SolarTimeEngineFunc(func(input SolarTimeInput) (*SolarTimeResult, error) {
			final := wallTime(2020, 1, 2, 0, minute, 0)
			if minute < 0 {
				final = wallTime(2020, 1, 1, 23, 59, 0)
			}
			return &SolarTimeResult{TrueSolarTime: final, MeanSolarTime: final, AlgorithmVersion: "boundary-test"}, nil
		}), calculator)
		if _, err := engine.Calculate(context.Background(), Input{Gender: 1, CalendarType: 0, Year: 2020, Month: 1, Day: 2, Hour: 0, Location: LocationInput{HasCoordinates: true}}); err != nil {
			t.Fatal(err)
		}
		wantDay := 2
		if minute < 0 {
			wantDay = 1
		}
		if calculator.input.TrueSolarTime.Day() != wantDay || calculator.input.DayBoundaryRule != Midnight00 {
			t.Fatalf("minute=%d input=%+v", minute, calculator.input)
		}
	}
}

func TestLunarGoMidnight00Boundary(t *testing.T) {
	calculator := LunarGoCalculator{}
	times := []struct {
		name string
		time time.Time
	}{
		{name: "22:59", time: wallTime(2020, 1, 1, 22, 59, 0)},
		{name: "23:00", time: wallTime(2020, 1, 1, 23, 0, 0)},
		{name: "23:59", time: wallTime(2020, 1, 1, 23, 59, 0)},
		{name: "00:00", time: wallTime(2020, 1, 2, 0, 0, 0)},
		{name: "00:01", time: wallTime(2020, 1, 2, 0, 1, 0)},
	}
	results := make([]*model.ChartData, 0, len(times))
	for _, item := range times {
		chart, err := calculator.Calculate(context.Background(), CalculatorInput{Gender: 1, TrueSolarTime: item.time, DayBoundaryRule: Midnight00})
		if err != nil {
			t.Fatalf("%s: %v", item.name, err)
		}
		results = append(results, chart)
	}
	day := func(chart *model.ChartData) string { return chart.Pillars.Day.Stem + chart.Pillars.Day.Branch }
	if day(results[0]) != day(results[1]) || day(results[1]) != day(results[2]) {
		t.Fatalf("23:00 must not change the day pillar under MIDNIGHT_00")
	}
	if day(results[2]) == day(results[3]) || day(results[3]) != day(results[4]) {
		t.Fatalf("day pillar must change at 00:00: before=%s after=%s", day(results[2]), day(results[3]))
	}
}

type SolarTimeEngineFunc func(SolarTimeInput) (*SolarTimeResult, error)

func (f SolarTimeEngineFunc) Calculate(input SolarTimeInput) (*SolarTimeResult, error) {
	return f(input)
}
