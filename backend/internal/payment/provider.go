package payment

import (
	"context"
	"strconv"
)

// CheckoutInput 下单参数。
type CheckoutInput struct {
	OrderID     uint64
	AmountCents int64
	Currency    string
	ProductName string
	SuccessURL  string
	CancelURL   string
	Metadata    map[string]string
}

// CheckoutResult 下单结果（渠道无关）。
type CheckoutResult struct {
	SessionID   string
	CheckoutURL string
}

// WebhookEvent 已验签的回调事件（渠道无关）。
type WebhookEvent struct {
	EventID         string
	Type            string
	Provider        string // provider 名（如 "stripe"），由 provider 实现层在 VerifyAndParse 时填入
	SessionID       string
	PaymentIntentID string
	OrderID         uint64
	Raw             []byte
}

// PaymentProvider 每个支付渠道实现此接口。
type PaymentProvider interface {
	// Name 返回支付渠道名（如 "stripe"），由 provider 实现层定义。
	Name() string
	CreateCheckout(ctx context.Context, in CheckoutInput) (*CheckoutResult, error)
	VerifyAndParse(payload []byte, sigHeader string) (*WebhookEvent, error)
}

// ParseOrderID 从 metadata 中解析 order_id，缺失或非数字返回 0。
func ParseOrderID(metadata map[string]string) uint64 {
	v, ok := metadata["order_id"]
	if !ok || v == "" {
		return 0
	}
	id, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0
	}
	return id
}
