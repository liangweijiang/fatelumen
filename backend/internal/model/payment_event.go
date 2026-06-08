package model

import "time"

// PaymentEvent 支付回调事件去重表（所有渠道共用，保证 Webhook 幂等）。
type PaymentEvent struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Provider    string    `gorm:"type:varchar(24);not null" json:"provider"`
	EventID     string    `gorm:"column:event_id;type:varchar(191);not null;uniqueIndex:uk_provider_event" json:"event_id"`
	EventType   string    `gorm:"column:event_type;type:varchar(64);not null" json:"event_type"`
	OrderID     *uint64   `gorm:"comment:关联订单" json:"order_id"`
	ProcessedAt time.Time `gorm:"not null" json:"processed_at"`
}

func (PaymentEvent) TableName() string { return "payment_events" }

// CreditLedger 积分流水。
type CreditLedger struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint64    `gorm:"not null;index" json:"user_id"`
	Delta        int       `gorm:"not null;comment:正=充值 负=消费" json:"delta"`
	BalanceAfter int       `gorm:"not null" json:"balance_after"`
	Reason       string    `gorm:"type:varchar(64);not null" json:"reason"`
	RefID        *uint64   `gorm:"comment:关联 order_id 或 report_id" json:"ref_id"`
	CreatedAt    time.Time `gorm:"not null" json:"created_at"`
}

func (CreditLedger) TableName() string { return "credit_ledger" }
