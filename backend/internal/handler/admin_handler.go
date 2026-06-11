package handler

import (
	"context"
	"strconv"
	"strings"

	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type adminStatsService interface {
	GetStats(ctx context.Context) (*service.Stats, error)
}

type adminUserService interface {
	ListUsers(ctx context.Context, keyword string, page, pageSize int) (*service.AdminUsersPage, error)
	GetUserDetail(ctx context.Context, userID uint64) (*service.AdminUserDetail, error)
	SetUserActive(ctx context.Context, operatorID, targetUserID uint64, active bool) error
}

type adminOrderService interface {
	ListOrders(ctx context.Context, status string, userID uint64, page, pageSize int) (*service.AdminOrdersPage, error)
	GetOrderDetail(ctx context.Context, orderID uint64) (*service.AdminOrderDetail, error)
}

type adminReportService interface {
	ListReports(ctx context.Context, status string, paid *bool, userID uint64, page, pageSize int) (*service.AdminReportsPage, error)
	GetReportDetail(ctx context.Context, reportID uint64) (*service.AdminReportDetail, error)
	UnlockReport(ctx context.Context, operatorID, reportID uint64, reason string) error
}

// AdminHandler 后台管理 HTTP 处理器。
type AdminHandler struct {
	statsSvc  adminStatsService
	userSvc   adminUserService
	orderSvc  adminOrderService
	reportSvc adminReportService
}

func NewAdminHandler(statsSvc *service.StatsService, userSvc *service.AdminUserService, orderSvc *service.AdminOrderService, reportSvc *service.AdminReportService) *AdminHandler {
	return &AdminHandler{statsSvc: statsSvc, userSvc: userSvc, orderSvc: orderSvc, reportSvc: reportSvc}
}

// Ping 临时探活接口，验证管理员权限链畅通。
func (h *AdminHandler) Ping(c *gin.Context) {
	response.OK(c, gin.H{"pong": true, "role": "admin"})
}

// Stats GET /api/v1/admin/stats — 数据看板统计。
func (h *AdminHandler) Stats(c *gin.Context) {
	stats, err := h.statsSvc.GetStats(c.Request.Context())
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.OK(c, stats)
}

// ListUsers GET /api/v1/admin/users — 用户列表。
func (h *AdminHandler) ListUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.userSvc.ListUsers(c.Request.Context(), keyword, page, pageSize)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.OK(c, result)
}

// GetUser GET /api/v1/admin/users/:id — 用户详情。
func (h *AdminHandler) GetUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid user id")
		return
	}

	detail, err := h.userSvc.GetUserDetail(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "user not found")
		return
	}
	response.OK(c, detail)
}

// SetActive PATCH /api/v1/admin/users/:id/active — 启用/禁用用户。
func (h *AdminHandler) SetActive(c *gin.Context) {
	operatorID := middleware.GetUserID(c)
	if operatorID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid user id")
		return
	}

	var body struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid request body")
		return
	}

	if err := h.userSvc.SetUserActive(c.Request.Context(), operatorID, id, body.Active); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.OK(c, gin.H{"id": id, "active": body.Active})
}

// ListOrders GET /api/v1/admin/orders — 订单列表（支持筛选）。
func (h *AdminHandler) ListOrders(c *gin.Context) {
	status := c.Query("status")
	userID, _ := strconv.ParseUint(c.DefaultQuery("user_id", "0"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.orderSvc.ListOrders(c.Request.Context(), status, userID, page, pageSize)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.OK(c, result)
}

// GetOrder GET /api/v1/admin/orders/:id — 订单详情（含原始回调 JSON）。
func (h *AdminHandler) GetOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid order id")
		return
	}

	detail, err := h.orderSvc.GetOrderDetail(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "order not found")
		return
	}
	response.OK(c, detail)
}

// ListReports GET /api/v1/admin/reports — 报告列表（支持筛选）。
func (h *AdminHandler) ListReports(c *gin.Context) {
	status := c.Query("status")
	userID, _ := strconv.ParseUint(c.DefaultQuery("user_id", "0"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var paid *bool
	if ps := c.Query("paid"); ps != "" {
		if strings.EqualFold(ps, "true") {
			t := true
			paid = &t
		} else if strings.EqualFold(ps, "false") {
			f := false
			paid = &f
		}
	}

	result, err := h.reportSvc.ListReports(c.Request.Context(), status, paid, userID, page, pageSize)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.OK(c, result)
}

// GetReport GET /api/v1/admin/reports/:id — 报告详情。
func (h *AdminHandler) GetReport(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid report id")
		return
	}

	detail, err := h.reportSvc.GetReportDetail(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "report not found")
		return
	}
	response.OK(c, detail)
}

// UnlockReport POST /api/v1/admin/reports/:id/unlock — 人工解锁报告。
func (h *AdminHandler) UnlockReport(c *gin.Context) {
	operatorID := middleware.GetUserID(c)
	if operatorID == 0 {
		response.Fail(c, response.CodeUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid report id")
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, response.CodeBadRequest, "invalid request body")
		return
	}

	if err := h.reportSvc.UnlockReport(c.Request.Context(), operatorID, id, body.Reason); err != nil {
		if err.Error() == "reason required" {
			response.Fail(c, response.CodeBadRequest, "reason required")
			return
		}
		response.Error(c, err.Error())
		return
	}
	response.OK(c, gin.H{"unlocked": true})
}
