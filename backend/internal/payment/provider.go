package payment

import (
	"context"
	"net/http"
)

// CheckoutResult 统一的下单结果。前端按 Action 分支，不感知具体渠道。
type CheckoutResult struct {
	Action      string // "redirect" | "client_confirm"
	CheckoutURL string // redirect 模式：托管支付页
	ClientToken string // client_confirm 模式：前端 SDK 用
	ProviderRef string // 渠道侧主标识，写入 orders.provider_ref
}

// CheckoutInput 下单参数。
type CheckoutInput struct {
	Order      *Order // 已落库的订单（含 sku/amount/currency/type）
	UserID     uint64
	Locale     string
	SuccessURL string
	CancelURL  string
}

// EventType 已归一化的事件类型。
type EventType string

const (
	EventPaymentSucceeded EventType = "payment_succeeded"
	EventPaymentFailed    EventType = "payment_failed"
	EventRefunded         EventType = "refunded"
	EventIgnored          EventType = "ignored"
)

// PaymentEvent 已归一化的回调事件（各渠道验签+解析后产出，业务层只认这个）。
type PaymentEvent struct {
	Provider      string
	EventID       string // 渠道事件唯一 ID，用于幂等
	Type          EventType
	ProviderRef   string // 用于回查订单
	ProviderTxnID string
	AmountCents   int
	Currency      string
	Raw           []byte // 原始 payload，存 orders.provider_meta 供对账
}

// PaymentProvider 每个渠道实现此接口。
type PaymentProvider interface {
	// ID 渠道标识，如 "stripe"
	ID() string
	// Checkout 起支付：创建渠道侧会话/订单
	Checkout(ctx context.Context, in CheckoutInput) (*CheckoutResult, error)
	// ParseWebhook 验签 + 解析回调，归一化为 PaymentEvent；验签失败返回 error
	ParseWebhook(ctx context.Context, headers http.Header, body []byte) (*PaymentEvent, error)
	// Refund 退款（v2 再实现，MVP 可返回 ErrNotSupported）
	Refund(ctx context.Context, order *Order, reason string) error
}

// Order 是 payment 包内对订单模型的抽象（避免循环依赖），实际存 model.Order。
type Order struct {
	ID          uint64
	UserID      uint64
	Type        string
	SKU         string
	AmountCents int
	Currency    string
	Status      string
}
