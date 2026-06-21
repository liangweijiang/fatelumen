package payment

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"

	"github.com/smartwalle/alipay/v3"
)

// alipayProvider 实现 PaymentProvider，封装支付宝（国内开放平台）渠道。
type alipayProvider struct {
	client    *alipay.Client
	notifyURL string
	returnURL string
}

// NewAlipayProvider 创建支付宝支付渠道实现。production=false 即沙箱环境。
// 加载支付宝公钥失败时返回 error。
func NewAlipayProvider(appID, privateKey, alipayPublicKey, notifyURL, returnURL string, production bool) (PaymentProvider, error) {
	client, err := alipay.New(appID, privateKey, production)
	if err != nil {
		slog.Error("alipay client init failed",
			"err", err,
			"provider", "alipay",
		)
		return nil, err
	}
	if err := client.LoadAliPayPublicKey(alipayPublicKey); err != nil {
		slog.Error("alipay load public key failed",
			"err", err,
			"provider", "alipay",
		)
		return nil, err
	}
	return &alipayProvider{
		client:    client,
		notifyURL: notifyURL,
		returnURL: returnURL,
	}, nil
}

// Name 返回支付渠道标识。
func (p *alipayProvider) Name() string { return "alipay" }

// CreateCheckout 创建支付宝 PC 网页支付，返回收银台跳转 URL。
func (p *alipayProvider) CreateCheckout(ctx context.Context, in CheckoutInput) (*CheckoutResult, error) {
	outTradeNo := strconv.FormatUint(in.OrderID, 10)

	var pay = alipay.TradePagePay{}
	pay.NotifyURL = p.notifyURL
	pay.ReturnURL = p.returnURL
	pay.Subject = in.ProductName
	pay.OutTradeNo = outTradeNo
	pay.TotalAmount = fmt.Sprintf("%.2f", float64(in.AmountCents)/100)
	pay.ProductCode = "FAST_INSTANT_TRADE_PAY"

	payURL, err := p.client.TradePagePay(pay)
	if err != nil {
		slog.Error("alipay create checkout failed",
			"err", err,
			"provider", "alipay",
			"order_id", in.OrderID,
		)
		return nil, err
	}

	slog.Info("alipay checkout created",
		"session_id", outTradeNo,
		"order_id", in.OrderID,
	)
	// 支付宝无独立 session 概念，SessionID 复用商户订单号。
	return &CheckoutResult{
		SessionID:   outTradeNo,
		CheckoutURL: payURL.String(),
	}, nil
}

// VerifyAndParse 验签支付宝异步通知并解析为渠道无关事件。
// 支付宝是表单 POST 通知（非 JSON），payload 为原始 application/x-www-form-urlencoded 字节，
// sigHeader 在本渠道未使用（验签所需字段已包含在表单内）。
func (p *alipayProvider) VerifyAndParse(payload []byte, sigHeader string) (*WebhookEvent, error) {
	values, err := url.ParseQuery(string(payload))
	if err != nil {
		slog.Error("alipay parse notification payload failed",
			"err", err,
			"provider", "alipay",
		)
		return nil, err
	}

	// DecodeNotification 内部已验签，验签失败返回 error。
	notification, err := p.client.DecodeNotification(context.Background(), values)
	if err != nil {
		slog.Error("alipay webhook verify failed",
			"err", err,
			"provider", "alipay",
		)
		return nil, err
	}

	orderID, err := strconv.ParseUint(notification.OutTradeNo, 10, 64)
	if err != nil {
		slog.Error("alipay parse out_trade_no failed",
			"err", err,
			"provider", "alipay",
			"out_trade_no", notification.OutTradeNo,
		)
		return nil, err
	}

	we := &WebhookEvent{
		EventID:  notification.TradeNo,
		Provider: "alipay",
		OrderID:  orderID,
		Raw:      payload,
	}

	switch notification.TradeStatus {
	case alipay.TradeStatusSuccess, alipay.TradeStatusFinished:
		we.Type = "checkout.session.completed"
	default:
		// 其余状态保留原始 trade_status，让上层 HandleWebhook 忽略。
		we.Type = string(notification.TradeStatus)
	}

	slog.Info("alipay webhook parsed",
		"trade_status", string(notification.TradeStatus),
		"order_id", we.OrderID,
		"session_id", notification.OutTradeNo,
	)
	return we, nil
}
