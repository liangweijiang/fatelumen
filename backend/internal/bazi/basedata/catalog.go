package basedata

import (
	"fmt"
	"sort"
)

const VersionV1 = "bazi-base-data-v1"

type Names struct {
	ZH string `json:"zh"`
	EN string `json:"en"`
	JA string `json:"ja"`
	KO string `json:"ko"`
}
type Element struct {
	Code         string `json:"code"`
	Names        Names  `json:"names"`
	Generates    string `json:"generates"`
	Controls     string `json:"controls"`
	GeneratedBy  string `json:"generated_by"`
	ControlledBy string `json:"controlled_by"`
}
type Stem struct {
	Code    string `json:"code"`
	Element string `json:"element"`
	YinYang string `json:"yin_yang"`
	Order   int    `json:"order"`
	Names   Names  `json:"names"`
}
type HiddenStem struct {
	Stem   string  `json:"stem"`
	Weight float64 `json:"weight"`
	Level  string  `json:"level"`
}
type Branch struct {
	Code        string       `json:"code"`
	Element     string       `json:"element"`
	YinYang     string       `json:"yin_yang"`
	Order       int          `json:"order"`
	Names       Names        `json:"names"`
	HiddenStems []HiddenStem `json:"hidden_stems"`
}
type Relation struct {
	Code    string   `json:"code"`
	Scope   string   `json:"scope"`
	Type    string   `json:"type"`
	Element string   `json:"element,omitempty"`
	Members []string `json:"members"`
	Score   float64  `json:"score"`
	Names   Names    `json:"names"`
}
type TenGodRule struct {
	Code            string `json:"code"`
	ElementRelation string `json:"element_relation"`
	YinYangRelation string `json:"yin_yang_relation"`
	Category        string `json:"category"`
	Names           Names  `json:"names"`
}
type PositionWeight struct {
	Code   string  `json:"code"`
	Weight float64 `json:"weight"`
	Names  Names   `json:"names"`
}
type Threshold struct {
	Code  string  `json:"code"`
	Value float64 `json:"value"`
	Names Names   `json:"names"`
}

type Catalog struct {
	Version         string                        `json:"version"`
	Elements        []Element                     `json:"elements"`
	Stems           []Stem                        `json:"stems"`
	Branches        []Branch                      `json:"branches"`
	Relations       []Relation                    `json:"relations"`
	TenGodRules     []TenGodRule                  `json:"ten_god_rules"`
	PositionWeights []PositionWeight              `json:"position_weights"`
	MonthScores     map[string]map[string]float64 `json:"month_scores"`
	Thresholds      []Threshold                   `json:"thresholds"`
}

func V1() Catalog { return catalogV1() }

func (c Catalog) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("base data version is required")
	}
	if len(c.Elements) != 5 {
		return fmt.Errorf("elements=%d want 5", len(c.Elements))
	}
	if len(c.Stems) != 10 {
		return fmt.Errorf("stems=%d want 10", len(c.Stems))
	}
	if len(c.Branches) != 12 {
		return fmt.Errorf("branches=%d want 12", len(c.Branches))
	}
	elements := map[string]bool{}
	for _, e := range c.Elements {
		elements[e.Code] = true
	}
	for _, e := range c.Elements {
		if !elements[e.Generates] || !elements[e.Controls] || !elements[e.GeneratedBy] || !elements[e.ControlledBy] {
			return fmt.Errorf("element %s relationship is incomplete", e.Code)
		}
	}
	stems := map[string]bool{}
	for _, s := range c.Stems {
		stems[s.Code] = true
		if !elements[s.Element] {
			return fmt.Errorf("stem %s has invalid element %s", s.Code, s.Element)
		}
	}
	branches := map[string]bool{}
	for _, b := range c.Branches {
		branches[b.Code] = true
		sum := 0.0
		for _, h := range b.HiddenStems {
			if !stems[h.Stem] {
				return fmt.Errorf("branch %s references invalid hidden stem %s", b.Code, h.Stem)
			}
			sum += h.Weight
		}
		if sum < .999999 || sum > 1.000001 {
			return fmt.Errorf("branch %s hidden stem weight sum %.4f", b.Code, sum)
		}
	}
	for _, r := range c.Relations {
		for _, member := range r.Members {
			if r.Scope == "stem" && !stems[member] {
				return fmt.Errorf("relation %s references invalid stem %s", r.Code, member)
			}
			if r.Scope == "branch" && !branches[member] {
				return fmt.Errorf("relation %s references invalid branch %s", r.Code, member)
			}
		}
	}
	if len(c.TenGodRules) != 10 {
		return fmt.Errorf("ten-god rules=%d want 10", len(c.TenGodRules))
	}
	tenGodCodes := map[string]bool{}
	tenGodNames := map[string]bool{}
	for _, rule := range c.TenGodRules {
		if tenGodCodes[rule.Code] || tenGodNames[rule.Names.ZH] {
			return fmt.Errorf("duplicate ten-god rule %s/%s", rule.Code, rule.Names.ZH)
		}
		tenGodCodes[rule.Code] = true
		tenGodNames[rule.Names.ZH] = true
	}
	for _, e := range []string{"木", "火", "土", "金", "水"} {
		if len(c.MonthScores[e]) != 5 {
			return fmt.Errorf("month score row %s incomplete", e)
		}
	}
	return nil
}

func (c Catalog) Stem(code string) (Stem, bool) {
	for _, v := range c.Stems {
		if v.Code == code {
			return v, true
		}
	}
	return Stem{}, false
}
func (c Catalog) Branch(code string) (Branch, bool) {
	for _, v := range c.Branches {
		if v.Code == code {
			return v, true
		}
	}
	return Branch{}, false
}
func (c Catalog) PositionWeight(code string) float64 {
	for _, v := range c.PositionWeights {
		if v.Code == code {
			return v.Weight
		}
	}
	return 0
}
func (c Catalog) Threshold(code string) float64 {
	for _, v := range c.Thresholds {
		if v.Code == code {
			return v.Value
		}
	}
	return 0
}

type Summary struct {
	Version         string `json:"version"`
	Valid           bool   `json:"valid"`
	ValidationError string `json:"validation_error,omitempty"`
	Elements        int    `json:"elements"`
	Stems           int    `json:"stems"`
	Branches        int    `json:"branches"`
	Relations       int    `json:"relations"`
	TenGodRules     int    `json:"ten_god_rules"`
}

func (c Catalog) Summary() Summary {
	err := c.Validate()
	s := Summary{Version: c.Version, Valid: err == nil, Elements: len(c.Elements), Stems: len(c.Stems), Branches: len(c.Branches), Relations: len(c.Relations), TenGodRules: len(c.TenGodRules)}
	if err != nil {
		s.ValidationError = err.Error()
	}
	return s
}

func SortRelations(v []Relation) {
	sort.SliceStable(v, func(i, j int) bool {
		if v[i].Type == v[j].Type {
			return v[i].Code < v[j].Code
		}
		return v[i].Type < v[j].Type
	})
}
