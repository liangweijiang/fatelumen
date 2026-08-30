package strength

import "fatelumen/backend/internal/bazi/basedata"

type hiddenStem struct {
	Stem   string
	Weight float64
}

type RuleSet struct {
	Version                                                                    string
	PositionWeight                                                             map[string]float64
	MonthScore                                                                 map[string]map[string]float64
	HiddenStems                                                                map[string][]hiddenStem
	StrongThreshold, WeakThreshold, FollowStrongThreshold, FollowWeakThreshold float64
}

func RuleV1() RuleSet {
	base := basedata.V1()
	weights := map[string]float64{}
	for _, v := range base.PositionWeights {
		weights[v.Code] = v.Weight
	}
	hidden := map[string][]hiddenStem{}
	for _, b := range base.Branches {
		for _, h := range b.HiddenStems {
			hidden[b.Code] = append(hidden[b.Code], hiddenStem{h.Stem, h.Weight})
		}
	}
	return RuleSet{
		Version:        RuleVersionV1,
		PositionWeight: weights, MonthScore: base.MonthScores, HiddenStems: hidden,
		StrongThreshold: base.Threshold("strong"), WeakThreshold: base.Threshold("weak"), FollowStrongThreshold: base.Threshold("follow_strong"), FollowWeakThreshold: base.Threshold("follow_weak"),
	}
}
