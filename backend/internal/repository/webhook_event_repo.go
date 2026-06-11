package repository

import (
	"errors"
	"time"

	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
)

// WebhookEventRepo Webhook 事件去重存储。
type WebhookEventRepo struct {
	db *gorm.DB
}

func NewWebhookEventRepo(db *gorm.DB) *WebhookEventRepo {
	return &WebhookEventRepo{db: db}
}

// MarkProcessed 记录已处理的 Webhook 事件，返回 duplicate 表示是否已处理过。
// 依赖 provider+event_id 唯一索引判重。
func (r *WebhookEventRepo) MarkProcessed(provider, eventID, eventType string) (duplicate bool, err error) {
	evt := &model.ProcessedWebhookEvent{
		Provider:  provider,
		EventID:   eventID,
		EventType: eventType,
		CreatedAt: time.Now(),
	}
	err = r.db.Create(evt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}
