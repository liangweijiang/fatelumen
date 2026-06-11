package handler

import (
	"fatelumen/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// AdminHandler 后台管理 HTTP 处理器（骨架）。
type AdminHandler struct{}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

// Ping 临时探活接口，验证管理员权限链畅通。
func (h *AdminHandler) Ping(c *gin.Context) {
	response.OK(c, gin.H{"pong": true, "role": "admin"})
}
