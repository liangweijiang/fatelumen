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

// MarkPaid 在事务中完成订单支付状态流转。
// 先用 Transit 校验 created→paid 合法性，再写入 ProviderTxnID 和原始回调数据。
func (r *OrderRepo) MarkPaid(orderID uint64, providerTxnID string, meta []byte) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Where("id = ?", orderID).First(&order).Error; err != nil {
			return err
		}
		if err := order.Transit(model.OrderStatusPaid); err != nil {
			return err
		}
		updates := map[string]interface{}{
			"status":          order.Status,
			"provider_txn_id": providerTxnID,
			"updated_at":      order.UpdatedAt,
		}
		if meta != nil {
			updates["provider_meta"] = model.JSONRaw(meta)
		}
		return tx.Model(&model.Order{}).Where("id = ?", orderID).Updates(updates).Error
	})
}

// ListByUser 列出用户所有订单。
func (r *OrderRepo) ListByUser(userID uint64) ([]model.Order, error) {
	var orders []model.Order
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&orders).Error
	return orders, err
}

// CountByUser 统计用户订单数。
func (r *OrderRepo) CountByUser(userID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&model.Order{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// OrderFilter admin 订单筛选条件。
type OrderFilter struct {
	Status string
	UserID uint64
}

// AdminListOrders admin 分页订单列表（可筛选 status/userID）。
func (r *OrderRepo) AdminListOrders(filter OrderFilter, limit, offset int) ([]model.Order, int64, error) {
	query := r.db.Model(&model.Order{})
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var orders []model.Order
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&orders).Error
	return orders, total, err
}

// AdminGetOrderByID admin 查单（不限 user_id，管理员可看任意订单）。
func (r *OrderRepo) AdminGetOrderByID(id uint64) (*model.Order, error) {
	var order model.Order
	err := r.db.First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}
