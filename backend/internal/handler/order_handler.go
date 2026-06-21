package handler

import (
	"context"
	"errors"
	"strconv"

	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type orderSvc interface {
	CreateOrder(ctx context.Context, userID uint64, in service.CreateOrderInput) (*service.CreateOrderResult, error)
	GetOrder(ctx context.Context, userID, orderID uint64) (*model.Order, error)
	ListOrders(ctx context.Context, userID uint64) ([]model.Order, error)
}

// OrderHandler 订单 HTTP 处理器。
type OrderHandler struct {
	svc orderSvc
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

type createOrderIn struct {
	ReportID uint64 `json:"report_id"`
	SKU      string `json:"sku"`
	Provider string `json:"provider"`
}

// Create POST /api/v1/orders
func (h *OrderHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}

	var in createOrderIn
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid request body")
		return
	}
	if in.Provider == "" {
		response.Fail(c, response.CodeBadRequest, "provider is required")
		return
	}
	// 买报告必须带 report_id；买积分套餐（sku=pack_*）不需要 report_id
	if in.SKU == "" && in.ReportID == 0 {
		response.Fail(c, response.CodeBadRequest, "report_id is required")
		return
	}

	result, err := h.svc.CreateOrder(c.Request.Context(), userID, service.CreateOrderInput{
		ReportID: in.ReportID,
		SKU:      in.SKU,
		Provider: in.Provider,
	})
	if err != nil {
		if errors.Is(err, service.ErrReportAlreadyPurchased) {
			response.Fail(c, response.CodeOrderUnpaid, "report already purchased")
			return
		}
		response.Error(c, err.Error())
		return
	}
	response.OK(c, gin.H{
		"order_id":     result.Order.ID,
		"status":       result.Order.Status,
		"checkout_url": result.CheckoutURL,
	})
}

// Get GET /api/v1/orders/:id
func (h *OrderHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid order id")
		return
	}

	order, err := h.svc.GetOrder(c.Request.Context(), userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, response.CodeNotFound, "order not found")
			return
		}
		response.Error(c, err.Error())
		return
	}
	response.OK(c, order)
}

// List GET /api/v1/orders
func (h *OrderHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}

	orders, err := h.svc.ListOrders(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.OK(c, orders)
}
