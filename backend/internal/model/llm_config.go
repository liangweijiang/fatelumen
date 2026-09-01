package model

import "time"

// LLMProviderConfig stores an administrator-managed OpenAI-compatible endpoint.
// APIKeyCiphertext is never serialized by handlers.
type LLMProviderConfig struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Code             string    `gorm:"type:varchar(64);not null;index" json:"code"`
	Name             string    `gorm:"type:varchar(100);not null" json:"name"`
	BaseURL          string    `gorm:"type:varchar(500);not null" json:"base_url"`
	APIKeyCiphertext string    `gorm:"type:text;not null" json:"-"`
	APIKeyHint       string    `gorm:"type:varchar(32);not null" json:"api_key_hint"`
	Enabled          bool      `gorm:"not null;default:true;index" json:"enabled"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (LLMProviderConfig) TableName() string { return "llm_provider_configs" }

// LLMModelConfig belongs to exactly one provider. ProviderID is intentionally
// part of the unique key because different gateways may expose the same model ID.
type LLMModelConfig struct {
	ID         uint64            `gorm:"primaryKey;autoIncrement" json:"id"`
	ProviderID uint64            `gorm:"not null;uniqueIndex:uk_llm_model_provider_model;index" json:"provider_id"`
	Provider   LLMProviderConfig `gorm:"foreignKey:ProviderID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"provider"`
	Name       string            `gorm:"type:varchar(120);not null" json:"name"`
	ModelID    string            `gorm:"column:model_id;type:varchar(200);not null;uniqueIndex:uk_llm_model_provider_model" json:"model_id"`
	Priority   int               `gorm:"not null;default:100;index" json:"priority"`
	MaxRetries int               `gorm:"not null;default:3" json:"max_retries"`
	Enabled    bool              `gorm:"not null;default:true;index" json:"enabled"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

func (LLMModelConfig) TableName() string { return "llm_model_configs" }
