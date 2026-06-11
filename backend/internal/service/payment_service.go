package service

import (
	"context"
	"errors"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/payment"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
)

// ---------- 私有接口（单测用 fake 替换）----------

type payOrderStore interface {
	GetBySessionID(sessionID string) (*model.Order, error)
	FulfillPaidOrder(provider, eventID string, orderID uint64) error
}

// PaymentService Webhook 接收与报告解锁编排。
type PaymentService struct {
	pay    payment.PaymentProvider
	orders payOrderStore
}

func NewPaymentService(
	pay payment.PaymentProvider,
	orderRepo *repository.OrderRepo,
) *PaymentService {
	return &PaymentService{
		pay:    pay,
		orders: orderRepo,
	}
}

// HandleWebhook 处理支付回调：验签 → 幂等 → 原子履约。
// 返回 error 仅表示验签失败/恶意请求（调用方应返回 400）。
// 幂等/重复回调/非 completed 类型等均返回 nil（调用方返回 200）。
func (s *PaymentService) HandleWebhook(ctx context.Context, payload []byte, sigHeader string) error {
	evt, err := s.pay.VerifyAndParse(payload, sigHeader)
	if err != nil {
		logger.FromCtx(ctx).Warn("webhook verify failed", "err", err)
		return err
	}

	// 只处理 checkout.session.completed
	if evt.Type != "checkout.session.completed" {
		logger.FromCtx(ctx).Info("webhook event ignored",
			"event_type", evt.Type,
			"event_id", evt.EventID,
		)
		return nil
	}

	// OrderID 缺失
	if evt.OrderID == 0 {
		logger.FromCtx(ctx).Error("webhook missing order_id",
			"event_id", evt.EventID,
			"session_id", evt.SessionID,
		)
		return nil
	}

	// 查订单（事务外，只读）
	order, err := s.orders.GetBySessionID(evt.SessionID)
	if err != nil {
		logger.FromCtx(ctx).Error("webhook order not found",
			"err", err,
			"session_id", evt.SessionID,
			"event_id", evt.EventID,
		)
		return nil
	}

	// 原子履约：去重 + 订单 paid + 报告解锁 在同一事务
	err = s.orders.FulfillPaidOrder("stripe", evt.EventID, order.ID)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEvent) {
			logger.FromCtx(ctx).Info("webhook duplicate event ignored",
				"event_id", evt.EventID,
			)
			return nil
		}
		logger.FromCtx(ctx).Error("webhook fulfill failed",
			"err", err,
			"order_id", order.ID,
			"event_id", evt.EventID,
		)
		return nil
	}

	logger.FromCtx(ctx).Info("webhook fulfilled",
		"order_id", order.ID,
		"event_id", evt.EventID,
	)
	return nil
}
