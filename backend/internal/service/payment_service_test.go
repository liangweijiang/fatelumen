package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/payment"
	"fatelumen/backend/internal/repository"
)

// ---------- fakes ----------

type fakeFulfillStore struct {
	mu               sync.Mutex
	orders           map[string]*model.Order // sessionID -> order
	processedEvents  map[string]bool         // eventID -> processed
	reportPaidCount  map[uint64]int          // reportID -> call count
	fulfillErr       error                   // force error from FulfillPaidOrder
	forceDuplicate   bool                    // force ErrDuplicateEvent
	forceTransitErr  bool                    // force transit error
	forceAlreadyPaid bool                    // force already-paid path
}

func newFakeFulfillStore() *fakeFulfillStore {
	return &fakeFulfillStore{
		orders:          make(map[string]*model.Order),
		processedEvents: make(map[string]bool),
		reportPaidCount: make(map[uint64]int),
	}
}

func (f *fakeFulfillStore) GetBySessionID(sessionID string) (*model.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.orders[sessionID]
	if !ok {
		return nil, errors.New("not found")
	}
	return o, nil
}

func (f *fakeFulfillStore) FulfillPaidOrder(provider, eventID string, orderID uint64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.fulfillErr != nil {
		return f.fulfillErr
	}
	if f.forceDuplicate {
		return repository.ErrDuplicateEvent
	}

	// Phase 1: validate all conditions (like a real transaction)
	if f.processedEvents[eventID] {
		return repository.ErrDuplicateEvent
	}

	var order *model.Order
	for _, o := range f.orders {
		if o.ID == orderID {
			order = o
			break
		}
	}
	if order == nil {
		return errors.New("order not found")
	}

	if f.forceAlreadyPaid || order.Status == model.OrderStatusPaid {
		f.processedEvents[eventID] = true
		return nil
	}

	if f.forceTransitErr {
		return errors.New("invalid order status transition: paid -> paid")
	}

	if !model.CanTransit(order.Status, model.OrderStatusPaid) {
		return errors.New("invalid order status transition")
	}

	// Phase 2: commit — all state changes happen together
	f.processedEvents[eventID] = true
	order.Status = model.OrderStatusPaid
	if order.ReportID > 0 {
		f.reportPaidCount[order.ReportID]++
	}
	return nil
}

type fakePayProvider struct {
	verifyErr error
	evt       *payment.WebhookEvent
}

func (f *fakePayProvider) Name() string { return "test" }

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
	reg := payment.NewRegistry()
	reg.Register("stripe", pay)
	svc := &PaymentService{reg: reg}

	err := svc.HandleWebhook(context.Background(), "stripe", []byte(`{}`), "bad-sig")
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
	reg := payment.NewRegistry()
	reg.Register("stripe", pay)
	svc := &PaymentService{reg: reg}

	err := svc.HandleWebhook(context.Background(), "stripe", []byte(`{}`), "sig")
	if err != nil {
		t.Fatalf("expected nil for non-completed event, got: %v", err)
	}
}

func TestHandleWebhook_Success(t *testing.T) {
	orders := newFakeFulfillStore()
	orders.orders["cs_test_1"] = &model.Order{
		ID:       10,
		UserID:   1,
		ReportID: 5,
		Status:   model.OrderStatusCreated,
	}

	pay := &fakePayProvider{
		evt: &payment.WebhookEvent{
			EventID:   "evt_123",
			Type:      "checkout.session.completed",
			SessionID: "cs_test_1",
			OrderID:   10,
		},
	}

	reg := payment.NewRegistry()
	reg.Register("stripe", pay)
	svc := &PaymentService{reg: reg, orders: orders}

	err := svc.HandleWebhook(context.Background(), "stripe", []byte(`{}`), "sig")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !orders.processedEvents["evt_123"] {
		t.Error("expected event to be marked as processed")
	}
	if c := orders.reportPaidCount[5]; c != 1 {
		t.Errorf("expected report 5 to be unlocked once, got %d", c)
	}
}

