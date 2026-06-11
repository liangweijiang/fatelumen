package payment

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
)

// stripeProvider 实现 PaymentProvider，封装 Stripe 渠道。
type stripeProvider struct {
	secretKey     string
	webhookSecret string
}

// NewStripeProvider 创建 Stripe 支付渠道实现。
func NewStripeProvider(secretKey, webhookSecret string) PaymentProvider {
	return &stripeProvider{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
	}
}

// CreateCheckout 创建 Stripe Checkout Session，返回会话 ID 与前端跳转 URL。
func (s *stripeProvider) CreateCheckout(ctx context.Context, in CheckoutInput) (*CheckoutResult, error) {
	stripe.Key = s.secretKey

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(in.Currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(in.ProductName),
					},
					UnitAmount: stripe.Int64(in.AmountCents),
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(in.SuccessURL),
		CancelURL:  stripe.String(in.CancelURL),
		Metadata:   in.Metadata,
	}

	sess, err := session.New(params)
	if err != nil {
		slog.Error("stripe create checkout failed",
			"err", err,
			"provider", "stripe",
			"order_id", in.OrderID,
		)
		return nil, err
	}

	slog.Info("stripe checkout session created",
		"session_id", sess.ID,
		"order_id", in.OrderID,
	)
	return &CheckoutResult{
		SessionID:   sess.ID,
		CheckoutURL: sess.URL,
	}, nil
}

// VerifyAndParse 验签 Stripe Webhook 并解析为渠道无关事件。
func (s *stripeProvider) VerifyAndParse(payload []byte, sigHeader string) (*WebhookEvent, error) {
	event, err := webhook.ConstructEvent(payload, sigHeader, s.webhookSecret)
	if err != nil {
		slog.Error("stripe webhook verify failed",
			"err", err,
			"provider", "stripe",
		)
		return nil, err
	}

	we := &WebhookEvent{
		Type: string(event.Type),
		Raw:  payload,
	}

	if event.Type == "checkout.session.completed" {
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			slog.Error("stripe webhook unmarshal checkout session failed",
				"err", err,
				"event_type", string(event.Type),
			)
			return nil, err
		}
		we.SessionID = sess.ID
		if sess.PaymentIntent != nil {
			we.PaymentIntentID = sess.PaymentIntent.ID
		}
		we.OrderID = ParseOrderID(sess.Metadata)
	}

	slog.Info("stripe webhook parsed",
		"event_type", string(event.Type),
		"order_id", we.OrderID,
		"session_id", we.SessionID,
	)
	return we, nil
}
