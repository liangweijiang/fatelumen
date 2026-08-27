package birthchart

import (
	"math"
	"time"
)

const SolarAlgorithmVersion = "eot-noaa-julian-century-v2"

type DefaultSolarTimeEngine struct{}

func (DefaultSolarTimeEngine) Calculate(input SolarTimeInput) (*SolarTimeResult, error) {
	if input.Longitude < -180 || input.Longitude > 180 || input.StandardUTCOffset < -14*3600 || input.StandardUTCOffset > 14*3600 {
		return nil, ErrInvalidInput
	}
	standardMeridian := float64(input.StandardUTCOffset) / 3600 * 15
	longitudeCorrection := (input.Longitude - standardMeridian) * 4
	meanSolarTime := input.LocalStandardTime.Add(minutesDuration(longitudeCorrection))
	eot := equationOfTimeMinutes(input.LocalStandardTime, input.StandardUTCOffset)
	return &SolarTimeResult{
		StandardMeridian:          standardMeridian,
		LongitudeCorrectionMinute: longitudeCorrection,
		MeanSolarTime:             meanSolarTime,
		EquationOfTimeMinute:      eot,
		TrueSolarTime:             meanSolarTime.Add(minutesDuration(eot)),
		AlgorithmVersion:          SolarAlgorithmVersion,
	}, nil
}

func equationOfTimeMinutes(localStandardTime time.Time, standardUTCOffsetSeconds int) float64 {
	utc := time.Date(localStandardTime.Year(), localStandardTime.Month(), localStandardTime.Day(), localStandardTime.Hour(), localStandardTime.Minute(), localStandardTime.Second(), localStandardTime.Nanosecond(), time.UTC).
		Add(-time.Duration(standardUTCOffsetSeconds) * time.Second)
	century := (julianDay(utc) - 2451545.0) / 36525.0
	geomMeanLong := math.Mod(280.46646+century*(36000.76983+century*0.0003032), 360)
	if geomMeanLong < 0 {
		geomMeanLong += 360
	}
	geomMeanAnomaly := 357.52911 + century*(35999.05029-0.0001537*century)
	eccentricity := 0.016708634 - century*(0.000042037+0.0000001267*century)
	meanObliquity := 23 + (26+(21.448-century*(46.815+century*(0.00059-century*0.001813)))/60)/60
	omega := 125.04 - 1934.136*century
	obliquityCorrection := meanObliquity + 0.00256*math.Cos(degreesToRadians(omega))
	y := math.Pow(math.Tan(degreesToRadians(obliquityCorrection)/2), 2)
	l := degreesToRadians(geomMeanLong)
	m := degreesToRadians(geomMeanAnomaly)
	equation := y*math.Sin(2*l) - 2*eccentricity*math.Sin(m) + 4*eccentricity*y*math.Sin(m)*math.Cos(2*l) - 0.5*y*y*math.Sin(4*l) - 1.25*eccentricity*eccentricity*math.Sin(2*m)
	return 4 * radiansToDegrees(equation)
}

func julianDay(t time.Time) float64 {
	year, month, day := t.Date()
	y := year
	m := int(month)
	if m <= 2 {
		y--
		m += 12
	}
	a := y / 100
	b := 2 - a + a/4
	fraction := (float64(t.Hour()) + float64(t.Minute())/60 + float64(t.Second())/3600 + float64(t.Nanosecond())/float64(time.Second)/3600) / 24
	return math.Floor(365.25*float64(y+4716)) + math.Floor(30.6001*float64(m+1)) + float64(day) + fraction + float64(b) - 1524.5
}

func minutesDuration(minutes float64) time.Duration {
	return time.Duration(math.Round(minutes * float64(time.Minute)))
}

func degreesToRadians(value float64) float64 { return value * math.Pi / 180 }
func radiansToDegrees(value float64) float64 { return value * 180 / math.Pi }
