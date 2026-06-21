package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type fakeOrderSvc struct {
	createFn func(ctx context.Context, userID uint64, in service.CreateOrderInput) (*service.CreateOrderResult, error)
	getFn    func(ctx context.Context, userID, orderID uint64) (*model.Order, error)
	listFn   func(ctx context.Context, userID uint64) ([]model.Order, error)
}

func (f *fakeOrderSvc) CreateOrder(ctx context.Context, userID uint64, in service.CreateOrderInput) (*service.CreateOrderResult, error) {
	return f.createFn(ctx, userID, in)
}

func (f *fakeOrderSvc) GetOrder(ctx context.Context, userID, orderID uint64) (*model.Order, error) {
	return f.getFn(ctx, userID, orderID)
}

func (f *fakeOrderSvc) ListOrders(ctx context.Context, userID uint64) ([]model.Order, error) {
	return f.listFn(ctx, userID)
}

func testOrderHandler(svc orderSvc) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func setupAuthedOrderRouter(h *OrderHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		c.Next()
	})
	r.POST("/api/v1/orders", h.Create)
	r.GET("/api/v1/orders/:id", h.Get)
	r.GET("/api/v1/orders", h.List)
	return r
}

func setupNoAuthOrderRouter(h *OrderHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/orders", h.Create)
	r.GET("/api/v1/orders/:id", h.Get)
	r.GET("/api/v1/orders", h.List)
	return r
}

// --- POST create ---

func TestCreateOrder_Success_Handler(t *testing.T) {
	svc := &fakeOrderSvc{
		createFn: func(ctx context.Context, userID uint64, in service.CreateOrderInput) (*service.CreateOrderResult, error) {
			return &service.CreateOrderResult{
				Order:       &model.Order{ID: 5, UserID: userID, ReportID: in.ReportID, Status: "created"},
				CheckoutURL: "https://checkout.stripe.com/pay/test",
			}, nil
		},
	}
	h := testOrderHandler(svc)
	router := setupAuthedOrderRouter(h)

	req := httptest.NewRequest("POST", "/api/v1/orders",
		strings.NewReader(`{"report_id": 10, "provider": "stripe"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp response.Resp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Code != response.CodeOK {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data map, got %T", resp.Data)
	}
	if data["order_id"] != float64(5) {
		t.Errorf("expected order_id=5, got %v", data["order_id"])
	}
	if data["status"] != "created" {
		t.Errorf("expected status=created, got %v", data["status"])
	}
	if data["checkout_url"] != "https://checkout.stripe.com/pay/test" {
		t.Errorf("expected checkout_url, got %v", data["checkout_url"])
	}
}

func TestCreateOrder_MissingReportID_Handler(t *testing.T) {
	svc := &fakeOrderSvc{}
	h := testOrderHandler(svc)
	router := setupAuthedOrderRouter(h)

	req := httptest.NewRequest("POST", "/api/v1/orders",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp response.Resp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Code != response.CodeBadRequest {
		t.Fatalf("expected code %d, got %d", response.CodeBadRequest, resp.Code)
	}
}

func TestCreateOrder_ServiceError_Handler(t *testing.T) {
	svc := &fakeOrderSvc{
		createFn: func(ctx context.Context, userID uint64, in service.CreateOrderInput) (*service.CreateOrderResult, error) {
			return nil, errors.New("report not found or not owned")
		},
	}
	h := testOrderHandler(svc)
	router := setupAuthedOrderRouter(h)

	req := httptest.NewRequest("POST", "/api/v1/orders",
		strings.NewReader(`{"report_id": 1, "provider": "stripe"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp response.Resp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Code != response.CodeServerError {
		t.Fatalf("expected code %d, got %d", response.CodeServerError, resp.Code)
	}
}

// --- GET /:id ---

func TestGetOrder_Success_Handler(t *testing.T) {
	svc := &fakeOrderSvc{
		getFn: func(ctx context.Context, userID, orderID uint64) (*model.Order, error) {
			return &model.Order{
				ID: orderID, UserID: userID, ReportID: 1,
				Status:      "paid",
				AmountCents: 999,
				Currency:    "usd",
			}, nil
		},
	}
	h := testOrderHandler(svc)
	router := setupAuthedOrderRouter(h)

	req := httptest.NewRequest("GET", "/api/v1/orders/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp response.Resp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Code != response.CodeOK {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestGetOrder_NotFound_Handler(t *testing.T) {
	svc := &fakeOrderSvc{
		getFn: func(ctx context.Context, userID, orderID uint64) (*model.Order, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	h := testOrderHandler(svc)
	router := setupAuthedOrderRouter(h)

	req := httptest.NewRequest("GET", "/api/v1/orders/99", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp response.Resp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Code != response.CodeNotFound {
		t.Fatalf("expected code %d, got %d", response.CodeNotFound, resp.Code)
	}
}

// --- GET list ---

func TestListOrders_Success_Handler(t *testing.T) {
	svc := &fakeOrderSvc{
		listFn: func(ctx context.Context, userID uint64) ([]model.Order, error) {
			return []model.Order{
				{ID: 1, UserID: userID, Status: "paid", AmountCents: 999},
				{ID: 2, UserID: userID, Status: "created", AmountCents: 999},
			}, nil
		},
	}
	h := testOrderHandler(svc)
	router := setupAuthedOrderRouter(h)

	req := httptest.NewRequest("GET", "/api/v1/orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp response.Resp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Code != response.CodeOK {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}

	arr, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected array, got %T", resp.Data)
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 items, got %d", len(arr))
	}
}

// --- Unauthorized tests ---

func TestCreateOrder_Unauthorized_Handler(t *testing.T) {
	svc := &fakeOrderSvc{}
	h := testOrderHandler(svc)
	router := setupNoAuthOrderRouter(h)

	req := httptest.NewRequest("POST", "/api/v1/orders",
		strings.NewReader(`{"report_id": 1}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp response.Resp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Code != response.CodeUnauthorized {
		t.Fatalf("expected code %d, got %d", response.CodeUnauthorized, resp.Code)
	}
}

func TestGetOrder_Unauthorized_Handler(t *testing.T) {
	svc := &fakeOrderSvc{}
	h := testOrderHandler(svc)
	router := setupNoAuthOrderRouter(h)

	req := httptest.NewRequest("GET", "/api/v1/orders/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp response.Resp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Code != response.CodeUnauthorized {
		t.Fatalf("expected code %d, got %d", response.CodeUnauthorized, resp.Code)
	}
}

func TestListOrders_Unauthorized_Handler(t *testing.T) {
	svc := &fakeOrderSvc{}
	h := testOrderHandler(svc)
	router := setupNoAuthOrderRouter(h)

	req := httptest.NewRequest("GET", "/api/v1/orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp response.Resp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Code != response.CodeUnauthorized {
		t.Fatalf("expected code %d, got %d", response.CodeUnauthorized, resp.Code)
	}
}
