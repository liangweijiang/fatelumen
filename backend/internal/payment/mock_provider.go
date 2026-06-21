package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
)

// mockProvider 仅用于本地/dev 环境的支付渠道，HMAC-SHA256 自签验签，
// 不依赖任何第三方，便于端到端验通下单与履约链路。生产环境绝不注册。
type mockProvider struct {
	secret  string
	baseURL string
}

// NewMockProvider 创建本地 Mock 支付渠道。secret 用于 HMAC 自签验签，
// baseURL 为本服务对外基址（如 http://localhost:8080），用于拼接本地收银台 URL。
func NewMockProvider(secret, baseURL string) PaymentProvider {
	return &mockProvider{secret: secret, baseURL: baseURL}
}

// Name 返回支付渠道标识。
func (m *mockProvider) Name() string { return "mock" }

// CreateCheckout 不跳第三方，返回本地假收银台 URL，SessionID 用 mock_<order_id> 便于回调反查。
func (m *mockProvider) CreateCheckout(ctx context.Context, in CheckoutInput) (*CheckoutResult, error) {
	sessionID := fmt.Sprintf("mock_%d", in.OrderID)
	checkoutURL := fmt.Sprintf("%s/api/v1/dev/pay/%d", m.baseURL, in.OrderID)
	slog.Info("mock checkout created",
		"session_id", sessionID,
		"order_id", in.OrderID,
	)
	return &CheckoutResult{
		SessionID:   sessionID,
		CheckoutURL: checkoutURL,
	}, nil
}

// Sign 对 payload 生成十六进制 HMAC-SHA256 签名（供 dev 触发接口构造合法回调）。
func (m *mockProvider) Sign(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(m.secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// mockEventPayload mock 回调体结构。
type mockEventPayload struct {
	EventID   string `json:"event_id"`
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	OrderID   uint64 `json:"order_id"`
}

// VerifyAndParse 校验 HMAC 签名（sigHeader 为十六进制签名）并解析为渠道无关事件。
func (m *mockProvider) VerifyAndParse(payload []byte, sigHeader string) (*WebhookEvent, error) {
	expected := m.Sign(payload)
	if !hmac.Equal([]byte(expected), []byte(sigHeader)) {
		slog.Error("mock webhook verify failed",
			"provider", "mock",
		)
		return nil, fmt.Errorf("mock signature mismatch")
	}
	var p mockEventPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		slog.Error("mock webhook unmarshal failed",
			"err", err,
			"provider", "mock",
		)
		return nil, err
	}
	we := &WebhookEvent{
		EventID:   p.EventID,
		Type:      p.Type,
		Provider:  "mock",
		SessionID: p.SessionID,
		OrderID:   p.OrderID,
		Raw:       payload,
	}
	slog.Info("mock webhook parsed",
		"event_type", p.Type,
		"order_id", p.OrderID,
		"session_id", p.SessionID,
	)
	return we, nil
}

// BuildCompletedEvent 构造一条已签名的 checkout.session.completed 回调（供 dev 触发接口使用）。
// 返回 (payload, hexSignature)。
func (m *mockProvider) BuildCompletedEvent(orderID uint64) ([]byte, string, error) {
	p := mockEventPayload{
		EventID:   "mock_evt_" + strconv.FormatUint(orderID, 10),
		Type:      "checkout.session.completed",
		SessionID: fmt.Sprintf("mock_%d", orderID),
		OrderID:   orderID,
	}
	payload, err := json.Marshal(p)
	if err != nil {
		return nil, "", err
	}
	return payload, m.Sign(payload), nil
}

// MockProviderRef 对外暴露 mock provider 的具体类型，供 dev 触发接口构造已签名回调。
type MockProviderRef = mockProvider

// NewMockProviderRef 返回具体类型指针（dev 触发接口用）。
func NewMockProviderRef(secret, baseURL string) *MockProviderRef {
	return &mockProvider{secret: secret, baseURL: baseURL}
}
