package strength

const RuleVersionV1 = "strength-basis-v1"

type Pillar struct{ Stem, Branch string }
type Input struct{ Year, Month, Day, Hour Pillar }

type Contribution struct {
	Code, Source, Position, Symbol, Element, TenGod, Category string
	Score, Adjustment                                         float64
}

type Relation struct {
	Code, Type, Element, Reason string
	Positions, Symbols          []string
	Score                       float64
	Transformed                 bool
}

type Result struct {
	RuleVersion, DayElement, RootLevel                     string
	MonthScore, SupportScore, RestraintScore, SupportRatio float64
	Contributions                                          []Contribution
	Relations                                              []Relation
	Warnings                                               []string
}
