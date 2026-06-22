package resource

import (
	"context"
	"strconv"

	"fatelumen/backend/internal/service"
)

type orderListSvc interface {
	ListOrders(ctx context.Context, status string, userID uint64, page, pageSize int) (*service.AdminOrdersPage, error)
	GetOrderDetail(ctx context.Context, orderID uint64) (*service.AdminOrderDetail, error)
}

type OrdersResource struct {
	svc orderListSvc
}

func NewOrdersResource(svc orderListSvc) *OrdersResource {
	return &OrdersResource{svc: svc}
}

func (r *OrdersResource) Name() string { return "orders" }

func (r *OrdersResource) Schema() []Field {
	return []Field{
		{Key: "id", Label: "ID", Type: "int", Sortable: true},
		{Key: "user_id", Label: "用户ID", Type: "int", Filterable: true},
		{Key: "user_email", Label: "用户邮箱", Type: "string"},
		{Key: "status", Label: "状态", Type: "enum", Enum: []EnumOption{{Value: "pending", Label: "待支付"}, {Value: "paid", Label: "已支付"}, {Value: "refunded", Label: "已退款"}}, Filterable: true},
		{Key: "amount_cents", Label: "金额", Type: "money", Sortable: true},
		{Key: "created_at", Label: "下单时间", Type: "datetime", Sortable: true},
	}
}

func (r *OrdersResource) List(ctx *AdminContext, q ListQuery) (*ListResult, error) {
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
	res, err := r.svc.ListOrders(context.Background(), status, userID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: res.Items, Total: res.Total, Page: res.Page, PageSize: res.PageSize}, nil
}

func (r *OrdersResource) Detail(ctx *AdminContext, id string) (interface{}, error) {
	oid, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, err
	}
	return r.svc.GetOrderDetail(context.Background(), oid)
}

func (r *OrdersResource) Update(ctx *AdminContext, id string, patch map[string]interface{}) (interface{}, error) {
	return nil, nil
}

func (r *OrdersResource) Actions() []Action { return nil }
