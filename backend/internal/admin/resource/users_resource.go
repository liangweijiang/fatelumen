package resource

import (
	"context"
	"strconv"

	"fatelumen/backend/internal/service"
)

// userListSvc 抽象出本资源需要的 service 能力(便于测试注入)。
type userListSvc interface {
	ListUsers(ctx context.Context, keyword string, page, pageSize int) (*service.AdminUsersPage, error)
	GetUserDetail(ctx context.Context, userID uint64) (*service.AdminUserDetail, error)
}

// UsersResource 用户资源(只读 + 后续可加 ban action)。
type UsersResource struct {
	svc userListSvc
}

func NewUsersResource(svc userListSvc) *UsersResource {
	return &UsersResource{svc: svc}
}

func (r *UsersResource) Name() string { return "users" }

func (r *UsersResource) Schema() []Field {
	return []Field{
		{Key: "id", Label: "ID", Type: "int", Sortable: true},
		{Key: "email", Label: "邮箱", Type: "string", Searchable: true, Filterable: true},
		{Key: "name", Label: "昵称", Type: "string", Searchable: true},
		{Key: "role", Label: "角色", Type: "enum", Enum: []EnumOption{{Value: "user", Label: "用户"}, {Value: "admin", Label: "管理员"}}, Filterable: true},
		{Key: "active", Label: "启用", Type: "bool", Filterable: true},
		{Key: "unlimited", Label: "无限体验", Type: "bool", Filterable: true},
		{Key: "created_at", Label: "注册时间", Type: "datetime", Sortable: true},
	}
}

func (r *UsersResource) List(ctx *AdminContext, q ListQuery) (*ListResult, error) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	res, err := r.svc.ListUsers(context.Background(), q.Search, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: res.Items, Total: res.Total, Page: res.Page, PageSize: res.PageSize}, nil
}

func (r *UsersResource) Detail(ctx *AdminContext, id string) (interface{}, error) {
	uid, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, err
	}
	return r.svc.GetUserDetail(context.Background(), uid)
}

func (r *UsersResource) Update(ctx *AdminContext, id string, patch map[string]interface{}) (interface{}, error) {
	return nil, nil
}

func (r *UsersResource) Actions() []Action { return nil }