func TestHandleWebhook_DuplicateEvent(t *testing.T) {
	orders := newFakeFulfillStore()
	orders.forceDuplicate = true // simulate duplicate in transaction

	orders.orders["cs_test_1"] = &model.Order{
		ID:       10,
		ReportID: 5,
		Status:   model.OrderStatusCreated,
	}

	pay := &fakePayProvider{
		evt: &payment.WebhookEvent{
			EventID:   "evt_dup",
			Type:      "checkout.session.completed",
			SessionID: "cs_test_1",
			OrderID:   10,
		},
	}

	reg := payment.NewRegistry()
	reg.Register("stripe", pay)
	svc := &PaymentService{reg: reg, orders: orders}

	err := svc.HandleWebhook(context.Background(), "stripe", []byte(`{}`), "sig")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Report must not be unlocked on duplicate
	if c := orders.reportPaidCount[5]; c != 0 {
		t.Errorf("expected report NOT to be unlocked on duplicate, got %d unlocks", c)
	}
	// Order must not be paid
	if o := orders.orders["cs_test_1"]; o.Status != model.OrderStatusCreated {
		t.Errorf("expected order status to remain created on duplicate, got %s", o.Status)
	}
}

func TestHandleWebhook_OrderAlreadyPaid(t *testing.T) {
	orders := newFakeFulfillStore()
	orders.forceAlreadyPaid = true // order was paid by a prior event

	orders.orders["cs_test_1"] = &model.Order{
		ID:       10,
		ReportID: 5,
		Status:   model.OrderStatusCreated, // fake ignores this when forceAlreadyPaid=true
	}

	pay := &fakePayProvider{
		evt: &payment.WebhookEvent{
			EventID:   "evt_new",
			Type:      "checkout.session.completed",
			SessionID: "cs_test_1",
			OrderID:   10,
		},
	}

	reg := payment.NewRegistry()
	reg.Register("stripe", pay)
	svc := &PaymentService{reg: reg, orders: orders}

	err := svc.HandleWebhook(context.Background(), "stripe", []byte(`{}`), "sig")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Event should be recorded (new event_id, dedup passes)
	if !orders.processedEvents["evt_new"] {
		t.Error("expected new event_id to be recorded")
	}
	// Report should NOT be re-unlocked
	if c := orders.reportPaidCount[5]; c != 0 {
		t.Errorf("expected report NOT to be re-unlocked, got %d", c)
	}
}

func TestHandleWebhook_MissingOrderID(t *testing.T) {
	orders := newFakeFulfillStore()

	pay := &fakePayProvider{
		evt: &payment.WebhookEvent{
			EventID:   "evt_123",
			Type:      "checkout.session.completed",
			SessionID: "cs_test_1",
			OrderID:   0, // missing
		},
	}

	reg := payment.NewRegistry()
	reg.Register("stripe", pay)
	svc := &PaymentService{reg: reg, orders: orders}

	err := svc.HandleWebhook(context.Background(), "stripe", []byte(`{}`), "sig")
	if err != nil {
		t.Fatalf("expected nil (200) for missing order_id, got: %v", err)
	}

	if len(orders.processedEvents) > 0 {
		t.Error("expected no events to be processed when order_id is missing")
	}
}

func TestHandleWebhook_OrderNotFoundBySession(t *testing.T) {
	orders := newFakeFulfillStore()
	// No orders stored

	pay := &fakePayProvider{
		evt: &payment.WebhookEvent{
			EventID:   "evt_123",
			Type:      "checkout.session.completed",
			SessionID: "cs_unknown",
			OrderID:   99,
		},
	}

	reg := payment.NewRegistry()
	reg.Register("stripe", pay)
	svc := &PaymentService{reg: reg, orders: orders}

	err := svc.HandleWebhook(context.Background(), "stripe", []byte(`{}`), "sig")
	if err != nil {
		t.Fatalf("expected nil (200) for session not found, got: %v", err)
	}

	if len(orders.processedEvents) > 0 {
		t.Error("expected no events to be processed when session not found")
	}
}

// ---------- Fulfill atomicity tests (via fake) ----------

func TestFulfill_Success(t *testing.T) {
	orders := newFakeFulfillStore()
	orders.orders["cs_test_1"] = &model.Order{
		ID:       10,
		UserID:   1,
		ReportID: 5,
		Status:   model.OrderStatusCreated,
	}

	pay := &fakePayProvider{
		evt: &payment.WebhookEvent{
			EventID:   "evt_001",
			Type:      "checkout.session.completed",
			SessionID: "cs_test_1",
			OrderID:   10,
		},
	}

	reg := payment.NewRegistry()
	reg.Register("stripe", pay)
	svc := &PaymentService{reg: reg, orders: orders}
	err := svc.HandleWebhook(context.Background(), "stripe", []byte(`{}`), "sig")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify order paid
	if orders.orders["cs_test_1"].Status != model.OrderStatusPaid {
		t.Errorf("expected order to be paid, got %s", orders.orders["cs_test_1"].Status)
	}
	// Verify report unlocked
	if orders.reportPaidCount[5] != 1 {
		t.Errorf("expected report 5 unlocked once, got %d", orders.reportPaidCount[5])
	}
	// Verify webhook event recorded
	if !orders.processedEvents["evt_001"] {
		t.Error("expected webhook event to be recorded")
	}
}

