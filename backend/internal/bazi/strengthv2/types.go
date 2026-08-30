package strengthv2

const RuleVersionV2 = "day-master-strength-v2.0"

type CategoryPower struct{ Peer, Resource, Output, Wealth, Officer float64 }

type Structure struct {
	Code, Type, TargetElement, State string
	Confidence                       float64
}

type Input struct {
	DayElement, MonthBranch                      string
	SeasonCoefficient, SeasonProgress, RootPower float64
	Categories                                   CategoryPower
	ElementRatios                                map[string]float64
	Structures                                   []Structure
	HourUnknown                                  bool
}

type PatternCandidate struct {
	Type, Subtype, Alternative string
	Matched                    bool
	Confidence                 float64
	Evidence, RejectedBy       []string
}

type Trace struct {
	Rule, Result, Reason string
	Score                float64
	Values               map[string]float64
}

type Result struct {
	RuleVersion, Level              string
	BaseScore, Score, Confidence    float64
	Support, Pressure, SupportRatio float64
	DeLing, DeDi, DeShi             float64
	RootPower                       float64
	Patterns                        []PatternCandidate
	Trace                           []Trace
	Warnings                        []string
}
