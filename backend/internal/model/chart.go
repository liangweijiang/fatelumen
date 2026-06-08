package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// ==================== ChartData 命盘 JSON 结构 ====================

// Pillar 单柱（年/月/日/时）。
type Pillar struct {
	Stem         string   `json:"stem"`
	Branch       string   `json:"branch"`
	StemElement  string   `json:"stem_element"`
	BranchElement string  `json:"branch_element"`
	TenGodStem   string   `json:"ten_god_stem"`
	TenGodHidden []string `json:"ten_god_hidden"`
	HiddenStems  []string `json:"hidden_stems"`
	NaYin        string   `json:"nayin"`
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
	Level       string   `json:"level"` // "strong" / "weak" / "balanced"
	Score       int      `json:"score"`
	Favorable   []string `json:"favorable"`
	Unfavorable []string `json:"unfavorable"`
}

// CurrentYearFortune 本年流年。
type CurrentYearFortune struct {
	Year    int    `json:"year"`
	Stem    string `json:"stem"`
	Branch  string `json:"branch"`
	Element string `json:"element"`
}

// ChartMeta 排盘元信息。
type ChartMeta struct {
	SolarDate   string `json:"solar_date"`
	LunarDate   string `json:"lunar_date"`
	Gender      string `json:"gender"`
	CalcLib     string `json:"calc_lib"`
	CalcVersion string `json:"calc_version"`
}

// ChartData 完整命盘 JSON（存储于 charts.chart_data）。
type ChartData struct {
	Pillars            Pillars             `json:"pillars"`
	DayMaster          DayMaster           `json:"day_master"`
	FiveElementsCount  map[string]int      `json:"five_elements_count"`
	Strength           Strength            `json:"strength"`
	LuckCycles         []LuckCycle         `json:"luck_cycles"`
	CurrentYearFortune *CurrentYearFortune `json:"current_year_fortune,omitempty"`
	HourUnknown        bool                `json:"hour_unknown"`
	Meta               ChartMeta           `json:"meta"`
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
