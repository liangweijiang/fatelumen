package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/payment"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
)

// ErrReportAlreadyPurchased 报告已通过该用户的其他订单购买。
var ErrReportAlreadyPurchased = errors.New("report already purchased")

type orderStore interface {
	Create(*model.Order) error
	GetByID(id, userID uint64) (*model.Order, error)
	ListByUser(userID uint64) ([]model.Order, error)
	UpdateProviderRef(id uint64, sessionID string) error
	FindActiveByUserReport(userID, reportID uint64) ([]model.Order, error)
}

type orderReportStore interface {
	GetByID(id, userID uint64) (*model.Report, error)
}

// CreateOrderResult 下单返回值，包含订单与结算页跳转 URL。
type CreateOrderResult struct {
	Order       *model.Order `json:"order"`
	CheckoutURL string       `json:"checkout_url"`
}

// OrderService 订单业务编排。
type OrderService struct {
	orderRepo  orderStore
	reportRepo orderReportStore
	pay        payment.PaymentProvider
	priceCents int
	successURL string
	cancelURL  string
}

func NewOrderService(
	orderRepo *repository.OrderRepo,
	reportRepo *repository.ReportRepo,
	pay payment.PaymentProvider,
	priceCents int,
	successURL string,
	cancelURL string,
) *OrderService {
	return &OrderService{
		orderRepo:  orderRepo,
		reportRepo: reportRepo,
		pay:        pay,
		priceCents: priceCents,
		successURL: successURL,
		cancelURL:  cancelURL,
	}
}

// CreateOrder 创建订单并发起支付。
func (s *OrderService) CreateOrder(ctx context.Context, userID, reportID uint64) (*CreateOrderResult, error) {
	// 校验 reportID 属于该用户
	report, err := s.reportRepo.GetByID(reportID, userID)
	if err != nil {
		logger.FromCtx(ctx).Warn("order create failed: report not found or not owned",
			"user_id", userID,
			"report_id", reportID,
			"err", err,
		)
		return nil, fmt.Errorf("report not found or not owned")
	}
	_ = report // 仅校验归属

	// 幂等检查：同一用户对同一报告是否已有活跃订单
	existing, err := s.orderRepo.FindActiveByUserReport(userID, reportID)
	if err != nil {
		logger.FromCtx(ctx).Error("order create failed: lookup existing orders",
			"err", err,
			"user_id", userID,
			"report_id", reportID,
		)
		return nil, err
	}
	for _, o := range existing {
		if o.Status == model.OrderStatusPaid {
			logger.FromCtx(ctx).Warn("order create blocked: report already purchased",
				"user_id", userID,
				"report_id", reportID,
				"existing_order_id", o.ID,
			)
			return nil, ErrReportAlreadyPurchased
		}
	}
	// 存在 created 或 pending 订单：复用（取最近一条）
	if len(existing) > 0 {
		reused := existing[0] // sorted by created_at DESC
		logger.FromCtx(ctx).Info("reuse existing pending order",
			"order_id", reused.ID,
			"user_id", userID,
			"report_id", reportID,
			"status", reused.Status,
		)
		checkoutInput := payment.CheckoutInput{
			OrderID:     reused.ID,
			AmountCents: int64(reused.AmountCents),
			Currency:    reused.Currency,
			ProductName: "Destiny Report",
			SuccessURL:  s.successURL,
			CancelURL:   s.cancelURL,
			Metadata: map[string]string{
				"order_id": strconv.FormatUint(reused.ID, 10),
			},
		}
		result, err := s.pay.CreateCheckout(ctx, checkoutInput)
		if err != nil {
			logger.FromCtx(ctx).Error("payment checkout failed for reused order",
				"err", err,
				"order_id", reused.ID,
				"provider", s.pay.Name(),
			)
			return nil, err
		}
		if err := s.orderRepo.UpdateProviderRef(reused.ID, result.SessionID); err != nil {
			logger.FromCtx(ctx).Error("failed to save provider ref for reused order",
				"err", err,
				"order_id", reused.ID,
				"session_id", result.SessionID,
			)
			return nil, err
		}
		reused.ProviderRef = result.SessionID
		return &CreateOrderResult{
			Order:       &reused,
			CheckoutURL: result.CheckoutURL,
		}, nil
	}

	order := &model.Order{
		UserID:      userID,
		ReportID:    reportID,
		Type:        "report",
		SKU:         "report_single",
		AmountCents: s.priceCents,
		Currency:    "usd",
		Provider:    s.pay.Name(),
		Status:      model.OrderStatusCreated,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.orderRepo.Create(order); err != nil {
		logger.FromCtx(ctx).Error("order create failed",
			"err", err,
			"user_id", userID,
			"report_id", reportID,
		)
		return nil, err
	}

	logger.FromCtx(ctx).Info("order created",
		"order_id", order.ID,
		"user_id", userID,
		"report_id", reportID,
	)

	checkoutInput := payment.CheckoutInput{
		OrderID:     order.ID,
		AmountCents: int64(order.AmountCents),
		Currency:    order.Currency,
		ProductName: "Destiny Report",
		SuccessURL:  s.successURL,
		CancelURL:   s.cancelURL,
		Metadata: map[string]string{
			"order_id": strconv.FormatUint(order.ID, 10),
		},
	}

	result, err := s.pay.CreateCheckout(ctx, checkoutInput)
	if err != nil {
		logger.FromCtx(ctx).Error("payment checkout failed",
			"err", err,
			"order_id", order.ID,
			"provider", s.pay.Name(),
		)
		return nil, err
	}

	order.ProviderRef = result.SessionID
	if err := s.orderRepo.UpdateProviderRef(order.ID, result.SessionID); err != nil {
		logger.FromCtx(ctx).Error("failed to save provider ref after checkout",
			"err", err,
			"order_id", order.ID,
			"session_id", result.SessionID,
		)
		return nil, err
	}

	logger.FromCtx(ctx).Info("checkout session created",
		"order_id", order.ID,
		"session_id", result.SessionID,
	)

	return &CreateOrderResult{
		Order:       order,
		CheckoutURL: result.CheckoutURL,
	}, nil
}

// GetOrder 获取订单详情（防越权）。
func (s *OrderService) GetOrder(ctx context.Context, userID, orderID uint64) (*model.Order, error) {
	return s.orderRepo.GetByID(orderID, userID)
}

// ListOrders 列出用户订单。
func (s *OrderService) ListOrders(ctx context.Context, userID uint64) ([]model.Order, error) {
	return s.orderRepo.ListByUser(userID)
}
