package model

// B0 contract version constants. They identify immutable report snapshots and
// are intentionally independent from database migration versions.
const (
	ReportInputSchemaVersion = "report-input-v1"
	ChartSnapshotVersion     = "chart-snapshot-v3"
	FactsSchemaVersion       = "interpretation-facts-v3"
)

// ReportInputSnapshot is the privacy-minimized input captured when a full
// report is created. Authentication, contact and payment credentials must
// never be copied into this structure.
type ReportInputSnapshot struct {
	CalendarType    int     `json:"calendar_type"`
	Year            int     `json:"year"`
	Month           int     `json:"month"`
	Day             int     `json:"day"`
	Hour            int     `json:"hour"`
	Minute          int     `json:"minute"`
	IsLeapMonth     bool    `json:"is_leap_month"`
	Gender          int     `json:"gender"`
	CountryCode     string  `json:"country_code,omitempty"`
	RegionCode      string  `json:"region_code,omitempty"`
	PlaceID         string  `json:"place_id,omitempty"`
	Longitude       float64 `json:"longitude"`
	Latitude        float64 `json:"latitude"`
	TimezoneID      string  `json:"timezone_id"`
	Locale          string  `json:"locale"`
	ProfileMode     string  `json:"profile_mode,omitempty"`
	TargetProfileID *uint64 `json:"target_profile_id,omitempty"`
	SchemaVersion   string  `json:"schema_version"`
}

// TimeCalculationSnapshot captures every deterministic time conversion used
// before lunar-go is called.
type TimeCalculationSnapshot struct {
	LocalCivilTime             string  `json:"local_civil_time"`
	TimezoneID                 string  `json:"timezone_id"`
	HistoricalUTCOffsetSeconds int     `json:"historical_utc_offset_seconds"`
	StandardUTCOffsetSeconds   int     `json:"standard_utc_offset_seconds"`
	DSTApplied                 bool    `json:"dst_applied"`
	DSTOffsetSeconds           int     `json:"dst_offset_seconds"`
	StandardMeridian           float64 `json:"standard_meridian"`
	LongitudeCorrectionMinutes float64 `json:"longitude_correction_minutes"`
	EquationOfTimeMinutes      float64 `json:"equation_of_time_minutes"`
	LocalStandardTime          string  `json:"local_standard_time"`
	MeanSolarTime              string  `json:"mean_solar_time"`
	TrueSolarTime              string  `json:"true_solar_time"`
	CrossedDateBoundary        bool    `json:"crossed_date_boundary"`
	DayBoundaryRule            string  `json:"day_boundary_rule"`
	SolarAlgorithmVersion      string  `json:"solar_algorithm_version"`
	LocationDatabaseVersion    string  `json:"location_database_version"`
	TimezoneDatabaseVersion    string  `json:"timezone_database_version"`
}

// ChartSnapshot is the immutable chart used to derive one report.
type ChartSnapshot struct {
	ChartHash          string    `json:"chart_hash"`
	ChartSchemaVersion string    `json:"chart_schema_version"`
	EngineVersion      string    `json:"engine_version"`
	LunarGoVersion     string    `json:"lunar_go_version"`
	Data               ChartData `json:"data"`
}

// FactEvidence makes every derived judgment inspectable in the admin panel.
type FactEvidence struct {
	RuleCode string   `json:"rule_code"`
	Source   string   `json:"source"`
	Pillars  []string `json:"pillars,omitempty"`
	Symbols  []string `json:"symbols,omitempty"`
	Reason   string   `json:"reason"`
}

// ScoredFact represents a deterministic result with traceable evidence.
type ScoredFact struct {
	Code      string         `json:"code"`
	Level     string         `json:"level,omitempty"`
	Score     *float64       `json:"score,omitempty"`
	Values    []string       `json:"values,omitempty"`
	Evidence  []FactEvidence `json:"evidence"`
	Conflicts []string       `json:"conflicts,omitempty"`
}

// DayMasterStrengthFact embeds the complete, versioned calculation trace.
// The generic fields remain aligned with ScoredFact for admin presentation,
// while Analysis is the authoritative deterministic result.
type DayMasterStrengthFact struct {
	Code       string              `json:"code"`
	Level      string              `json:"level"`
	Score      *float64            `json:"score"`
	Values     []string            `json:"values"`
	Evidence   []FactEvidence      `json:"evidence"`
	Conflicts  []string            `json:"conflicts"`
	Analysis   StrengthAnalysis    `json:"analysis"`
	AnalysisV2 *StrengthV2Analysis `json:"analysis_v2,omitempty"`
}

