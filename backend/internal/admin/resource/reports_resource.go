package resource

import (
	"context"
	"strconv"
	"strings"

	"fatelumen/backend/internal/service"
)

type reportListSvc interface {
	ListReports(ctx context.Context, status string, paid *bool, userID uint64, page, pageSize int) (*service.AdminReportsPage, error)
	GetReportDetail(ctx context.Context, reportID uint64) (*service.AdminReportDetail, error)
}

type ReportsResource struct {
	svc reportListSvc
}

func NewReportsResource(svc reportListSvc) *ReportsResource {
	return &ReportsResource{svc: svc}
}

func (r *ReportsResource) Name() string { return "reports" }

func (r *ReportsResource) Schema() []Field {
	return []Field{
		{Key: "id", Label: "ID", Type: "int", Sortable: true},
		{Key: "user_id", Label: "用户ID", Type: "int", Filterable: true},
		{Key: "status", Label: "状态", Type: "enum", Enum: []EnumOption{{Value: "pending", Label: "排队中"}, {Value: "processing", Label: "推演中"}, {Value: "completed", Label: "已完成"}, {Value: "failed", Label: "失败"}}, Filterable: true},
		{Key: "paid", Label: "已解锁", Type: "bool", Filterable: true},
		{Key: "created_at", Label: "创建时间", Type: "datetime", Sortable: true},
	}
}

func (r *ReportsResource) List(ctx *AdminContext, q ListQuery) (*ListResult, error) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	status := ""
	if v, ok := q.Filters["status"].(string); ok {
		status = v
	}
	var userID uint64
	if v, ok := q.Filters["user_id"].(string); ok {
		userID, _ = strconv.ParseUint(v, 10, 64)
	}
	var paid *bool
	if v, ok := q.Filters["paid"].(string); ok && v != "" {
		if strings.EqualFold(v, "true") {
			t := true
			paid = &t
		} else if strings.EqualFold(v, "false") {
			f := false
			paid = &f
		}
	}
	res, err := r.svc.ListReports(context.Background(), status, paid, userID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: res.Items, Total: res.Total, Page: res.Page, PageSize: res.PageSize}, nil
}

func (r *ReportsResource) Detail(ctx *AdminContext, id string) (interface{}, error) {
	rid, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, err
	}
	return r.svc.GetReportDetail(context.Background(), rid)
}

func (r *ReportsResource) Update(ctx *AdminContext, id string, patch map[string]interface{}) (interface{}, error) {
	return nil, nil
}

func (r *ReportsResource) Actions() []Action { return nil }
