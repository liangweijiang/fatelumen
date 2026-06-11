package handler

import (
	"context"

	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type adminStatsService interface {
	GetStats(ctx context.Context) (*service.Stats, error)
}

// AdminHandler 后台管理 HTTP 处理器。
type AdminHandler struct {
	statsSvc adminStatsService
}

func NewAdminHandler(statsSvc *service.StatsService) *AdminHandler {
	return &AdminHandler{statsSvc: statsSvc}
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
