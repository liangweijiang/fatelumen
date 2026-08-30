package climate

import (
	"fmt"
	"math"
)

const RuleVersionV1 = "climate-v1.0"

type RuleSet struct {
	Version                   string
	MonthTemperature          map[string]float64
	MonthMoisture             map[string]float64
	FireWaterTemperatureScale float64
	WaterFireMoistureScale    float64
}

type Input struct {
	MonthBranch  string
	ElementRatio map[string]float64
}

type Evidence struct {
	Rule, Reason string
	Values       map[string]float64
}

type Result struct {
	RuleVersion                     string
	Temperature, Moisture           float64
	TemperatureLevel, MoistureLevel string
	Candidates                      []string
	Evidence                        []Evidence
	Warnings                        []string
}

func RuleV1() RuleSet {
	return RuleSet{Version: RuleVersionV1,
		MonthTemperature:          map[string]float64{"亥": -65, "子": -85, "丑": -75, "寅": -40, "卯": -15, "辰": 5, "巳": 55, "午": 85, "未": 75, "申": 40, "酉": 15, "戌": -5},
		MonthMoisture:             map[string]float64{"亥": 65, "子": 85, "丑": 60, "寅": 15, "卯": 10, "辰": 45, "巳": -55, "午": -75, "未": -60, "申": -10, "酉": -15, "戌": -50},
		FireWaterTemperatureScale: .35, WaterFireMoistureScale: .30}
}

func Evaluate(in Input, rules RuleSet) (Result, error) {
	base, ok := rules.MonthTemperature[in.MonthBranch]
	if !ok {
		return Result{}, fmt.Errorf("unsupported month branch %q", in.MonthBranch)
	}
	moistureBase := rules.MonthMoisture[in.MonthBranch]
	fire, water := in.ElementRatio["火"], in.ElementRatio["水"]
	temp := clamp(base+(fire-water)*rules.FireWaterTemperatureScale, -100, 100)
	moisture := clamp(moistureBase+(water-fire)*rules.WaterFireMoistureScale, -100, 100)
	r := Result{RuleVersion: rules.Version, Temperature: round(temp), Moisture: round(moisture), TemperatureLevel: axisLevel(temp, "cold", "hot"), MoistureLevel: axisLevel(moisture, "dry", "wet")}
	if temp <= -40 {
		r.Candidates = append(r.Candidates, "火")
	} else if temp >= 40 {
		r.Candidates = append(r.Candidates, "水")
	}
	if moisture <= -40 {
		r.Candidates = appendUnique(r.Candidates, "水")
	} else if moisture >= 40 {
		r.Candidates = appendUnique(r.Candidates, "火")
	}
	r.Evidence = []Evidence{{Rule: rules.Version + ".month_temperature", Reason: "month branch base temperature", Values: map[string]float64{"base": base}}, {Rule: rules.Version + ".global_adjustment", Reason: "fire/water effective ratio adjustment", Values: map[string]float64{"fire": fire, "water": water, "temperature": r.Temperature, "moisture": r.Moisture}}}
	r.Warnings = []string{"climate_global_adjustment_coefficients_require_domain_calibration"}
	return r, nil
}

func axisLevel(v float64, negative, positive string) string {
	a := math.Abs(v)
	severity := "normal"
	if a > 70 {
		severity = "severe"
	} else if a >= 40 {
		severity = "obvious"
	}
	if severity == "normal" {
		return severity
	}
	if v < 0 {
		return severity + "_" + negative
	}
	return severity + "_" + positive
}
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
func round(v float64) float64 { return math.Round(v*100) / 100 }
func appendUnique(v []string, x string) []string {
	for _, e := range v {
		if e == x {
			return v
		}
	}
	return append(v, x)
}
