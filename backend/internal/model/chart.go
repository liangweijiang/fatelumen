package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// ==================== ChartData 命盘 JSON 结构 ====================

// Pillar 单柱（年/月/日/时）。
type Pillar struct {
	Stem          string   `json:"stem"`
	Branch        string   `json:"branch"`
	StemElement   string   `json:"stem_element"`
	BranchElement string   `json:"branch_element"`
	TenGodStem    string   `json:"ten_god_stem"`
	TenGodHidden  []string `json:"ten_god_hidden"`
	HiddenStems   []string `json:"hidden_stems"`
	NaYin         string   `json:"nayin"`
}

// Pillars 四柱。
type Pillars struct {
	Year  Pillar `json:"year"`
	Month Pillar `json:"month"`
	Day   Pillar `json:"day"`
	Hour  Pillar `json:"hour"`
}

// LuckCycle 大运。
type LuckCycle struct {
	GanZhi    string `json:"ganzhi"`
	StartAge  int    `json:"start_age"`
	StartYear int    `json:"start_year"`
	Element   string `json:"element,omitempty"`
}

// DayMaster 日主。
type DayMaster struct {
	Stem    string `json:"stem"`
	Element string `json:"element"`
	YinYang string `json:"yin_yang"`
}

// Strength 身强身弱判定。
type Strength struct {
	Level       string            `json:"level"` // "strong" / "weak" / "balanced"
	Score       int               `json:"score"`
	Favorable   []string          `json:"favorable"`
	Unfavorable []string          `json:"unfavorable"`
	Analysis    *StrengthAnalysis `json:"analysis,omitempty"`
}

// StrengthAnalysis records the deterministic evidence behind a strength result.
// It is versioned independently so historical charts remain explainable when
// weights or thresholds are adjusted later.
type StrengthAnalysis struct {
	RuleVersion    string                 `json:"rule_version"`
	DayElement     string                 `json:"day_element"`
	MonthScore     float64                `json:"month_score"`
	SupportScore   float64                `json:"support_score"`
	RestraintScore float64                `json:"restraint_score"`
	SupportRatio   float64                `json:"support_ratio"`
	RootLevel      string                 `json:"root_level"`
	Pattern        string                 `json:"pattern"`
	PatternSubtype string                 `json:"pattern_subtype,omitempty"`
	FalseFollowing bool                   `json:"false_following"`
	Contributions  []StrengthContribution `json:"contributions"`
	Relations      []StrengthRelation     `json:"relations"`
	Warnings       []string               `json:"warnings,omitempty"`
}

type StrengthContribution struct {
	Code       string  `json:"code"`
	Source     string  `json:"source"`
	Position   string  `json:"position,omitempty"`
	Symbol     string  `json:"symbol,omitempty"`
	Element    string  `json:"element"`
	TenGod     string  `json:"ten_god,omitempty"`
	Category   string  `json:"category"`
	Score      float64 `json:"score"`
	Adjustment float64 `json:"adjustment,omitempty"`
}

type StrengthRelation struct {
	Code        string   `json:"code"`
	Type        string   `json:"type"`
	Positions   []string `json:"positions"`
	Symbols     []string `json:"symbols"`
	Element     string   `json:"element,omitempty"`
	Score       float64  `json:"score"`
	Transformed bool     `json:"transformed,omitempty"`
	Reason      string   `json:"reason"`
}

// TenGodAnalysis records the deterministic ten-god distribution used by a
// full report. Favorable-element judgments are deliberately outside this
// contract and will be supplied by a separately versioned rule engine.
type TenGodAnalysis struct {
	RuleVersion   string                `json:"rule_version"`
	DayStem       string                `json:"day_stem"`
	DayElement    string                `json:"day_element"`
	TotalScore    float64               `json:"total_score"`
	DominantGods  []string              `json:"dominant_gods"`
	SecondaryGods []string              `json:"secondary_gods"`
	MissingGods   []string              `json:"missing_gods"`
	VisibleGods   []string              `json:"visible_gods"`
	RootedGods    []string              `json:"rooted_gods"`
	Concentration float64               `json:"concentration"`
	Gods          []TenGodScore         `json:"gods"`
	Categories    []TenGodCategoryScore `json:"categories"`
	Evidence      []TenGodEvidence      `json:"evidence"`
	Warnings      []string              `json:"warnings,omitempty"`
}

