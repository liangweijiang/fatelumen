package elementpower

const (
	RuleVersionV1 = "bazi-power-v1.0"
	RuleVersionV2 = "bazi-power-v2.0"
)

type Pillar struct{ Stem, Branch string }

type Input struct {
	Year, Month, Day, Hour Pillar
	SeasonProgress         float64
	LongLifeStages         map[string]string
}

type Vector struct {
	Wood, Fire, Earth, Metal, Water float64
}

type Season struct {
	Branch, NextBranch string
	Progress           float64
	Coefficients       Vector
}

type Contribution struct {
	Code, Position, Symbol, Element, HiddenLevel string
	BaseWeight, HiddenRatio, RawPower            float64
	SeasonCoefficient, VisibilityMultiplier      float64
	SeasonalPower                                float64
}

type Root struct {
	Position, Branch, Stem, Level string
	HiddenRatio, Quality, Power   float64
}

type Interaction struct {
	Iteration                              int
	Type                                   string
	SourceElement, TargetElement           string
	SourceBefore, TargetBefore, Amount     float64
	Efficiency, ContactFactor, RatioFactor float64
}

type StructureRelation struct {
	Code, Type, TargetElement, State, Reason string
	Positions, Symbols                       []string
	Confidence, TransferRate                 float64
	Before, After                            Vector
}

type Trace struct {
	Rule, Source, Reason string
	Values               map[string]float64
}

type Result struct {
	RuleVersion                    string
	Season                         Season
	RawPower, SeasonalPower        Vector
	EffectivePower, EffectiveRatio Vector
	Contributions                  []Contribution
	Roots                          []Root
	RootPower                      float64
	Interactions                   []Interaction
	Structures                     []StructureRelation
	Trace                          []Trace
	Warnings                       []string
}
