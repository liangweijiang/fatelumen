package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// ---------- Full Report JSON ----------

// Chapter 单章（12 章之一）。
type Chapter struct {
	No            int         `json:"no"`
	Key           string      `json:"key"`
	Title         string      `json:"title"`
	Body          string      `json:"body"`
	StrengthScore int         `json:"strength_score,omitempty"`
	Cycles        []CycleNote `json:"cycles,omitempty"`
	Years         []YearNote  `json:"years,omitempty"`
	Tags          []string    `json:"tags,omitempty"`
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

// YearlyFortuneItem 流年运势单项。
type YearlyFortuneItem struct {
	Year int    `json:"year"`
	Note string `json:"note"`
}

// ReportContent 深度报告 JSON 结构（§9.2）。
// 各章节字段均为专业命理解读，禁止绝对化、医疗、投资、寿命断言。
type ReportContent struct {
	Locale        string               `json:"locale"`
	SummaryLine   string               `json:"summary_line"`
	Summary       string               `json:"summary"`
	Personality   string               `json:"personality"`
	Career        string               `json:"career"`
	Relationship  string               `json:"relationship"`
	Health        string               `json:"health"`
	YearlyFortune []YearlyFortuneItem  `json:"yearly_fortune"`
	Suggestions   []string             `json:"suggestions"`
	Chapters      []Chapter            `json:"chapters,omitempty"`
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

// 报告状态机：pending → processing → done，失败置 failed。
const (
	ReportStatusPending    = "pending"
	ReportStatusProcessing = "processing"
	ReportStatusDone       = "done"
	ReportStatusFailed     = "failed"
)

// ---------- Report API View ----------

// Report 是用户侧完整报告接口的兼容返回结构。
//
// 正式报告持久化仅使用 FullReport 领域模型。本结构刻意不包含 GORM
// 映射标签和 TableName 方法，避免旧 reports 表被再次创建或误写。
type Report struct {
	ID         uint64        `json:"id"`
	UserID     uint64        `json:"user_id"`
	ProfileID  uint64        `json:"profile_id"`
	ChartID    uint64        `json:"chart_id"`
	OrderID    *uint64       `json:"order_id"`
	Locale     string        `json:"locale"`
	Status     string        `json:"status"`
	PayMethod  string        `json:"pay_method"`
	Content    ReportContent `json:"content"`
	PDFURL     string        `json:"pdf_url"`
	ErrorMsg   string        `json:"error_msg"`
	RetryCount int           `json:"retry_count"`
	Paid       bool          `json:"paid"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}