type TenGodScore struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Category       string  `json:"category"`
	RawScore       float64 `json:"raw_score"`
	EffectiveScore float64 `json:"effective_score"`
	Ratio          float64 `json:"ratio"`
	Rank           int     `json:"rank"`
	Visible        bool    `json:"visible"`
	Rooted         bool    `json:"rooted"`
}

type TenGodCategoryScore struct {
	Category           string  `json:"category"`
	RawScore           float64 `json:"raw_score"`
	RelationAdjustment float64 `json:"relation_adjustment"`
	EffectiveScore     float64 `json:"effective_score"`
	Ratio              float64 `json:"ratio"`
	Rank               int     `json:"rank"`
}

type TenGodEvidence struct {
	Code       string   `json:"code"`
	Source     string   `json:"source"`
	Position   string   `json:"position,omitempty"`
	Symbols    []string `json:"symbols,omitempty"`
	TenGod     string   `json:"ten_god,omitempty"`
	Category   string   `json:"category,omitempty"`
	RawScore   float64  `json:"raw_score,omitempty"`
	Adjustment float64  `json:"adjustment,omitempty"`
	Reason     string   `json:"reason"`
}

type ElementPowerVector struct {
	Wood  float64 `json:"wood"`
	Fire  float64 `json:"fire"`
	Earth float64 `json:"earth"`
	Metal float64 `json:"metal"`
	Water float64 `json:"water"`
}

type ElementSeasonAnalysis struct {
	Branch       string             `json:"branch"`
	NextBranch   string             `json:"next_branch"`
	Progress     float64            `json:"progress"`
	Coefficients ElementPowerVector `json:"coefficients"`
}

type ElementPowerContribution struct {
	Code                 string  `json:"code"`
	Position             string  `json:"position"`
	Symbol               string  `json:"symbol"`
	Element              string  `json:"element"`
	HiddenLevel          string  `json:"hidden_level,omitempty"`
	BaseWeight           float64 `json:"base_weight"`
	HiddenRatio          float64 `json:"hidden_ratio"`
	RawPower             float64 `json:"raw_power"`
	SeasonCoefficient    float64 `json:"season_coefficient"`
	VisibilityMultiplier float64 `json:"visibility_multiplier"`
	SeasonalPower        float64 `json:"seasonal_power"`
}

type ElementRootEvidence struct {
	Position    string  `json:"position"`
	Branch      string  `json:"branch"`
	Stem        string  `json:"stem"`
	Level       string  `json:"level"`
	HiddenRatio float64 `json:"hidden_ratio"`
	Quality     float64 `json:"quality"`
	Power       float64 `json:"power"`
}

type ElementInteractionEvidence struct {
	Iteration     int     `json:"iteration"`
	Type          string  `json:"type"`
	SourceElement string  `json:"source_element"`
	TargetElement string  `json:"target_element"`
	SourceBefore  float64 `json:"source_before"`
	TargetBefore  float64 `json:"target_before"`
	Amount        float64 `json:"amount"`
	Efficiency    float64 `json:"efficiency"`
	ContactFactor float64 `json:"contact_factor"`
	RatioFactor   float64 `json:"ratio_factor"`
}

type ElementStructureEvidence struct {
	Code          string             `json:"code"`
	Type          string             `json:"type"`
	TargetElement string             `json:"target_element,omitempty"`
	State         string             `json:"state"`
	Reason        string             `json:"reason"`
	Positions     []string           `json:"positions"`
	Symbols       []string           `json:"symbols"`
	Confidence    float64            `json:"confidence"`
	TransferRate  float64            `json:"transfer_rate"`
	Before        ElementPowerVector `json:"before"`
	After         ElementPowerVector `json:"after"`
}

// ElementPowerAnalysis is the independently versioned V2 power-engine trace.
type ElementPowerAnalysis struct {
	RuleVersion    string                       `json:"rule_version"`
	Season         ElementSeasonAnalysis        `json:"season"`
	RawPower       ElementPowerVector           `json:"raw_power"`
	SeasonalPower  ElementPowerVector           `json:"seasonal_power"`
	EffectivePower ElementPowerVector           `json:"effective_power"`
	EffectiveRatio ElementPowerVector           `json:"effective_ratio"`
	Contributions  []ElementPowerContribution   `json:"contributions"`
	Roots          []ElementRootEvidence        `json:"roots"`
	RootPower      float64                      `json:"root_power"`
	Interactions   []ElementInteractionEvidence `json:"interactions"`
	Structures     []ElementStructureEvidence   `json:"structures"`
	Warnings       []string                     `json:"warnings,omitempty"`
}

