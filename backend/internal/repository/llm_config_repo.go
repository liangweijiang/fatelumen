package repository

import (
	"context"
	"fmt"

	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
)

// LLMConfigRepo exposes only the enabled, ordered runtime route catalogue.
// Administrative CRUD remains in the admin handler; report execution uses
// this read boundary so it cannot mutate live configuration.
type LLMConfigRepo struct {
	db *gorm.DB
}

func NewLLMConfigRepo(db *gorm.DB) *LLMConfigRepo { return &LLMConfigRepo{db: db} }

func (r *LLMConfigRepo) ListEnabledRoutes(ctx context.Context) ([]model.LLMModelConfig, error) {
	var rows []model.LLMModelConfig
	err := r.db.WithContext(ctx).
		Joins("Provider").
		Where("llm_model_configs.enabled = ? AND Provider.enabled = ?", true, true).
		Order("llm_model_configs.priority ASC, llm_model_configs.id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list enabled llm routes: %w", err)
	}
	return rows, nil
}
