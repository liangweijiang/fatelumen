package model

import "time"

// ContentItem is one localized public item. type is knowledge, faq, or case.
// A translation is an independent row so each language can be drafted/published safely.
type ContentItem struct {
	ID          uint64     `gorm:"primaryKey" json:"id"`
	ContentKey  string     `gorm:"type:varchar(64);not null;uniqueIndex:uk_content_translation,priority:1" json:"content_key"`
	Slug        string     `gorm:"type:varchar(128);not null;index" json:"slug"`
	Type        string     `gorm:"type:varchar(16);not null;index" json:"type"`
	Category    string     `gorm:"type:varchar(64)" json:"category"`
	CoverURL    string     `gorm:"type:varchar(512)" json:"cover_url"`
	Locale      string     `gorm:"type:varchar(8);not null;uniqueIndex:uk_content_translation,priority:2" json:"locale"`
	Title       string     `gorm:"type:varchar(255);not null" json:"title"`
	Summary     string     `gorm:"type:text" json:"summary"`
	Tags        JSONRaw    `gorm:"type:json" json:"tags"`
	Markdown    string     `gorm:"type:longtext;not null" json:"markdown"`
	Status      string     `gorm:"type:varchar(16);not null;default:'draft';index" json:"status"`
	PublishedAt *time.Time `json:"published_at"`
	SortOrder   int        `gorm:"not null;default:0" json:"sort_order"`
	Pinned      bool       `gorm:"not null;default:false;index" json:"pinned"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (ContentItem) TableName() string { return "content_items" }

type PricingPlan struct {
	ID                  uint64    `gorm:"primaryKey" json:"id"`
	SKU                 string    `gorm:"type:varchar(64);not null;uniqueIndex:uk_pricing_sku_locale,priority:1" json:"sku"`
	Locale              string    `gorm:"type:varchar(8);not null;uniqueIndex:uk_pricing_sku_locale,priority:2" json:"locale"`
	Title               string    `gorm:"type:varchar(128);not null" json:"title"`
	Summary             string    `gorm:"type:text" json:"summary"`
	Benefits            JSONRaw   `gorm:"type:json" json:"benefits"`
	AmountCents         int       `json:"amount_cents"`
	OriginalAmountCents int       `gorm:"not null;default:0" json:"original_amount_cents"`
	ReportCount         int       `gorm:"not null;default:0" json:"report_count"`
	CreditsGranted      int       `gorm:"not null;default:0" json:"credits_granted"`
	Currency            string    `gorm:"type:varchar(8);not null;default:'usd'" json:"currency"`
	Enabled             bool      `gorm:"not null;default:false" json:"enabled"`
	SortOrder           int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (PricingPlan) TableName() string { return "pricing_plans" }
