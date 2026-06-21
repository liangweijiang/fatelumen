package repository

import (
	"errors"
	"strings"
	"time"

	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
)

// ErrDuplicateEvent webhook 去重哨兵错误。
var ErrDuplicateEvent = errors.New("duplicate webhook event")

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

// FindActiveByUserReport 查询同一用户对同一报告的活跃订单（created/pending/paid），按创建时间倒序。
// excluded/failed/refunded 不会被返回。
func (r *OrderRepo) FindActiveByUserReport(userID, reportID uint64) ([]model.Order, error) {
	var orders []model.Order
	err := r.db.
		Where("user_id = ? AND report_id = ? AND status IN ?", userID, reportID, []string{
			model.OrderStatusCreated,
			model.OrderStatusPending,
			model.OrderStatusPaid,
		}).
		Order("created_at DESC").
		Find(&orders).Error
	return orders, err
}

// FulfillPaidOrder 在单事务内完成：webhook 去重插入 + 订单 Transit(paid) + 报告解锁。
// 任一步失败整体回滚；去重唯一索引冲突视为重复事件，返回 ErrDuplicateEvent。
func (r *OrderRepo) FulfillPaidOrder(provider, eventID string, orderID uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. 插入 webhook_event 记录（去重）
		evt := &model.ProcessedWebhookEvent{
			Provider:  provider,
			EventID:   eventID,
			EventType: "checkout.session.completed",
			CreatedAt: time.Now(),
		}
		if err := tx.Create(evt).Error; err != nil {
			if isDuplicateKeyError(err) {
				return ErrDuplicateEvent
			}
			return err
		}

		// 2. 查订单
		var order model.Order
		if err := tx.Where("id = ?", orderID).First(&order).Error; err != nil {
			return err
		}

		// 订单已 paid → 幂等（新 event_id 但订单已被前序事件解锁）
		if order.Status == model.OrderStatusPaid {
			return nil
		}

		// 3. 状态机流转
		if err := order.Transit(model.OrderStatusPaid); err != nil {
			return err
		}
		if err := tx.Model(&model.Order{}).Where("id = ?", orderID).Updates(map[string]interface{}{
			"status":     order.Status,
			"updated_at": order.UpdatedAt,
		}).Error; err != nil {
			return err
		}

		// 4. 按订单类型履约
		switch order.Type {
		case "credits":
			// 积分套餐：给用户加积分余额 + 记一笔流水（同事务保证幂等）
			if order.CreditsGranted > 0 {
				if err := tx.Model(&model.User{}).Where("id = ?", order.UserID).
					Update("credits", gorm.Expr("credits + ?", order.CreditsGranted)).Error; err != nil {
					return err
				}
				var u model.User
				if err := tx.Select("credits").Where("id = ?", order.UserID).First(&u).Error; err != nil {
					return err
				}
				refID := orderID
				ledger := &model.CreditLedger{
					UserID:       order.UserID,
					Delta:        order.CreditsGranted,
					BalanceAfter: u.Credits,
					Reason:       "purchase",
					RefID:        &refID,
					CreatedAt:    time.Now(),
				}
				if err := tx.Create(ledger).Error; err != nil {
					return err
				}
			}
		default:
			// report 单：解锁报告
			if order.ReportID > 0 {
				if err := tx.Model(&model.Report{}).Where("id = ?", order.ReportID).Updates(map[string]interface{}{
					"paid":     true,
					"order_id": orderID,
				}).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// isDuplicateKeyError 判断唯一键冲突（兼容 MySQL 与 SQLite）。
func isDuplicateKeyError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") ||
		strings.Contains(msg, "UNIQUE constraint")
}
