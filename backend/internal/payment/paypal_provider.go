package payment

// PayPal 渠道实现。
//
// 验签约定：plutov/paypal/v4 的 VerifyWebhookSignature 需要完整的 *http.Request
// （它从请求头读取 PAYPAL-AUTH-ALGO / PAYPAL-CERT-URL / PAYPAL-TRANSMISSION-ID /
// PAYPAL-TRANSMISSION-SIG / PAYPAL-TRANSMISSION-TIME，并读取 body 调用 PayPal 校验接口）。
// 但 PaymentProvider.VerifyAndParse 的签名只有 (payload []byte, sigHeader string)，
// 为不破坏统一接口，约定：上层把 PayPal 的多个验签 header 用 JSON 编码成一个字符串
// 传入 sigHeader（形如 {"PAYPAL-TRANSMISSION-ID":"...","PAYPAL-TRANSMISSION-SIG":"..."}）。
// 本 provider 内 json.Unmarshal 还原成 map，重建 *http.Request 后调用
// VerifyWebhookSignature 完成验签。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/plutov/paypal/v4"
)

// paypalProvider 实现 PaymentProvider，封装 PayPal 渠道。
type paypalProvider struct {
	client    *paypal.Client
	webhookID string
}

// NewPaypalProvider 创建 PayPal 支付渠道实现。production=false 即沙箱环境。
func NewPaypalProvider(clientID, secret, webhookID string, production bool) PaymentProvider {
	apiBase := paypal.APIBaseSandBox
	if production {
		apiBase = paypal.APIBaseLive
	}
	client, err := paypal.NewClient(clientID, secret, apiBase)
	if err != nil {
		slog.Error("paypal client init failed",
			"err", err,
			"provider", "paypal",
		)
	}
	return &paypalProvider{
		client:    client,
		webhookID: webhookID,
	}
}

// Name 返回支付渠道标识。
func (p *paypalProvider) Name() string { return "paypal" }

// CreateCheckout 创建 PayPal 订单（CAPTURE 意图），返回审批跳转 URL。
func (p *paypalProvider) CreateCheckout(ctx context.Context, in CheckoutInput) (*CheckoutResult, error) {
	units := []paypal.PurchaseUnitRequest{
		{
			CustomID: strconv.FormatUint(in.OrderID, 10),
			Amount: &paypal.PurchaseUnitAmount{
				Currency: strings.ToUpper(in.Currency),
				Value:    fmt.Sprintf("%.2f", float64(in.AmountCents)/100),
			},
		},
	}
	appCtx := &paypal.ApplicationContext{
		ReturnURL: in.SuccessURL,
		CancelURL: in.CancelURL,
	}

	order, err := p.client.CreateOrder(ctx, paypal.OrderIntentCapture, units, nil, appCtx)
	if err != nil {
		slog.Error("paypal create order failed",
			"err", err,
			"provider", "paypal",
			"order_id", in.OrderID,
		)
		return nil, err
	}

	var checkoutURL string
	for _, link := range order.Links {
		if link.Rel == "approve" {
			checkoutURL = link.Href
			break
		}
	}

	slog.Info("paypal order created",
		"session_id", order.ID,
		"order_id", in.OrderID,
	)
	return &CheckoutResult{
		SessionID:   order.ID,
		CheckoutURL: checkoutURL,
	}, nil
}

// VerifyAndParse 验签 PayPal Webhook 并解析为渠道无关事件。
// sigHeader 为 PayPal 验签 header 的 JSON 编码（约定见文件头注释）。
func (p *paypalProvider) VerifyAndParse(payload []byte, sigHeader string) (*WebhookEvent, error) {
	var headers map[string]string
	if err := json.Unmarshal([]byte(sigHeader), &headers); err != nil {
		slog.Error("paypal decode sig header failed",
			"err", err,
			"provider", "paypal",
		)
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, "/", bytes.NewReader(payload))
	if err != nil {
		slog.Error("paypal build verify request failed",
			"err", err,
			"provider", "paypal",
		)
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := p.client.VerifyWebhookSignature(context.Background(), req, p.webhookID)
	if err != nil {
		slog.Error("paypal webhook verify failed",
			"err", err,
			"provider", "paypal",
		)
		return nil, err
	}
	if resp.VerificationStatus != "SUCCESS" {
		err := fmt.Errorf("paypal webhook verification failed: %s", resp.VerificationStatus)
		slog.Error("paypal webhook verification not success",
			"err", err,
			"provider", "paypal",
			"status", resp.VerificationStatus,
		)
		return nil, err
	}

	var evt struct {
		ID        string          `json:"id"`
		EventType string          `json:"event_type"`
		Resource  json.RawMessage `json:"resource"`
	}
	if err := json.Unmarshal(payload, &evt); err != nil {
		slog.Error("paypal unmarshal webhook event failed",
			"err", err,
			"provider", "paypal",
		)
		return nil, err
	}

	we := &WebhookEvent{
		EventID:  evt.ID,
		Provider: "paypal",
		OrderID:  parsePaypalOrderID(evt.Resource),
		Raw:      payload,
	}

	switch evt.EventType {
	case "CHECKOUT.ORDER.APPROVED", "PAYMENT.CAPTURE.COMPLETED":
		we.Type = "checkout.session.completed"
	default:
		we.Type = evt.EventType
	}

	slog.Info("paypal webhook parsed",
		"event_type", evt.EventType,
		"order_id", we.OrderID,
	)
	return we, nil
}

// parsePaypalOrderID 从 webhook resource 中取回下单时塞入的 custom_id 并解析为订单号。
// 优先取 resource.custom_id，缺失则回退到 resource.purchase_units[0].custom_id。
func parsePaypalOrderID(resource json.RawMessage) uint64 {
	if len(resource) == 0 {
		return 0
	}
	var r struct {
		CustomID      string `json:"custom_id"`
		PurchaseUnits []struct {
			CustomID string `json:"custom_id"`
		} `json:"purchase_units"`
	}
	if err := json.Unmarshal(resource, &r); err != nil {
		return 0
	}
	customID := r.CustomID
	if customID == "" && len(r.PurchaseUnits) > 0 {
		customID = r.PurchaseUnits[0].CustomID
	}
	if customID == "" {
		return 0
	}
	id, err := strconv.ParseUint(customID, 10, 64)
	if err != nil {
		return 0
	}
	return id
}