type StrengthPatternCandidate struct {
	Type        string   `json:"type"`
	Subtype     string   `json:"subtype,omitempty"`
	Alternative string   `json:"alternative,omitempty"`
	Matched     bool     `json:"matched"`
	Confidence  float64  `json:"confidence"`
	Evidence    []string `json:"evidence"`
	RejectedBy  []string `json:"rejected_by,omitempty"`
}

type StrengthV2Trace struct {
	Rule   string             `json:"rule"`
	Result string             `json:"result"`
	Reason string             `json:"reason"`
	Score  float64            `json:"score,omitempty"`
	Values map[string]float64 `json:"values,omitempty"`
}

type StrengthV2Analysis struct {
	RuleVersion  string                     `json:"rule_version"`
	Level        string                     `json:"level"`
	BaseScore    float64                    `json:"base_score"`
	Score        float64                    `json:"score"`
	Confidence   float64                    `json:"confidence"`
	Support      float64                    `json:"support"`
	Pressure     float64                    `json:"pressure"`
	SupportRatio float64                    `json:"support_ratio"`
	DeLing       float64                    `json:"de_ling"`
	DeDi         float64                    `json:"de_di"`
	DeShi        float64                    `json:"de_shi"`
	RootPower    float64                    `json:"root_power"`
	Patterns     []StrengthPatternCandidate `json:"patterns"`
	Trace        []StrengthV2Trace          `json:"trace"`
	Warnings     []string                   `json:"warnings,omitempty"`
}

type AnalysisEvidence struct {
	Rule   string             `json:"rule"`
	Reason string             `json:"reason"`
	Values map[string]float64 `json:"values,omitempty"`
}

type ClimateAnalysis struct {
	RuleVersion      string             `json:"rule_version"`
	Temperature      float64            `json:"temperature"`
	Moisture         float64            `json:"moisture"`
	TemperatureLevel string             `json:"temperature_level"`
	MoistureLevel    string             `json:"moisture_level"`
	Candidates       []string           `json:"candidates"`
	Evidence         []AnalysisEvidence `json:"evidence"`
	Warnings         []string           `json:"warnings,omitempty"`
}

type PatternReviewCandidate struct {
	Code         string   `json:"code"`
	State        string   `json:"state"`
	Confidence   float64  `json:"confidence"`
	Requirements []string `json:"requirements"`
	RejectedBy   []string `json:"rejected_by,omitempty"`
	Evidence     []string `json:"evidence"`
}

type PatternAnalysis struct {
	RuleVersion string                   `json:"rule_version"`
	Primary     string                   `json:"primary,omitempty"`
	Candidates  []PatternReviewCandidate `json:"candidates"`
	Warnings    []string                 `json:"warnings,omitempty"`
}

type DiseaseItem struct {
	Code              string   `json:"code"`
	Level             string   `json:"level"`
	Severity          float64  `json:"severity"`
	CandidateElements []string `json:"candidate_elements,omitempty"`
	Evidence          []string `json:"evidence"`
}

type DiseaseAnalysis struct {
	RuleVersion string        `json:"rule_version"`
	Primary     *DiseaseItem  `json:"primary,omitempty"`
	Secondary   []DiseaseItem `json:"secondary"`
	Minor       []DiseaseItem `json:"minor"`
	All         []DiseaseItem `json:"all"`
	Warnings    []string      `json:"warnings,omitempty"`
}

type MediationConflict struct {
	Controller      string   `json:"controller"`
	Controlled      string   `json:"controlled"`
	Bridge          string   `json:"bridge"`
	State           string   `json:"state"`
	ControllerPower float64  `json:"controller_power"`
	ControlledPower float64  `json:"controlled_power"`
	BridgePower     float64  `json:"bridge_power"`
	Ratio           float64  `json:"ratio"`
	Confidence      float64  `json:"confidence"`
	Evidence        []string `json:"evidence"`
}

type MediationAnalysis struct {
	RuleVersion string              `json:"rule_version"`
	FlowScore   float64             `json:"flow_score"`
	Conflicts   []MediationConflict `json:"conflicts"`
	Warnings    []string            `json:"warnings,omitempty"`
}

