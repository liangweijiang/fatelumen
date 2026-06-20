package handler

import (
	"strconv"

	"fatelumen/backend/internal/admin/resource"
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/repository"

	"github.com/gin-gonic/gin"
)

// ResourceHandler 资源驱动后台入口:统一 list/detail/schema。
type ResourceHandler struct {
	registry *resource.Registry
	audit    *repository.AuditRepo
}

func NewResourceHandler(registry *resource.Registry, audit *repository.AuditRepo) *ResourceHandler {
	return &ResourceHandler{registry: registry, audit: audit}
}

func (h *ResourceHandler) adminCtx(c *gin.Context) *resource.AdminContext {
	return &resource.AdminContext{
		AdminID: middleware.GetUserID(c),
		IP:      c.ClientIP(),
	}
}

// Schema GET /admin/resources/:resource/_schema
func (h *ResourceHandler) Schema(c *gin.Context) {
	res, ok := h.registry.Get(c.Param("resource"))
	if !ok {
		response.Fail(c, response.CodeNotFound, "resource not found")
		return
	}
	type actionMeta struct {
		Name  string `json:"name"`
		Label string `json:"label"`
	}
	actions := make([]actionMeta, 0)
	for _, a := range res.Actions() {
		actions = append(actions, actionMeta{Name: a.Name, Label: a.Label})
	}
	response.OK(c, gin.H{"name": res.Name(), "fields": res.Schema(), "actions": actions})
}

// List GET /admin/resources/:resource
func (h *ResourceHandler) List(c *gin.Context) {
	res, ok := h.registry.Get(c.Param("resource"))
	if !ok {
		response.Fail(c, response.CodeNotFound, "resource not found")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	q := resource.ListQuery{
		Page:     page,
		PageSize: pageSize,
		Sort:     c.Query("sort"),
		Search:   c.Query("search"),
		Filters:  map[string]interface{}{},
	}
	for _, f := range res.Schema() {
		if f.Filterable {
			if v := c.Query(f.Key); v != "" {
				q.Filters[f.Key] = v
			}
		}
	}
	result, err := res.List(h.adminCtx(c), q)
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("resource list failed", "err", err, "resource", res.Name())
		response.Error(c, err.Error())
		return
	}
	response.OK(c, result)
}

// Detail GET /admin/resources/:resource/:id
func (h *ResourceHandler) Detail(c *gin.Context) {
	res, ok := h.registry.Get(c.Param("resource"))
	if !ok {
		response.Fail(c, response.CodeNotFound, "resource not found")
		return
	}
	detail, err := res.Detail(h.adminCtx(c), c.Param("id"))
	if err != nil {
		response.Fail(c, response.CodeNotFound, "not found")
		return
	}
	response.OK(c, detail)
}

// Action POST /admin/resources/:resource/:id/actions/:action — 自定义动作,落审计
func (h *ResourceHandler) Action(c *gin.Context) {
	res, ok := h.registry.Get(c.Param("resource"))
	if !ok {
		response.Fail(c, response.CodeNotFound, "resource not found")
		return
	}
	actionName := c.Param("action")
	var target *resource.Action
	for i := range res.Actions() {
		if res.Actions()[i].Name == actionName {
			target = &res.Actions()[i]
			break
		}
	}
	if target == nil {
		response.Fail(c, response.CodeNotFound, "action not found")
		return
	}
	var params map[string]interface{}
	_ = c.ShouldBindJSON(&params)
	ac := h.adminCtx(c)
	out, err := target.Handler(ac, c.Param("id"), params)
	if err != nil {
		logger.FromCtx(c.Request.Context()).Error("resource action failed", "err", err, "resource", res.Name(), "action", actionName)
		response.Error(c, err.Error())
		return
	}
	h.audit.Write(c.Request.Context(), model.AdminAuditLog{
		AdminID:    ac.AdminID,
		AdminName:  ac.AdminName,
		Action:     actionName,
		Resource:   res.Name(),
		ResourceID: c.Param("id"),
		IP:         ac.IP,
	})
	response.OK(c, out)
}
