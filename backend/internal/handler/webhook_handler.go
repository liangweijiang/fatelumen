package handler

import (
	"context"

	"fatelumen/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type paymentSvc interface {
	HandleWebhook(ctx context.Context, payload []byte, sigHeader string) error
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

	if err := h.svc.HandleWebhook(c.Request.Context(), payload, sig); err != nil {
		c.JSON(400, gin.H{"error": "webhook verification failed"})
		return
	}

	c.JSON(200, gin.H{"received": true})
}
