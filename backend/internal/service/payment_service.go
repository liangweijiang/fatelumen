package service

import (
	"context"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/payment"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
)

// ---------- 私有接口（单测用 fake 替换）----------

type payOrderStore interface {
	GetBySessionID(sessionID string) (*model.Order, error)
	MarkPaid(orderID uint64, providerTxnID string, meta []byte) error
}

type payReportStore interface {
	MarkPaid(reportID, orderID uint64) error
}

type webhookEventStore interface {
	MarkProcessed(provider, eventID, eventType string) (duplicate bool, err error)
}

// PaymentService Webhook 接收与报告解锁编排。
type PaymentService struct {
	pay     payment.PaymentProvider
	orders  payOrderStore
	reports payReportStore
	events  webhookEventStore
}

func NewPaymentService(
	pay payment.PaymentProvider,
	orderRepo *repository.OrderRepo,
	reportRepo *repository.ReportRepo,
	eventRepo *repository.WebhookEventRepo,
) *PaymentService {
	return &PaymentService{
		pay:     pay,
		orders:  orderRepo,
		reports: reportRepo,
		events:  eventRepo,
	}
}

// HandleWebhook 处理支付回调：验签 → 幂等 → 解锁报告。
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

	// 幂等去重
	dup, err := s.events.MarkProcessed("stripe", evt.EventID, evt.Type)
	if err != nil {
		logger.FromCtx(ctx).Error("webhook mark processed failed",
			"err", err,
			"event_id", evt.EventID,
		)
		return nil
	}
	if dup {
		logger.FromCtx(ctx).Info("duplicate webhook ignored",
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

	// 查订单
	order, err := s.orders.GetBySessionID(evt.SessionID)
	if err != nil {
		logger.FromCtx(ctx).Error("webhook order not found",
			"err", err,
			"session_id", evt.SessionID,
			"event_id", evt.EventID,
		)
		return nil
	}

	// 订单已支付（双保险）
	if order.Status == model.OrderStatusPaid {
		logger.FromCtx(ctx).Info("order already paid",
			"order_id", order.ID,
			"event_id", evt.EventID,
		)
		return nil
	}

	// 标记订单已支付（事务内 Transit）
	if err := s.orders.MarkPaid(order.ID, evt.PaymentIntentID, evt.Raw); err != nil {
		logger.FromCtx(ctx).Error("webhook mark order paid failed",
			"err", err,
			"order_id", order.ID,
			"event_id", evt.EventID,
		)
		return nil
	}

	// 解锁报告
	if err := s.reports.MarkPaid(order.ReportID, order.ID); err != nil {
		logger.FromCtx(ctx).Error("webhook mark report paid failed",
			"err", err,
			"report_id", order.ReportID,
			"order_id", order.ID,
		)
		return nil
	}

	logger.FromCtx(ctx).Info("report unlocked",
		"order_id", order.ID,
		"report_id", order.ReportID,
		"event_id", evt.EventID,
	)
	return nil
}
