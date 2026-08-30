package tengod

const RuleVersionV1 = "ten-god-rule-v1"

type Contribution struct {
	Code, Source, Position, Symbol, Element, TenGod, Category string
	Score                                                     float64
}

type Relation struct {
	Code, Type, Element, Reason string
	Positions, Symbols          []string
	Score                       float64
	Transformed                 bool
}

type Input struct {
	DayStem, DayElement string
	Contributions       []Contribution
	Relations           []Relation
}

type GodScore struct {
	Code, Name, Category     string
	RawScore, EffectiveScore float64
	Ratio                    float64
	Rank                     int
	Visible, Rooted          bool
}

type CategoryScore struct {
	Category                                            string
	RawScore, RelationAdjustment, EffectiveScore, Ratio float64
	Rank                                                int
}

type Evidence struct {
	Code, Source, Position, TenGod, Category, Reason string
	Symbols                                          []string
	RawScore, Adjustment                             float64
}

type Result struct {
	RuleVersion, DayStem, DayElement string
	TotalScore, Concentration        float64
	DominantGods, SecondaryGods      []string
	MissingGods, VisibleGods         []string
	RootedGods                       []string
	Gods                             []GodScore
	Categories                       []CategoryScore
	Evidence                         []Evidence
	Warnings                         []string
}
