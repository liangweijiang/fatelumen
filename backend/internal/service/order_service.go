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

// CreateOrderInput 下单入参。SKU 为空时默认 report_single（买报告）。
// 买报告时 ReportID 必填；买积分套餐（pack_*）时 ReportID 忽略。
type CreateOrderInput struct {
	ReportID uint64
	SKU      string
	Provider string
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
	reg        *payment.Registry
	successURL string
	cancelURL  string
}

func NewOrderService(
	orderRepo *repository.OrderRepo,
	reportRepo *repository.ReportRepo,
	reg *payment.Registry,
	successURL string,
	cancelURL string,
) *OrderService {
	return &OrderService{
		orderRepo:  orderRepo,
		reportRepo: reportRepo,
		reg:        reg,
		successURL: successURL,
		cancelURL:  cancelURL,
	}
}

// CreateOrder 创建订单并发起支付，按 SKU 类型分流（report / credits）。
func (s *OrderService) CreateOrder(ctx context.Context, userID uint64, in CreateOrderInput) (*CreateOrderResult, error) {
	provider := in.Provider
	sku := in.SKU
	if sku == "" {
		sku = "report_single"
	}
	reportID := in.ReportID

	// 校验支付渠道
	prov, ok := s.reg.Get(provider)
	if !ok {
		logger.FromCtx(ctx).Warn("order create failed: unknown provider",
			"provider", provider,
			"user_id", userID,
		)
		return nil, fmt.Errorf("unknown payment provider: %s", provider)
	}

	// 校验 SKU 类型
	skuType := payment.SKUType(sku)
	if skuType == "" {
		logger.FromCtx(ctx).Warn("order create failed: unknown sku",
			"sku", sku,
			"user_id", userID,
		)
		return nil, fmt.Errorf("unknown sku: %s", sku)
	}

	// 按渠道结算币种取价
	currency := payment.CurrencyForProvider(provider)
	amount, ok := payment.PriceFor(sku, currency)
	if !ok {
		logger.FromCtx(ctx).Error("order create failed: price not found",
			"provider", provider,
			"currency", currency,
			"sku", sku,
		)
		return nil, fmt.Errorf("price not found for sku %s currency %s", sku, currency)
	}

	// 买报告才校验报告归属 + 报告级幂等；买积分套餐跳过报告相关逻辑
	if skuType == "report" {
		if reportID == 0 {
			return nil, fmt.Errorf("report_id is required for report sku")
		}
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
		return s.createReportOrder(ctx, userID, reportID, sku, provider, prov, amount, currency)
	}

	// 积分套餐订单
	return s.createCreditsOrder(ctx, userID, sku, provider, prov, amount, currency)
}

// createReportOrder 创建报告订单并发起支付（含同报告幂等与复用 pending 单）。
func (s *OrderService) createReportOrder(ctx context.Context, userID, reportID uint64, sku, provider string, prov payment.PaymentProvider, amount int, currency string) (*CreateOrderResult, error) {

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
		result, err := prov.CreateCheckout(ctx, checkoutInput)
		if err != nil {
			logger.FromCtx(ctx).Error("payment checkout failed for reused order",
				"err", err,
				"order_id", reused.ID,
				"provider", prov.Name(),
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
		SKU:         sku,
		AmountCents: amount,
		Currency:    currency,
		Provider:    provider,
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

	result, err := prov.CreateCheckout(ctx, checkoutInput)
	if err != nil {
		logger.FromCtx(ctx).Error("payment checkout failed",
			"err", err,
			"order_id", order.ID,
			"provider", prov.Name(),
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

// createCreditsOrder 创建积分套餐订单并发起支付。
func (s *OrderService) createCreditsOrder(ctx context.Context, userID uint64, sku, provider string, prov payment.PaymentProvider, amount int, currency string) (*CreateOrderResult, error) {
	order := &model.Order{
		UserID:         userID,
		ReportID:       0,
		Type:           "credits",
		SKU:            sku,
		AmountCents:    amount,
		Currency:       currency,
		CreditsGranted: payment.CreditsFor(sku),
		Provider:       provider,
		Status:         model.OrderStatusCreated,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := s.orderRepo.Create(order); err != nil {
		logger.FromCtx(ctx).Error("credits order create failed",
			"err", err,
			"user_id", userID,
			"sku", sku,
		)
		return nil, err
	}
	logger.FromCtx(ctx).Info("credits order created",
		"order_id", order.ID,
		"user_id", userID,
		"sku", sku,
		"credits", order.CreditsGranted,
	)
	checkoutInput := payment.CheckoutInput{
		OrderID:     order.ID,
		AmountCents: int64(order.AmountCents),
		Currency:    order.Currency,
		ProductName: "Credits " + sku,
		SuccessURL:  s.successURL,
		CancelURL:   s.cancelURL,
		Metadata: map[string]string{
			"order_id": strconv.FormatUint(order.ID, 10),
		},
	}
	result, err := prov.CreateCheckout(ctx, checkoutInput)
	if err != nil {
		logger.FromCtx(ctx).Error("credits checkout failed",
			"err", err,
			"order_id", order.ID,
			"provider", prov.Name(),
		)
		return nil, err
	}
	order.ProviderRef = result.SessionID
	if err := s.orderRepo.UpdateProviderRef(order.ID, result.SessionID); err != nil {
		logger.FromCtx(ctx).Error("failed to save provider ref for credits order",
			"err", err,
			"order_id", order.ID,
			"session_id", result.SessionID,
		)
		return nil, err
	}
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
