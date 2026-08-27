package birthchart

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

type precisionFixture struct {
	Metadata struct {
		MaximumErrorSeconds float64 `json:"maximum_error_seconds"`
	} `json:"metadata"`
	Cases []struct {
		Name                     string  `json:"name"`
		LocalStandardTime        string  `json:"local_standard_time"`
		Longitude                float64 `json:"longitude"`
		StandardUTCOffsetSeconds int     `json:"standard_utc_offset_seconds"`
	} `json:"cases"`
}

func TestSolarTimePrecisionAgainstNOAAJulianCenturyReference(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(source), "..", "..", "testdata", "birthchart", "precision_reference_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture precisionFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Metadata.MaximumErrorSeconds <= 0 || len(fixture.Cases) == 0 {
		t.Fatal("precision fixture must define cases and a positive threshold")
	}

	engine := DefaultSolarTimeEngine{}
	for _, tc := range fixture.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			local, err := time.Parse("2006-01-02T15:04:05", tc.LocalStandardTime)
			if err != nil {
				t.Fatal(err)
			}
			result, err := engine.Calculate(SolarTimeInput{
				LocalStandardTime: local, StandardUTCOffset: tc.StandardUTCOffsetSeconds, Longitude: tc.Longitude,
			})
			if err != nil {
				t.Fatal(err)
			}

			referenceEOT := noaaJulianCenturyEquationOfTime(local, tc.StandardUTCOffsetSeconds)
			referenceTrueSolar := local.Add(minutesDuration((tc.Longitude-float64(tc.StandardUTCOffsetSeconds)/240)*4 + referenceEOT))
			errorSeconds := math.Abs(result.TrueSolarTime.Sub(referenceTrueSolar).Seconds())
			if errorSeconds > fixture.Metadata.MaximumErrorSeconds {
				t.Fatalf("true solar error %.3fs exceeds %.3fs: got=%s reference=%s production_eot=%.6f reference_eot=%.6f",
					errorSeconds, fixture.Metadata.MaximumErrorSeconds, result.TrueSolarTime.Format(time.RFC3339Nano), referenceTrueSolar.Format(time.RFC3339Nano), result.EquationOfTimeMinute, referenceEOT)
			}
			if solarBoundaryBucket(result.TrueSolarTime) != solarBoundaryBucket(referenceTrueSolar) {
				t.Fatalf("reference crosses a day or shichen boundary: got=%s reference=%s", result.TrueSolarTime, referenceTrueSolar)
			}
			t.Logf("error=%.3fs true_solar=%s reference=%s", errorSeconds, result.TrueSolarTime.Format("2006-01-02 15:04:05"), referenceTrueSolar.Format("2006-01-02 15:04:05"))
		})
	}
}

// noaaJulianCenturyEquationOfTime follows NOAA's Julian-century solar equations.
// It intentionally does not share code with the production day-angle approximation.
func noaaJulianCenturyEquationOfTime(localStandardTime time.Time, standardUTCOffsetSeconds int) float64 {
	utc := time.Date(localStandardTime.Year(), localStandardTime.Month(), localStandardTime.Day(), localStandardTime.Hour(), localStandardTime.Minute(), localStandardTime.Second(), localStandardTime.Nanosecond(), time.UTC).
		Add(-time.Duration(standardUTCOffsetSeconds) * time.Second)
	julianDay := float64(utc.UnixNano())/float64(24*time.Hour) + 2440587.5
	century := (julianDay - 2451545.0) / 36525.0
	geomMeanLong := math.Mod(280.46646+century*(36000.76983+century*0.0003032), 360)
	geomMeanAnomaly := 357.52911 + century*(35999.05029-0.0001537*century)
	eccentricity := 0.016708634 - century*(0.000042037+0.0000001267*century)
	meanObliquity := 23 + (26+(21.448-century*(46.815+century*(0.00059-century*0.001813)))/60)/60
	omega := 125.04 - 1934.136*century
	obliquityCorrection := meanObliquity + 0.00256*math.Cos(referenceDegreesToRadians(omega))
	y := math.Pow(math.Tan(referenceDegreesToRadians(obliquityCorrection)/2), 2)
	l := referenceDegreesToRadians(geomMeanLong)
	m := referenceDegreesToRadians(geomMeanAnomaly)
	equation := y*math.Sin(2*l) - 2*eccentricity*math.Sin(m) + 4*eccentricity*y*math.Sin(m)*math.Cos(2*l) - 0.5*y*y*math.Sin(4*l) - 1.25*eccentricity*eccentricity*math.Sin(2*m)
	return 4 * referenceRadiansToDegrees(equation)
}

func solarBoundaryBucket(t time.Time) string {
	// The project uses midnight for the day boundary and two-hour shichen buckets
	// beginning at 01:00, with the zi hour spanning 23:00 through 00:59.
	hourBucket := (t.Hour() + 1) / 2
	if hourBucket == 12 {
		hourBucket = 0
	}
	return t.Format("2006-01-02") + "/" + string(rune('A'+hourBucket))
}

func referenceDegreesToRadians(value float64) float64 { return value * math.Pi / 180 }
func referenceRadiansToDegrees(value float64) float64 { return value * 180 / math.Pi }