type UsefulGodScoreBreakdown struct {
	FuYi            float64 `json:"fu_yi"`
	Disease         float64 `json:"disease"`
	Pattern         float64 `json:"pattern"`
	Climate         float64 `json:"climate"`
	Flow            float64 `json:"flow"`
	Availability    float64 `json:"availability"`
	MarginalUtility float64 `json:"marginal_utility"`
	SideEffects     float64 `json:"side_effects"`
}

type UsefulGodSimulation struct {
	Delta            float64 `json:"delta"`
	Health           float64 `json:"health"`
	MarginalUtility  float64 `json:"marginal_utility"`
	StrengthScore    float64 `json:"strength_score"`
	Temperature      float64 `json:"temperature"`
	Moisture         float64 `json:"moisture"`
	FlowScore        float64 `json:"flow_score"`
	PatternIntegrity float64 `json:"pattern_integrity"`
	DiseaseSeverity  float64 `json:"disease_severity"`
}

type UsefulGodCandidate struct {
	Element         string                  `json:"element"`
	Role            string                  `json:"role"`
	Score           float64                 `json:"score"`
	Confidence      float64                 `json:"confidence"`
	Availability    float64                 `json:"availability"`
	Valid           bool                    `json:"valid"`
	ResolvesPrimary bool                    `json:"resolves_primary"`
	PrimaryRequired bool                    `json:"primary_required"`
	Breakdown       UsefulGodScoreBreakdown `json:"breakdown"`
	Simulations     []UsefulGodSimulation   `json:"simulations"`
	Reasons         []string                `json:"reasons"`
	RejectReasons   []string                `json:"reject_reasons,omitempty"`
	OptimalRange    [2]float64              `json:"optimal_range"`
}

type UsefulGodAnalysis struct {
	RuleVersion string               `json:"rule_version"`
	Primary     string               `json:"primary,omitempty"`
	Secondary   string               `json:"secondary,omitempty"`
	Favorable   []string             `json:"favorable"`
	Taboo       []string             `json:"taboo"`
	Enemy       []string             `json:"enemy"`
	Neutral     []string             `json:"neutral"`
	Confidence  float64              `json:"confidence"`
	Candidates  []UsefulGodCandidate `json:"candidates"`
	Warnings    []string             `json:"warnings,omitempty"`
}

// CurrentYearFortune 本年流年。
type CurrentYearFortune struct {
	Year    int    `json:"year"`
	Stem    string `json:"stem"`
	Branch  string `json:"branch"`
	Element string `json:"element"`
}

// AnnualFortune 流年表单项，由 lunar-go 通过大运流年确定性计算。
type AnnualFortune struct {
	Year               int    `json:"year"`
	Age                int    `json:"age,omitempty"`
	GanZhi             string `json:"ganzhi"`
	Stem               string `json:"stem"`
	Branch             string `json:"branch"`
	Element            string `json:"element"`
	LuckCycleGanZhi    string `json:"luck_cycle_ganzhi,omitempty"`
	LuckCycleStartAge  int    `json:"luck_cycle_start_age,omitempty"`
	LuckCycleStartYear int    `json:"luck_cycle_start_year,omitempty"`
}

// AnnualCalendarYear is public, person-independent sexagenary calendar data.
// Personal fields such as age and luck-cycle ownership remain in AnnualFortune.
type AnnualCalendarYear struct {
	Year          int       `gorm:"primaryKey;autoIncrement:false" json:"year"`
	GanZhi        string    `gorm:"type:varchar(8);not null" json:"ganzhi"`
	Stem          string    `gorm:"type:varchar(4);not null" json:"stem"`
	Branch        string    `gorm:"type:varchar(4);not null" json:"branch"`
	StemElement   string    `gorm:"type:varchar(4);not null" json:"stem_element"`
	BranchElement string    `gorm:"type:varchar(4);not null" json:"branch_element"`
	StemYinYang   string    `gorm:"type:varchar(4);not null" json:"stem_yin_yang"`
	BranchYinYang string    `gorm:"type:varchar(4);not null" json:"branch_yin_yang"`
	Zodiac        string    `gorm:"type:varchar(8);not null" json:"zodiac"`
	CycleIndex    int       `gorm:"not null" json:"cycle_index"`
	DataVersion   string    `gorm:"type:varchar(64);not null" json:"data_version"`
	CreatedAt     time.Time `json:"created_at"`
}

