package model

import "time"

// ProcessedWebhookEvent 已处理的 Webhook 事件（幂等去重表）。
// provider + event_id 联合唯一索引，靠 INSERT 冲突判重。
type ProcessedWebhookEvent struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Provider  string    `gorm:"size:24;not null;uniqueIndex:idx_provider_event" json:"provider"`
	EventID   string    `gorm:"size:191;not null;uniqueIndex:idx_provider_event" json:"event_id"`
	EventType string    `gorm:"size:64" json:"event_type"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
}

func (ProcessedWebhookEvent) TableName() string { return "processed_webhook_events" }
