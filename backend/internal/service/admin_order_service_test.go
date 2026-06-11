package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/repository"
)

type fakeAdminOrderStore struct {
	orders  []model.Order
	total   int64
	listErr error
	order   *model.Order
	getErr  error
	filter  repository.OrderFilter
	limit   int
	offset  int
}

func newFakeAdminOrderStore() *fakeAdminOrderStore {
	return &fakeAdminOrderStore{}
}

func (f *fakeAdminOrderStore) AdminListOrders(filter repository.OrderFilter, limit, offset int) ([]model.Order, int64, error) {
	f.filter = filter
	f.limit = limit
	f.offset = offset
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	return f.orders, f.total, nil
}

func (f *fakeAdminOrderStore) AdminGetOrderByID(id uint64) (*model.Order, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.order, nil
}

func TestAdminListOrders_Pagination(t *testing.T) {
	store := newFakeAdminOrderStore()
	store.orders = []model.Order{
		{ID: 1, UserID: 10, Status: "paid", AmountCents: 999},
	}
	store.total = 25

	svc := &AdminOrderService{orderRepo: store}

	page, err := svc.ListOrders(context.Background(), "", 0, 2, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Total != 25 {
		t.Errorf("total: want 25, got %d", page.Total)
	}
	if page.Page != 2 {
		t.Errorf("page: want 2, got %d", page.Page)
	}
	if page.PageSize != 20 {
		t.Errorf("pageSize: want 20, got %d", page.PageSize)
	}
	if page.Items[0].ID != 1 {
		t.Errorf("items[0].id: want 1, got %d", page.Items[0].ID)
	}
}

func TestAdminListOrders_PageSizeCap(t *testing.T) {
	store := newFakeAdminOrderStore()
	svc := &AdminOrderService{orderRepo: store}

	page, err := svc.ListOrders(context.Background(), "", 0, 1, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.PageSize != 20 {
		t.Errorf("expected capped to 20, got %d", page.PageSize)
	}
}

func TestAdminListOrders_FilterPassthrough(t *testing.T) {
	store := newFakeAdminOrderStore()
	svc := &AdminOrderService{orderRepo: store}

	svc.ListOrders(context.Background(), "paid", 42, 1, 10)

	if store.filter.Status != "paid" {
		t.Errorf("filter status: want 'paid', got '%s'", store.filter.Status)
	}
	if store.filter.UserID != 42 {
		t.Errorf("filter userID: want 42, got %d", store.filter.UserID)
	}
	if store.limit != 10 {
		t.Errorf("limit: want 10, got %d", store.limit)
	}
	if store.offset != 0 {
		t.Errorf("offset: want 0, got %d", store.offset)
	}
}

func TestAdminListOrders_OffsetCalculation(t *testing.T) {
	store := newFakeAdminOrderStore()
	svc := &AdminOrderService{orderRepo: store}

	// page=3, size=20 → offset=40
	svc.ListOrders(context.Background(), "", 0, 3, 20)
	if store.offset != 40 {
		t.Errorf("offset: want 40, got %d", store.offset)
	}
}

func TestAdminGetOrderDetail_Success(t *testing.T) {
	store := newFakeAdminOrderStore()
	store.order = &model.Order{
		ID:            1,
		UserID:        10,
		ReportID:      5,
		Type:          "report",
		SKU:           "report_single",
		AmountCents:   999,
		Currency:      "usd",
		Status:        "paid",
		Provider:      "stripe",
		ProviderRef:   "cs_test_123",
		ProviderTxnID: "pi_456",
	}

	svc := &AdminOrderService{orderRepo: store}
	detail, err := svc.GetOrderDetail(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.ProviderRef != "cs_test_123" {
		t.Errorf("provider_ref: want cs_test_123, got %s", detail.ProviderRef)
	}
	if detail.ProviderTxnID != "pi_456" {
		t.Errorf("provider_txn_id: want pi_456, got %s", detail.ProviderTxnID)
	}
	if detail.Currency != "usd" {
		t.Errorf("currency: want usd, got %s", detail.Currency)
	}
}

func TestAdminGetOrderDetail_NotFound(t *testing.T) {
	store := newFakeAdminOrderStore()
	store.getErr = errors.New("not found")
	svc := &AdminOrderService{orderRepo: store}

	_, err := svc.GetOrderDetail(context.Background(), 99)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestAdminGetOrderDetail_ProviderMetaIsJSON(t *testing.T) {
	store := newFakeAdminOrderStore()
	store.order = &model.Order{
		ID:           1,
		ProviderMeta: model.JSONRaw(`{"payment_intent":"pi_123","amount":999}`),
	}

	svc := &AdminOrderService{orderRepo: store}
	detail, err := svc.GetOrderDetail(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, _ := json.Marshal(detail)
	s := string(b)

	// Provider meta must be a JSON object, not a base64 string
	if !strings.Contains(s, `"provider_meta":{`) {
		t.Errorf("provider_meta should be a JSON object, got: %s", s)
	}
	if strings.Contains(s, `"payment_intent"`) && !strings.Contains(s, `"pi_123"`) {
		t.Errorf("provider_meta should contain original keys")
	}
	// Must not be base64-encoded
	if strings.Contains(s, `"provider_meta":"`) {
		t.Errorf("provider_meta should NOT be a base64 string")
	}
}
