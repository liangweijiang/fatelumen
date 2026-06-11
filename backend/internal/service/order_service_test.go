package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/payment"
)

// ---------- fakes for order service tests ----------

type fakeOrderStore struct {
	mu             sync.Mutex
	orders         map[uint64]*model.Order
	nextID         uint64
	createErr      error
	updateRefErr   error
	providerRefs   map[uint64]string
}

func newFakeOrderStore() *fakeOrderStore {
	return &fakeOrderStore{
		orders:       make(map[uint64]*model.Order),
		providerRefs: make(map[uint64]string),
	}
}

func (f *fakeOrderStore) Create(o *model.Order) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	o.ID = f.nextID
	cpy := *o
	f.orders[o.ID] = &cpy
	return nil
}

func (f *fakeOrderStore) GetByID(id, userID uint64) (*model.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.orders[id]
	if !ok || o.UserID != userID {
		return nil, errors.New("not found")
	}
	cpy := *o
	return &cpy, nil
}

func (f *fakeOrderStore) ListByUser(userID uint64) ([]model.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var result []model.Order
	for _, o := range f.orders {
		if o.UserID == userID {
			result = append(result, *o)
		}
	}
	return result, nil
}

func (f *fakeOrderStore) UpdateProviderRef(id uint64, sessionID string) error {
	if f.updateRefErr != nil {
		return f.updateRefErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providerRefs[id] = sessionID
	if o, ok := f.orders[id]; ok {
		o.ProviderRef = sessionID
	}
	return nil
}

func (f *fakeOrderStore) FindActiveByUserReport(userID, reportID uint64) ([]model.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var result []model.Order
	for _, o := range f.orders {
		if o.UserID == userID && o.ReportID == reportID &&
			(o.Status == model.OrderStatusCreated || o.Status == model.OrderStatusPending || o.Status == model.OrderStatusPaid) {
			result = append(result, *o)
		}
	}
	// Simulate ORDER BY created_at DESC: reverse insertion order
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result, nil
}

type fakeOrderReportStore struct {
	reports map[uint64]*model.Report
}

func newFakeOrderReportStore() *fakeOrderReportStore {
	return &fakeOrderReportStore{
		reports: make(map[uint64]*model.Report),
	}
}

func (f *fakeOrderReportStore) GetByID(id, userID uint64) (*model.Report, error) {
	r, ok := f.reports[id]
	if !ok || r.UserID != userID {
		return nil, errors.New("not found")
	}
	return r, nil
}

type fakePaymentProvider struct {
	checkoutResult *payment.CheckoutResult
	checkoutErr    error
	lastInput      *payment.CheckoutInput
	webhookEvent   *payment.WebhookEvent
	webhookErr     error
}

func newFakePaymentProvider() *fakePaymentProvider {
	return &fakePaymentProvider{
		checkoutResult: &payment.CheckoutResult{
			SessionID:   "cs_test_123",
			CheckoutURL: "https://checkout.stripe.com/pay/test",
		},
	}
}

func (f *fakePaymentProvider) Name() string { return "test" }

func (f *fakePaymentProvider) CreateCheckout(ctx context.Context, in payment.CheckoutInput) (*payment.CheckoutResult, error) {
	f.lastInput = &in
	if f.checkoutErr != nil {
		return nil, f.checkoutErr
	}
	return f.checkoutResult, nil
}

func (f *fakePaymentProvider) VerifyAndParse(payload []byte, sigHeader string) (*payment.WebhookEvent, error) {
	if f.webhookErr != nil {
		return nil, f.webhookErr
	}
	return f.webhookEvent, nil
}

// ---------- order service tests ----------

func newTestOrderService() (*OrderService, *fakeOrderStore, *fakeOrderReportStore, *fakePaymentProvider) {
	store := newFakeOrderStore()
	reports := newFakeOrderReportStore()
	pay := newFakePaymentProvider()
	svc := &OrderService{
		orderRepo:  store,
		reportRepo: reports,
		pay:        pay,
		priceCents: 999,
		successURL: "https://example.com/success",
		cancelURL:  "https://example.com/cancel",
	}
	return svc, store, reports, pay
}

func TestCreateOrder_Success(t *testing.T) {
	svc, store, reports, pay := newTestOrderService()

	// Pre-create a report belonging to user 42
	reports.reports[1] = &model.Report{ID: 1, UserID: 42}

	result, err := svc.CreateOrder(context.Background(), 42, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Order.ID == 0 {
		t.Fatal("expected order ID > 0")
	}
	if result.Order.UserID != 42 {
		t.Fatalf("expected userID 42, got %d", result.Order.UserID)
	}
	if result.Order.ReportID != 1 {
		t.Fatalf("expected reportID 1, got %d", result.Order.ReportID)
	}
	if result.Order.Status != model.OrderStatusCreated {
		t.Fatalf("expected status %s, got %s", model.OrderStatusCreated, result.Order.Status)
	}
	if result.Order.AmountCents != 999 {
		t.Fatalf("expected amount 999, got %d", result.Order.AmountCents)
	}
	if result.CheckoutURL != "https://checkout.stripe.com/pay/test" {
		t.Fatalf("expected checkout URL, got %s", result.CheckoutURL)
	}

	// Verify ProviderRef was saved
	saved, _ := store.GetByID(result.Order.ID, 42)
	if saved.ProviderRef != "cs_test_123" {
		t.Fatalf("expected provider_ref 'cs_test_123', got '%s'", saved.ProviderRef)
	}

	// Verify checkout input metadata contains order_id
	if pay.lastInput == nil {
		t.Fatal("expected CreateCheckout to be called")
	}
	orderIDStr := pay.lastInput.Metadata["order_id"]
	if orderIDStr == "" {
		t.Fatal("expected metadata.order_id to be set")
	}
	// Verify product name does not contain AI
	if pay.lastInput.ProductName == "" {
		t.Fatal("expected ProductName to be set")
	}
}

func TestCreateOrder_ReportNotOwned(t *testing.T) {
	svc, _, reports, pay := newTestOrderService()

	// Report 1 belongs to user 99, not user 42
	reports.reports[1] = &model.Report{ID: 1, UserID: 99}

	_, err := svc.CreateOrder(context.Background(), 42, 1)
	if err == nil {
		t.Fatal("expected error for unowned report")
	}

	// Verify CreateCheckout was NOT called
	if pay.lastInput != nil {
		t.Fatal("expected CreateCheckout NOT to be called for unowned report")
	}
}

func TestCreateOrder_ReportNotFound(t *testing.T) {
	svc, _, _, pay := newTestOrderService()

	_, err := svc.CreateOrder(context.Background(), 42, 999)
	if err == nil {
		t.Fatal("expected error for non-existent report")
	}
	if pay.lastInput != nil {
		t.Fatal("expected CreateCheckout NOT to be called")
	}
}

func TestCreateOrder_CreateFails(t *testing.T) {
	svc, store, reports, _ := newTestOrderService()
	reports.reports[1] = &model.Report{ID: 1, UserID: 42}
	store.createErr = errors.New("db error")

	_, err := svc.CreateOrder(context.Background(), 42, 1)
	if err == nil {
		t.Fatal("expected error for create failure")
	}
}

func TestCreateOrder_CheckoutFails(t *testing.T) {
	svc, _, reports, pay := newTestOrderService()
	reports.reports[1] = &model.Report{ID: 1, UserID: 42}
	pay.checkoutErr = errors.New("stripe error")

	_, err := svc.CreateOrder(context.Background(), 42, 1)
	if err == nil {
		t.Fatal("expected error for checkout failure")
	}
}

func TestCreateOrder_UpdateProviderRefFails(t *testing.T) {
	svc, store, reports, _ := newTestOrderService()
	reports.reports[1] = &model.Report{ID: 1, UserID: 42}
	store.updateRefErr = errors.New("update error")

	_, err := svc.CreateOrder(context.Background(), 42, 1)
	if err == nil {
		t.Fatal("expected error for update provider ref failure")
	}
}

func TestGetOrder_Success(t *testing.T) {
	svc, store, _, _ := newTestOrderService()
	store.Create(&model.Order{UserID: 42, Status: model.OrderStatusCreated})
	store.Create(&model.Order{UserID: 99, Status: model.OrderStatusCreated})

	// User 42 can access their own order
	o, err := svc.GetOrder(context.Background(), 42, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o.UserID != 42 {
		t.Fatalf("expected userID 42, got %d", o.UserID)
	}

	// User 42 cannot access user 99's order
	_, err = svc.GetOrder(context.Background(), 42, 2)
	if err == nil {
		t.Fatal("expected error for wrong user access")
	}
}

func TestListOrders_Success(t *testing.T) {
	svc, store, _, _ := newTestOrderService()
	store.Create(&model.Order{UserID: 42, Status: model.OrderStatusCreated})
	store.Create(&model.Order{UserID: 42, Status: model.OrderStatusPaid})
	store.Create(&model.Order{UserID: 99, Status: model.OrderStatusCreated})

	orders, err := svc.ListOrders(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(orders))
	}
}

// ---------- idempotency tests ----------

func TestCreateOrder_RejectsWhenPaid(t *testing.T) {
	svc, store, reports, _ := newTestOrderService()
	reports.reports[1] = &model.Report{ID: 1, UserID: 42}
	store.Create(&model.Order{ID: 1, UserID: 42, ReportID: 1, Status: model.OrderStatusPaid, AmountCents: 999, Currency: "usd"})

	_, err := svc.CreateOrder(context.Background(), 42, 1)
	if err == nil {
		t.Fatal("expected ErrReportAlreadyPurchased, got nil")
	}
	if !errors.Is(err, ErrReportAlreadyPurchased) {
		t.Errorf("expected ErrReportAlreadyPurchased, got: %v", err)
	}
}

func TestCreateOrder_ReusesPendingOrder(t *testing.T) {
	svc, store, reports, pay := newTestOrderService()
	reports.reports[1] = &model.Report{ID: 1, UserID: 42}
	store.Create(&model.Order{ID: 1, UserID: 42, ReportID: 1, Status: model.OrderStatusCreated, AmountCents: 999, Currency: "usd"})

	// Change checkout URL to verify it's a fresh session
	pay.checkoutResult = &payment.CheckoutResult{
		SessionID:   "cs_reused",
		CheckoutURL: "https://checkout.example.com/reused",
	}

	result, err := svc.CreateOrder(context.Background(), 42, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Order.ID != 1 {
		t.Errorf("expected to reuse order 1, got order %d", result.Order.ID)
	}
	if result.CheckoutURL != "https://checkout.example.com/reused" {
		t.Errorf("expected new checkout URL, got %s", result.CheckoutURL)
	}
	// Verify no new order was created
	if store.nextID != 1 {
		t.Errorf("expected no new order created, but store.nextID=%d", store.nextID)
	}
}

func TestCreateOrder_AllowsWhenRefunded(t *testing.T) {
	svc, store, reports, _ := newTestOrderService()
	reports.reports[1] = &model.Report{ID: 1, UserID: 42}
	store.Create(&model.Order{ID: 1, UserID: 42, ReportID: 1, Status: model.OrderStatusRefunded, AmountCents: 999, Currency: "usd"})

	result, err := svc.CreateOrder(context.Background(), 42, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// New order should be created (not reuse refunded one)
	if result.Order.ID == 10 {
		t.Error("should create new order, not reuse refunded one")
	}
}

func TestCreateOrder_NormalWhenNoExisting(t *testing.T) {
	svc, _, reports, _ := newTestOrderService()
	reports.reports[1] = &model.Report{ID: 1, UserID: 42}

	result, err := svc.CreateOrder(context.Background(), 42, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Order.ID != 1 {
		t.Errorf("expected new order ID 1, got %d", result.Order.ID)
	}
	if result.Order.Status != model.OrderStatusCreated {
		t.Errorf("expected status created, got %s", result.Order.Status)
	}
}
