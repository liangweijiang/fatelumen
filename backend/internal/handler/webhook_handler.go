package handler

import (
	"context"
	"encoding/json"

	"fatelumen/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type paymentSvc interface {
	HandleWebhook(ctx context.Context, provider string, payload []byte, sigHeader string) error
}

// WebhookHandler 支付回调 HTTP 处理器（无鉴权）。
type WebhookHandler struct {
	svc paymentSvc
}

func NewWebhookHandler(svc *service.PaymentService) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

// Stripe POST /api/v1/webhooks/stripe
func (h *WebhookHandler) Stripe(c *gin.Context) {
	payload, err := c.GetRawData()
	if err != nil {
		c.JSON(400, gin.H{"error": "failed to read request body"})
		return
	}

	sig := c.GetHeader("Stripe-Signature")

	if err := h.svc.HandleWebhook(c.Request.Context(), "stripe", payload, sig); err != nil {
		c.JSON(400, gin.H{"error": "webhook verification failed"})
		return
	}

	c.JSON(200, gin.H{"received": true})
}

// Alipay POST /api/v1/webhooks/alipay
// 支付宝异步通知为表单 POST，验签字段在表单体内，sigHeader 传空字符串。
// 支付宝靠响应体判断是否需要重发：成功须返回纯文本 "success"。
func (h *WebhookHandler) Alipay(c *gin.Context) {
	payload, err := c.GetRawData()
	if err != nil {
		c.String(400, "fail")
		return
	}

	if err := h.svc.HandleWebhook(c.Request.Context(), "alipay", payload, ""); err != nil {
		c.String(400, "fail")
		return
	}

	c.String(200, "success")
}

// Paypal POST /api/v1/webhooks/paypal
// PayPal 验签需要多个 header，按约定 JSON 编码成一个字符串传入 sigHeader。
func (h *WebhookHandler) Paypal(c *gin.Context) {
	payload, err := c.GetRawData()
	if err != nil {
		c.JSON(400, gin.H{"error": "failed to read request body"})
		return
	}

	headers := map[string]string{
		"Paypal-Transmission-Id":   c.GetHeader("Paypal-Transmission-Id"),
		"Paypal-Transmission-Time": c.GetHeader("Paypal-Transmission-Time"),
		"Paypal-Cert-Url":          c.GetHeader("Paypal-Cert-Url"),
		"Paypal-Auth-Algo":         c.GetHeader("Paypal-Auth-Algo"),
		"Paypal-Transmission-Sig":  c.GetHeader("Paypal-Transmission-Sig"),
	}
	sigJSON, err := json.Marshal(headers)
	if err != nil {
		c.JSON(400, gin.H{"error": "failed to encode signature headers"})
		return
	}

	if err := h.svc.HandleWebhook(c.Request.Context(), "paypal", payload, string(sigJSON)); err != nil {
		c.JSON(400, gin.H{"error": "webhook verification failed"})
		return
	}

	c.JSON(200, gin.H{"received": true})
}
