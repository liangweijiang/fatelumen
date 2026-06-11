package repository

import (
	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
)

// OrderRepo 订单数据访问层。
type OrderRepo struct {
	db *gorm.DB
}

func NewOrderRepo(db *gorm.DB) *OrderRepo {
	return &OrderRepo{db: db}
}

// Create 创建订单。
func (r *OrderRepo) Create(order *model.Order) error {
	return r.db.Create(order).Error
}

// GetByID 按 ID 查找订单，带 userID 归属校验防越权。
func (r *OrderRepo) GetByID(id, userID uint64) (*model.Order, error) {
	var order model.Order
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetBySessionID 按 Stripe Checkout Session ID 查找订单（回调用）。
func (r *OrderRepo) GetBySessionID(sessionID string) (*model.Order, error) {
	var order model.Order
	err := r.db.Where("provider_ref = ?", sessionID).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// UpdateStatusAndTxn 回调成功后更新订单状态、交易 ID 和渠道原始数据。
func (r *OrderRepo) UpdateStatusAndTxn(id uint64, status, providerTxnID string, meta []byte) error {
	updates := map[string]interface{}{
		"status":          status,
		"provider_txn_id": providerTxnID,
	}
	if meta != nil {
		updates["provider_meta"] = model.JSONRaw(meta)
	}
	return r.db.Model(&model.Order{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateProviderRef 下单后回写渠道会话 ID。
func (r *OrderRepo) UpdateProviderRef(id uint64, sessionID string) error {
	return r.db.Model(&model.Order{}).Where("id = ?", id).
		Update("provider_ref", sessionID).Error
}

// ListByUser 列出用户所有订单。
func (r *OrderRepo) ListByUser(userID uint64) ([]model.Order, error) {
	var orders []model.Order
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&orders).Error
	return orders, err
}