// RelationFact describes a deterministic stem or branch relationship.
type RelationFact struct {
	Code     string         `json:"code"`
	Type     string         `json:"type"`
	Pillars  []string       `json:"pillars"`
	Symbols  []string       `json:"symbols"`
	Evidence []FactEvidence `json:"evidence"`
}

// RuleMatch records one rule evaluated while building report facts.
type RuleMatch struct {
	RuleCode string         `json:"rule_code"`
	Module   string         `json:"module"`
	Matched  bool           `json:"matched"`
	Priority int            `json:"priority"`
	Evidence []FactEvidence `json:"evidence,omitempty"`
}

// ChapterFacts is the allow-list of facts supplied to one report chapter.
type ChapterFacts struct {
	ChapterKey string   `json:"chapter_key"`
	FactCodes  []string `json:"fact_codes"`
}

// InterpretationFactVersions pins every dependency that can affect a report.
type InterpretationFactVersions struct {
	BaziBaseDataVersion     string `json:"bazi_base_data_version"`
	TenGodRuleVersion       string `json:"ten_god_rule_version"`
	ElementPowerRuleVersion string `json:"element_power_rule_version"`
	StrengthV2RuleVersion   string `json:"strength_v2_rule_version"`
	TenGodEffectiveVersion  string `json:"ten_god_effective_version"`
	ClimateRuleVersion      string `json:"climate_rule_version"`
	PatternRuleVersion      string `json:"pattern_rule_version"`
	DiseaseRuleVersion      string `json:"disease_rule_version"`
	MediationRuleVersion    string `json:"mediation_rule_version"`
	UsefulGodRuleVersion    string `json:"useful_god_rule_version"`
	InputSchemaVersion      string `json:"input_schema_version"`
	ChartSchemaVersion      string `json:"chart_schema_version"`
	FactsSchemaVersion      string `json:"facts_schema_version"`
	RuleSetVersion          string `json:"rule_set_version"`
	PromptVersion           string `json:"prompt_version"`
	SolarAlgorithmVersion   string `json:"solar_algorithm_version"`
	LocationDatabaseVersion string `json:"location_database_version"`
	TimezoneDatabaseVersion string `json:"timezone_database_version"`
	LunarGoVersion          string `json:"lunar_go_version"`
}

// InterpretationFacts is the only structured fact package that a full-report
// prompt may consume. Quick readings may later use a strict subset of it.
type InterpretationFacts struct {
	FactsHash         string                     `json:"facts_hash"`
	Input             ReportInputSnapshot        `json:"input"`
	TimeCalculation   TimeCalculationSnapshot    `json:"time_calculation"`
	Chart             ChartSnapshot              `json:"chart"`
	ElementStrength   []ScoredFact               `json:"element_strength"`
	DayMasterStrength DayMasterStrengthFact      `json:"day_master_strength"`
	TenGodStructure   []ScoredFact               `json:"ten_god_structure"`
	StemRelations     []RelationFact             `json:"stem_relations"`
	BranchRelations   []RelationFact             `json:"branch_relations"`
	PatternCandidates []ScoredFact               `json:"pattern_candidates"`
	Climate           *ClimateAnalysis           `json:"climate,omitempty"`
	Pattern           *PatternAnalysis           `json:"pattern,omitempty"`
	Disease           *DiseaseAnalysis           `json:"disease,omitempty"`
	Mediation         *MediationAnalysis         `json:"mediation,omitempty"`
	UsefulGod         *UsefulGodAnalysis         `json:"useful_god,omitempty"`
	FavorableElements ScoredFact                 `json:"favorable_elements"`
	LuckCycles        []LuckCycle                `json:"luck_cycles"`
	AnnualFortunes    []AnnualFortune            `json:"annual_fortunes"`
	ChapterFacts      []ChapterFacts             `json:"chapter_facts"`
	RuleMatches       []RuleMatch                `json:"rule_matches"`
	Warnings          []string                   `json:"warnings"`
	Versions          InterpretationFactVersions `json:"versions"`
}
