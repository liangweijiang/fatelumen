package elementpower

type RuleSet struct {
	Version                                                           string
	HiddenRatios                                                      map[string]float64
	PositionWeights                                                   map[string]float64
	SeasonMatrix                                                      map[string]map[string]float64
	GenerationRate                                                    float64
	GenerationEfficiency                                              float64
	ControlRate                                                       float64
	Iterations                                                        int
	TransformationFull, TransformationPartial, UntransformedRetention float64
	PunishmentRetention, HarmRetention, BreakRetention                float64
	StructureEnabled                                                  bool
}

func RuleV2() RuleSet { r := RuleV1(); r.Version = RuleVersionV2; r.StructureEnabled = true; return r }

func RuleV1() RuleSet {
	return RuleSet{
		Version:         RuleVersionV1,
		HiddenRatios:    map[string]float64{"main": .7, "middle": .2, "residual": .1, "single": 1, "double_main": .7, "double_residual": .3},
		PositionWeights: map[string]float64{"year_stem": .85, "month_stem": 1.15, "day_stem": 1, "hour_stem": .95, "year_branch": 1, "month_branch": 1.8, "day_branch": 1.35, "hour_branch": 1.05},
		SeasonMatrix: map[string]map[string]float64{
			"寅": {"木": 1.55, "火": 1.20, "土": .55, "金": .65, "水": .95}, "卯": {"木": 1.65, "火": 1.30, "土": .45, "金": .60, "水": .90},
			"辰": {"木": 1.20, "火": 1.05, "土": 1.35, "金": .75, "水": .85}, "巳": {"木": .90, "火": 1.60, "土": 1.25, "金": .50, "水": .60},
			"午": {"木": .80, "火": 1.70, "土": 1.30, "金": .45, "水": .55}, "未": {"木": .80, "火": 1.25, "土": 1.45, "金": .70, "水": .60},
			"申": {"木": .55, "火": .65, "土": .95, "金": 1.60, "水": 1.25}, "酉": {"木": .45, "火": .60, "土": .90, "金": 1.70, "水": 1.30},
			"戌": {"木": .65, "火": .80, "土": 1.45, "金": 1.15, "水": .65}, "亥": {"木": 1.25, "火": .55, "土": .70, "金": .95, "水": 1.60},
			"子": {"木": 1.30, "火": .45, "土": .65, "金": .90, "水": 1.70}, "丑": {"木": .70, "火": .65, "土": 1.45, "金": 1.05, "水": 1.15},
		},
		GenerationRate: .12, GenerationEfficiency: .85, ControlRate: .18, Iterations: 2,
		TransformationFull: .75, TransformationPartial: .50, UntransformedRetention: .90,
		PunishmentRetention: .94, HarmRetention: .96, BreakRetention: .97,
	}
}