func (AnnualCalendarYear) TableName() string { return "annual_calendar_years" }

type AnnualFortuneRange struct {
	StartYear     int    `json:"start_year"`
	EndYear       int    `json:"end_year"`
	Count         int    `json:"count"`
	Source        string `json:"source"`
	DataVersion   string `json:"data_version"`
	SelectionRule string `json:"selection_rule"`
}

// ChartMeta 排盘元信息。
type ChartMeta struct {
	SolarDate       string              `json:"solar_date"`
	LunarDate       string              `json:"lunar_date"`
	Zodiac          string              `json:"zodiac"`
	Gender          string              `json:"gender"`
	CalcLib         string              `json:"calc_lib"`
	CalcVersion     string              `json:"calc_version"`
	TimeCalculation TimeCalculationMeta `json:"time_calculation"`
}

type TimeCalculationMeta struct {
	TimezoneID                string  `json:"timezone_id"`
	HistoricalUTCOffset       int     `json:"historical_utc_offset_seconds"`
	StandardUTCOffset         int     `json:"standard_utc_offset_seconds"`
	DSTApplied                bool    `json:"dst_applied"`
	DSTOffset                 int     `json:"dst_offset_seconds"`
	Longitude                 float64 `json:"longitude"`
	Latitude                  float64 `json:"latitude"`
	StandardMeridian          float64 `json:"standard_meridian"`
	LongitudeCorrectionMinute float64 `json:"longitude_correction_minutes"`
	EquationOfTimeMinute      float64 `json:"equation_of_time_minutes"`
	LocalCivilTime            string  `json:"local_civil_time"`
	LocalStandardTime         string  `json:"local_standard_time"`
	MeanSolarTime             string  `json:"mean_solar_time"`
	TrueSolarTime             string  `json:"true_solar_time"`
	Mode                      string  `json:"mode"`
	DayBoundaryRule           string  `json:"day_boundary_rule"`
	SolarAlgorithmVersion     string  `json:"solar_algorithm_version"`
	EngineVersion             string  `json:"engine_version"`
}

// ChartData 完整命盘 JSON（存储于 charts.chart_data）。
type ChartData struct {
	Pillars            Pillars               `json:"pillars"`
	DayMaster          DayMaster             `json:"day_master"`
	FiveElementsCount  map[string]int        `json:"five_elements_count"`
	Strength           Strength              `json:"strength"`
	ElementPower       *ElementPowerAnalysis `json:"element_power,omitempty"`
	StrengthV2         *StrengthV2Analysis   `json:"strength_v2,omitempty"`
	TenGodEffective    *TenGodAnalysis       `json:"ten_god_effective,omitempty"`
	TenGodAnalysis     *TenGodAnalysis       `json:"ten_god_analysis,omitempty"`
	Climate            *ClimateAnalysis      `json:"climate,omitempty"`
	Pattern            *PatternAnalysis      `json:"pattern,omitempty"`
	Disease            *DiseaseAnalysis      `json:"disease,omitempty"`
	Mediation          *MediationAnalysis    `json:"mediation,omitempty"`
	UsefulGod          *UsefulGodAnalysis    `json:"useful_god,omitempty"`
	LuckCycles         []LuckCycle           `json:"luck_cycles"`
	CurrentYearFortune *CurrentYearFortune   `json:"current_year_fortune,omitempty"`
	AnnualFortunes     []AnnualFortune       `json:"annual_fortunes,omitempty"`
	AnnualFortuneRange *AnnualFortuneRange   `json:"annual_fortune_range,omitempty"`
	HourUnknown        bool                  `json:"hour_unknown"`
	Meta               ChartMeta             `json:"meta"`
}

// Value 实现 driver.Valuer，序列化为 JSON。
func (c ChartData) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan 实现 sql.Scanner，反序列化 JSON。
func (c *ChartData) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, c)
}

// ==================== Charts 排盘结果表模型 ====================

// Chart 排盘结果（确定性，可按 profile 哈希缓存复用）。
type Chart struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ProfileID uint64    `gorm:"not null;index" json:"profile_id"`
	ChartHash string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"chart_hash"`
	ChartData ChartData `gorm:"type:json;not null" json:"chart_data"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
}

func (Chart) TableName() string { return "charts" }
