package model

import (
	"database/sql/driver"
	"time"
)

// Order 订单（支付渠道无关，provider 字段标识具体渠道）。
type Order struct {
	ID             uint64        `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         uint64        `gorm:"not null;index" json:"user_id"`
	Type           string        `gorm:"type:varchar(16);not null;comment:report/credits" json:"type"`
	SKU            string        `gorm:"type:varchar(32);not null" json:"sku"`
	AmountCents    int           `gorm:"not null" json:"amount_cents"`
	Currency       string        `gorm:"type:varchar(8);not null;default:'usd'" json:"currency"`
	CreditsGranted int           `gorm:"not null;default:0" json:"credits_granted"`
	Provider       string        `gorm:"type:varchar(24);not null" json:"provider"`
	ProviderRef    string        `gorm:"type:varchar(191)" json:"provider_ref"`
	ProviderTxnID  string        `gorm:"type:varchar(191)" json:"provider_txn_id"`
	ProviderMeta   JSONRaw       `gorm:"type:json" json:"provider_meta"`
	Status         string        `gorm:"type:varchar(16);not null;default:'created'" json:"status"`
	CreatedAt      time.Time     `gorm:"not null" json:"created_at"`
	UpdatedAt      time.Time     `gorm:"not null" json:"updated_at"`
}

func (Order) TableName() string { return "orders" }

// JSONRaw 存储原始 JSON 字节。
type JSONRaw []byte

func (j JSONRaw) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return []byte(j), nil
}

func (j *JSONRaw) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	*j = make([]byte, len(b))
	copy(*j, b)
	return nil
}
