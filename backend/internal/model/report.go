package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// ---------- Full Report JSON ----------

// Chapter 单章（12 章之一）。
type Chapter struct {
	No            int           `json:"no"`
	Key           string        `json:"key"`
	Title         string        `json:"title"`
	Body          string        `json:"body"`
	StrengthScore int           `json:"strength_score,omitempty"`
	Cycles        []CycleNote   `json:"cycles,omitempty"`
	Years         []YearNote    `json:"years,omitempty"`
	Tags          []string      `json:"tags,omitempty"`
}

// CycleNote 大运备注。
type CycleNote struct {
	GanZhi    string `json:"ganzhi"`
	StartAge  int    `json:"start_age"`
	StartYear int    `json:"start_year"`
	Note      string `json:"note"`
}

// YearNote 流年备注。
type YearNote struct {
	Year   int    `json:"year"`
	GanZhi string `json:"ganzhi"`
	Note   string `json:"note"`
}

// ReportContent 完整报告 12 章 JSON。
type ReportContent struct {
	Locale      string    `json:"locale"`
	SummaryLine string    `json:"summary_line"`
	Chapters    []Chapter `json:"chapters"`
}

// Value 实现 driver.Valuer。
func (c ReportContent) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan 实现 sql.Scanner。
func (c *ReportContent) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, c)
}

// ---------- Report Model ----------

// Report 完整测算报告（异步状态机）。
type Report struct {
	ID         uint64        `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint64        `gorm:"not null;index" json:"user_id"`
	ProfileID  uint64        `gorm:"not null" json:"profile_id"`
	ChartID    uint64        `gorm:"not null" json:"chart_id"`
	OrderID    *uint64       `gorm:"comment:关联订单" json:"order_id"`
	Locale     string        `gorm:"type:varchar(8);not null;default:'en'" json:"locale"`
	Status     string        `gorm:"type:varchar(16);not null;default:'pending';index" json:"status"`
	PayMethod  string        `gorm:"type:varchar(16);not null;comment:order/credit" json:"pay_method"`
	Content    ReportContent `gorm:"type:json" json:"content"`
	PDFURL     string        `gorm:"type:varchar(512)" json:"pdf_url"`
	ErrorMsg   string        `gorm:"type:varchar(512)" json:"error_msg"`
	RetryCount int           `gorm:"not null;default:0" json:"retry_count"`
	CreatedAt  time.Time     `gorm:"not null" json:"created_at"`
	UpdatedAt  time.Time     `gorm:"not null" json:"updated_at"`
}

func (Report) TableName() string { return "reports" }