func TestFulfill_DuplicateEvent(t *testing.T) {
	orders := newFakeFulfillStore()
	orders.orders["cs_test_1"] = &model.Order{
		ID:       10,
		ReportID: 5,
		Status:   model.OrderStatusCreated,
	}
	// First call succeeds
	orders.processedEvents["evt_001"] = false // will be set to true on first call

	pay := &fakePayProvider{
		evt: &payment.WebhookEvent{
			EventID:   "evt_001",
			Type:      "checkout.session.completed",
			SessionID: "cs_test_1",
			OrderID:   10,
		},
	}

	reg := payment.NewRegistry()
	reg.Register("stripe", pay)
	svc := &PaymentService{reg: reg, orders: orders}

	// First call: succeeds
	err := svc.HandleWebhook(context.Background(), "stripe", []byte(`{}`), "sig")
	if err != nil {
		t.Fatalf("first call unexpected error: %v", err)
	}
	if orders.reportPaidCount[5] != 1 {
		t.Errorf("first call: expected 1 report unlock, got %d", orders.reportPaidCount[5])
	}
	if orders.orders["cs_test_1"].Status != model.OrderStatusPaid {
		t.Errorf("first call: expected order paid, got %s", orders.orders["cs_test_1"].Status)
	}

	// Second call: duplicate event (same eventID)
	err = svc.HandleWebhook(context.Background(), "stripe", []byte(`{}`), "sig")
	if err != nil {
		t.Fatalf("second call unexpected error: %v", err)
	}
	// Must NOT re-unlock report
	if orders.reportPaidCount[5] != 1 {
		t.Errorf("duplicate: expected still 1 report unlock, got %d", orders.reportPaidCount[5])
	}
}

func TestFulfill_AlreadyPaid(t *testing.T) {
	orders := newFakeFulfillStore()
	orders.orders["cs_test_1"] = &model.Order{
		ID:       10,
		ReportID: 5,
		Status:   model.OrderStatusPaid, // already paid
	}

	pay := &fakePayProvider{
		evt: &payment.WebhookEvent{
			EventID:   "evt_002",
			Type:      "checkout.session.completed",
			SessionID: "cs_test_1",
			OrderID:   10,
		},
	}

	reg := payment.NewRegistry()
	reg.Register("stripe", pay)
	svc := &PaymentService{reg: reg, orders: orders}
	err := svc.HandleWebhook(context.Background(), "stripe", []byte(`{}`), "sig")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Report must NOT be re-unlocked
	if orders.reportPaidCount[5] != 0 {
		t.Errorf("expected no report unlock for already-paid order, got %d", orders.reportPaidCount[5])
	}
	// Event should be recorded (new event_id)
	if !orders.processedEvents["evt_002"] {
		t.Error("expected event to be recorded even for already-paid order")
	}
}

func TestFulfill_CannotTransit(t *testing.T) {
	orders := newFakeFulfillStore()
	orders.forceTransitErr = true // simulate invalid transition, transaction rolls back

	orders.orders["cs_test_1"] = &model.Order{
		ID:       10,
		ReportID: 5,
		Status:   model.OrderStatusRefunded, // cannot transit to paid
	}

	pay := &fakePayProvider{
		evt: &payment.WebhookEvent{
			EventID:   "evt_003",
			Type:      "checkout.session.completed",
			SessionID: "cs_test_1",
			OrderID:   10,
		},
	}

	reg := payment.NewRegistry()
	reg.Register("stripe", pay)
	svc := &PaymentService{reg: reg, orders: orders}
	err := svc.HandleWebhook(context.Background(), "stripe", []byte(`{}`), "sig")
	if err != nil {
		t.Fatalf("expected nil (200 to stop retry) for transit error, got: %v", err)
	}

	// Report must NOT be unlocked
	if orders.reportPaidCount[5] != 0 {
		t.Errorf("expected no report unlock on transit error, got %d", orders.reportPaidCount[5])
	}
	// Event must NOT be recorded (transaction rolled back)
	if orders.processedEvents["evt_003"] {
		t.Error("expected event NOT to be recorded on transit failure (rollback)")
	}
	// Order status unchanged
	if orders.orders["cs_test_1"].Status != model.OrderStatusRefunded {
		t.Errorf("expected order status unchanged, got %s", orders.orders["cs_test_1"].Status)
	}
}
