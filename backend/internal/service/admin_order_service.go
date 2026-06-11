package service

import (
	"context"
	"encoding/json"
	"time"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
)

// ---------- DTOs ----------

// AdminOrderItem 订单列表项（不含 ProviderMeta，避免列表过大）。
type AdminOrderItem struct {
	ID          uint64    `json:"id"`
	UserID      uint64    `json:"user_id"`
	ReportID    uint64    `json:"report_id"`
	Type        string    `json:"type"`
	SKU         string    `json:"sku"`
	AmountCents int       `json:"amount_cents"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	Provider    string    `json:"provider"`
	CreatedAt   time.Time `json:"created_at"`
}

// AdminOrderDetail 订单详情（含 ProviderMeta 原始回调 JSON，对账用）。
type AdminOrderDetail struct {
	ID             uint64          `json:"id"`
	UserID         uint64          `json:"user_id"`
	ReportID       uint64          `json:"report_id"`
	Type           string          `json:"type"`
	SKU            string          `json:"sku"`
	AmountCents    int             `json:"amount_cents"`
	Currency       string          `json:"currency"`
	Status         string          `json:"status"`
	Provider       string          `json:"provider"`
	ProviderRef    string          `json:"provider_ref"`
	ProviderTxnID  string          `json:"provider_txn_id"`
	ProviderMeta   json.RawMessage `json:"provider_meta"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// AdminOrdersPage 分页结果。
type AdminOrdersPage struct {
	Items    []AdminOrderItem `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// ---------- private interfaces ----------

type adminOrderStore interface {
	AdminListOrders(filter repository.OrderFilter, limit, offset int) ([]model.Order, int64, error)
	AdminGetOrderByID(id uint64) (*model.Order, error)
}

// ---------- AdminOrderService ----------

type AdminOrderService struct {
	orderRepo adminOrderStore
}

func NewAdminOrderService(orderRepo *repository.OrderRepo) *AdminOrderService {
	return &AdminOrderService{orderRepo: orderRepo}
}

// ListOrders admin 分页订单列表。
func (s *AdminOrderService) ListOrders(ctx context.Context, status string, userID uint64, page, pageSize int) (*AdminOrdersPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	orders, total, err := s.orderRepo.AdminListOrders(repository.OrderFilter{
		Status: status,
		UserID: userID,
	}, pageSize, offset)
	if err != nil {
		logger.FromCtx(ctx).Error("admin list orders failed", "err", err)
		return nil, err
	}

	items := make([]AdminOrderItem, len(orders))
	for i, o := range orders {
		items[i] = AdminOrderItem{
			ID:          o.ID,
			UserID:      o.UserID,
			ReportID:    o.ReportID,
			Type:        o.Type,
			SKU:         o.SKU,
			AmountCents: o.AmountCents,
			Currency:    o.Currency,
			Status:      o.Status,
			Provider:    o.Provider,
			CreatedAt:   o.CreatedAt,
		}
	}
	return &AdminOrdersPage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetOrderDetail admin 订单详情。
func (s *AdminOrderService) GetOrderDetail(ctx context.Context, orderID uint64) (*AdminOrderDetail, error) {
	order, err := s.orderRepo.AdminGetOrderByID(orderID)
	if err != nil {
		logger.FromCtx(ctx).Error("admin get order detail failed",
			"err", err,
			"order_id", orderID,
		)
		return nil, err
	}

	var meta json.RawMessage
	if len(order.ProviderMeta) > 0 {
		meta = json.RawMessage(order.ProviderMeta)
	}

	return &AdminOrderDetail{
		ID:            order.ID,
		UserID:        order.UserID,
		ReportID:      order.ReportID,
		Type:          order.Type,
		SKU:           order.SKU,
		AmountCents:   order.AmountCents,
		Currency:      order.Currency,
		Status:        order.Status,
		Provider:      order.Provider,
		ProviderRef:   order.ProviderRef,
		ProviderTxnID: order.ProviderTxnID,
		ProviderMeta:  meta,
		CreatedAt:     order.CreatedAt,
		UpdatedAt:     order.UpdatedAt,
	}, nil
}
