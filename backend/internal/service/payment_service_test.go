package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/payment"
)

// ---------- fakes ----------

type fakePayOrderStore struct {
	mu             sync.Mutex
	orders         map[string]*model.Order
	markPaidCalled bool
	markPaidErr    error
}

func newFakePayOrderStore() *fakePayOrderStore {
	return &fakePayOrderStore{
		orders: make(map[string]*model.Order),
	}
}

func (f *fakePayOrderStore) GetBySessionID(sessionID string) (*model.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.orders[sessionID]
	if !ok {
		return nil, errors.New("not found")
	}
	return o, nil
}

func (f *fakePayOrderStore) MarkPaid(orderID uint64, providerTxnID string, meta []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.markPaidCalled = true
	return f.markPaidErr
}

type fakePayReportStore struct {
	mu             sync.Mutex
	markPaidCalled bool
	markPaidErr    error
}

func newFakePayReportStore() *fakePayReportStore {
	return &fakePayReportStore{}
}

func (f *fakePayReportStore) MarkPaid(reportID, orderID uint64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.markPaidCalled = true
	return f.markPaidErr
}

type fakeWebhookEventStore struct {
	mu        sync.Mutex
	processed map[string]bool
	dup       bool
	err       error
}

func newFakeWebhookEventStore() *fakeWebhookEventStore {
	return &fakeWebhookEventStore{
		processed: make(map[string]bool),
	}
}

func (f *fakeWebhookEventStore) MarkProcessed(provider, eventID, eventType string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return false, f.err
	}
	key := provider + ":" + eventID
	if f.processed[key] {
		return true, nil
	}
	if f.dup {
		return true, nil
	}
	f.processed[key] = true
	return false, nil
}

type fakePayProvider struct {
	verifyErr error
	evt       *payment.WebhookEvent
}

func (f *fakePayProvider) CreateCheckout(ctx context.Context, in payment.CheckoutInput) (*payment.CheckoutResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakePayProvider) VerifyAndParse(payload []byte, sigHeader string) (*payment.WebhookEvent, error) {
	if f.verifyErr != nil {
		return nil, f.verifyErr
	}
	return f.evt, nil
}

// ---------- tests ----------

func TestHandleWebhook_VerifyFails(t *testing.T) {
	pay := &fakePayProvider{verifyErr: errors.New("invalid signature")}
	svc := &PaymentService{pay: pay}

	err := svc.HandleWebhook(context.Background(), []byte(`{}`), "bad-sig")
	if err == nil {
		t.Fatal("expected error for failed verification")
	}
}

func TestHandleWebhook_NonCompletedEvent(t *testing.T) {
	pay := &fakePayProvider{
		evt: &payment.WebhookEvent{
			EventID: "evt_123",
			Type:    "payment_intent.succeeded",
		},
	}
	svc := &PaymentService{pay: pay}

	err := svc.HandleWebhook(context.Background(), []byte(`{}`), "sig")
	if err != nil {
		t.Fatalf("expected nil for non-completed event, got: %v", err)
	}
}

func TestHandleWebhook_Success(t *testing.T) {
	orders := newFakePayOrderStore()
	reports := newFakePayReportStore()
	events := newFakeWebhookEventStore()

	// Pre-create an order with session ID
	orders.orders["cs_test_1"] = &model.Order{
		ID:       10,
		UserID:   1,
		ReportID: 5,
		Status:   model.OrderStatusCreated,
	}

	pay := &fakePayProvider{
		evt: &payment.WebhookEvent{
			EventID:         "evt_123",
			Type:            "checkout.session.completed",
			SessionID:       "cs_test_1",
			PaymentIntentID: "pi_456",
			OrderID:         10,
			Raw:             []byte(`{}`),
		},
	}

	svc := &PaymentService{
		pay:     pay,
		orders:  orders,
		reports: reports,
		events:  events,
	}

	err := svc.HandleWebhook(context.Background(), []byte(`{}`), "sig")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !orders.markPaidCalled {
		t.Error("expected MarkPaid on order to be called")
	}
	if !reports.markPaidCalled {
		t.Error("expected MarkPaid on report to be called")
	}
}

func TestHandleWebhook_DuplicateEvent(t *testing.T) {
	orders := newFakePayOrderStore()
	reports := newFakePayReportStore()
	events := newFakeWebhookEventStore()
	events.dup = true // simulate duplicate

	orders.orders["cs_test_1"] = &model.Order{
		ID:       10,
		ReportID: 5,
		Status:   model.OrderStatusCreated,
	}

	pay := &fakePayProvider{
		evt: &payment.WebhookEvent{
			EventID:         "evt_dup",
			Type:            "checkout.session.completed",
			SessionID:       "cs_test_1",
			PaymentIntentID: "pi_456",
			OrderID:         10,
		},
	}

	svc := &PaymentService{
		pay:     pay,
		orders:  orders,
		reports: reports,
		events:  events,
	}

	err := svc.HandleWebhook(context.Background(), []byte(`{}`), "sig")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if orders.markPaidCalled {
		t.Error("expected MarkPaid NOT to be called on duplicate event")
	}
}

func TestHandleWebhook_OrderAlreadyPaid(t *testing.T) {
	orders := newFakePayOrderStore()
	reports := newFakePayReportStore()
	events := newFakeWebhookEventStore()

	orders.orders["cs_test_1"] = &model.Order{
		ID:       10,
		ReportID: 5,
		Status:   model.OrderStatusPaid, // already paid
	}

	pay := &fakePayProvider{
		evt: &payment.WebhookEvent{
			EventID:         "evt_123",
			Type:            "checkout.session.completed",
			SessionID:       "cs_test_1",
			PaymentIntentID: "pi_456",
			OrderID:         10,
		},
	}

	svc := &PaymentService{
		pay:     pay,
		orders:  orders,
		reports: reports,
		events:  events,
	}

	err := svc.HandleWebhook(context.Background(), []byte(`{}`), "sig")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if orders.markPaidCalled {
		t.Error("expected MarkPaid NOT to be called when order already paid")
	}
}

func TestHandleWebhook_MissingOrderID(t *testing.T) {
	orders := newFakePayOrderStore()
	reports := newFakePayReportStore()
	events := newFakeWebhookEventStore()

	pay := &fakePayProvider{
		evt: &payment.WebhookEvent{
			EventID:   "evt_123",
			Type:      "checkout.session.completed",
			SessionID: "cs_test_1",
			OrderID:   0, // missing
		},
	}

	svc := &PaymentService{
		pay:     pay,
		orders:  orders,
		reports: reports,
		events:  events,
	}

	err := svc.HandleWebhook(context.Background(), []byte(`{}`), "sig")
	if err != nil {
		t.Fatalf("expected nil (200) for missing order_id, got: %v", err)
	}

	if orders.markPaidCalled {
		t.Error("expected MarkPaid NOT to be called when order_id is missing")
	}
}
