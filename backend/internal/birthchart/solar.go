package birthchart

import (
	"math"
	"time"
)

const SolarAlgorithmVersion = "eot-noaa-approx-v1"

type DefaultSolarTimeEngine struct{}

func (DefaultSolarTimeEngine) Calculate(input SolarTimeInput) (*SolarTimeResult, error) {
	if input.Longitude < -180 || input.Longitude > 180 || input.StandardUTCOffset < -14*3600 || input.StandardUTCOffset > 14*3600 {
		return nil, ErrInvalidInput
	}
	standardMeridian := float64(input.StandardUTCOffset) / 3600 * 15
	longitudeCorrection := (input.Longitude - standardMeridian) * 4
	meanSolarTime := input.LocalStandardTime.Add(minutesDuration(longitudeCorrection))
	eot := equationOfTimeMinutes(input.LocalStandardTime)
	return &SolarTimeResult{
		StandardMeridian:          standardMeridian,
		LongitudeCorrectionMinute: longitudeCorrection,
		MeanSolarTime:             meanSolarTime,
		EquationOfTimeMinute:      eot,
		TrueSolarTime:             meanSolarTime.Add(minutesDuration(eot)),
		AlgorithmVersion:          SolarAlgorithmVersion,
	}, nil
}

func equationOfTimeMinutes(t time.Time) float64 {
	days := 365.0
	if isLeapYear(t.Year()) {
		days = 366
	}
	hour := float64(t.Hour()) + float64(t.Minute())/60 + float64(t.Second())/3600
	gamma := 2 * math.Pi / days * (float64(t.YearDay()-1) + (hour-12)/24)
	return 229.18 * (0.000075 + 0.001868*math.Cos(gamma) - 0.032077*math.Sin(gamma) - 0.014615*math.Cos(2*gamma) - 0.040849*math.Sin(2*gamma))
}

func minutesDuration(minutes float64) time.Duration {
	return time.Duration(math.Round(minutes * float64(time.Minute)))
}

func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}
