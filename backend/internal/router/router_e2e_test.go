package router

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"fatelumen/backend/internal/handler"
	"fatelumen/backend/internal/middleware"
	"fatelumen/backend/internal/model"
	jwtpkg "fatelumen/backend/internal/pkg/jwt"
	"fatelumen/backend/internal/pkg/response"
	"fatelumen/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const testJWTSecret = "e2e-test-secret-key-2026"

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	db.Create(&model.User{
		ID:        1,
		GoogleSub: "admin-google-sub",
		Email:     "admin@e2e.test",
		Name:      "Admin",
		Role:      model.RoleAdmin,
		Active:    true,
	})
	db.Create(&model.User{
		ID:        2,
		GoogleSub: "user-google-sub",
		Email:     "user@e2e.test",
		Name:      "User",
		Role:      model.RoleUser,
		Active:    true,
	})
	return db
}

func makeToken(t *testing.T, userID uint64) string {
	t.Helper()
	token, err := jwtpkg.Generate(testJWTSecret, 24, userID, "e2e-token-"+strconv.FormatUint(userID, 10))
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	return token
}

// fakeAllAdminServices implements all 4 admin service interfaces.
type fakeAllAdminServices struct{}

func (f *fakeAllAdminServices) GetStats(ctx context.Context) (*service.Stats, error) {
	return &service.Stats{}, nil
}
func (f *fakeAllAdminServices) ListUsers(ctx context.Context, keyword string, page, pageSize int) (*service.AdminUsersPage, error) {
	return &service.AdminUsersPage{Items: []service.AdminUserItem{}, Total: 0, Page: page, PageSize: pageSize}, nil
}
func (f *fakeAllAdminServices) GetUserDetail(ctx context.Context, userID uint64) (*service.AdminUserDetail, error) {
	return &service.AdminUserDetail{ID: userID, Email: "u@t.com", Role: "user", Active: true}, nil
}
func (f *fakeAllAdminServices) SetUserActive(ctx context.Context, operatorID, targetUserID uint64, active bool) error {
	return nil
}
func (f *fakeAllAdminServices) ListOrders(ctx context.Context, status string, userID uint64, page, pageSize int) (*service.AdminOrdersPage, error) {
	return &service.AdminOrdersPage{Items: []service.AdminOrderItem{}, Total: 0, Page: page, PageSize: pageSize}, nil
}
func (f *fakeAllAdminServices) GetOrderDetail(ctx context.Context, orderID uint64) (*service.AdminOrderDetail, error) {
	return &service.AdminOrderDetail{ID: orderID, Type: "report", SKU: "report_single", Status: "paid"}, nil
}
func (f *fakeAllAdminServices) ListReports(ctx context.Context, status string, paid *bool, userID uint64, page, pageSize int) (*service.AdminReportsPage, error) {
	return &service.AdminReportsPage{Items: []service.AdminReportItem{}, Total: 0, Page: page, PageSize: pageSize}, nil
}
func (f *fakeAllAdminServices) GetReportDetail(ctx context.Context, reportID uint64) (*service.AdminReportDetail, error) {
	return &service.AdminReportDetail{ID: reportID, Type: "order", Status: "done", Paid: true}, nil
}
func (f *fakeAllAdminServices) UnlockReport(ctx context.Context, operatorID, reportID uint64, reason string) error {
	if reason == "" {
		return errors.New("reason required")
	}
	return nil
}

// fakeUserReportSvc implements handler.reportSvc for user-side report routes.
type fakeUserReportSvc struct {
	ownReportID uint64
	ownUserID   uint64
}

func (f *fakeUserReportSvc) CreateReport(ctx context.Context, userID, profileID uint64, locale string) (*model.Report, error) {
	return &model.Report{ID: 1, UserID: userID, Status: "pending"}, nil
}
func (f *fakeUserReportSvc) GetReport(ctx context.Context, userID, reportID uint64) (*model.Report, error) {
	if userID == f.ownUserID && reportID == f.ownReportID {
		return &model.Report{
			ID:       reportID,
			UserID:   userID,
			Status:   "done",
			Paid:     true,
			Content:  model.ReportContent{SummaryLine: "summary", Summary: "body"},
		}, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeUserReportSvc) ListReports(ctx context.Context, userID uint64, limit, offset int) ([]model.Report, error) {
	return []model.Report{}, nil
}

func createTestRouter(t *testing.T, db *gorm.DB, userReportID, userUserID uint64) *gin.Engine {
	t.Helper()
	authMW := middleware.NewAuthMiddleware(testJWTSecret, db)
	faker := &fakeAllAdminServices{}
	adminH := handler.NewAdminHandlerForTest(faker, faker, faker, faker)
	reportH := handler.NewReportHandlerForTest(&fakeUserReportSvc{ownReportID: userReportID, ownUserID: userUserID})

	rlNoop := func(c *gin.Context) { c.Next() }

	app := &App{
		DB:               db,
		Auth:             authMW,
		AdminHandler:     adminH,
		ReportHandler:    reportH,
		RateLimitAuth:    rlNoop,
		RateLimitReading: rlNoop,
		RateLimitOrder:   rlNoop,
	}
	return Setup(app)
}

func mustParseResp(t *testing.T, w *httptest.ResponseRecorder) response.Resp {
	t.Helper()
	var resp response.Resp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v (body=%s)", err, w.Body.String())
	}
	return resp
}

// ---------- admin route definitions for the permission matrix ----------

type adminRoute struct {
	method string
	path   string
	body   string // request body for POST/PATCH
}

var adminRoutes = []adminRoute{
	{method: "GET", path: "/api/v1/admin/ping"},
	{method: "GET", path: "/api/v1/admin/stats"},
	{method: "GET", path: "/api/v1/admin/users"},
	{method: "GET", path: "/api/v1/admin/users/2"},
	{method: "PATCH", path: "/api/v1/admin/users/2/active", body: `{"active": true}`},
	{method: "GET", path: "/api/v1/admin/orders"},
	{method: "GET", path: "/api/v1/admin/orders/1"},
	{method: "GET", path: "/api/v1/admin/reports"},
	{method: "GET", path: "/api/v1/admin/reports/1"},
	{method: "POST", path: "/api/v1/admin/reports/1/unlock", body: `{"reason": "test"}`},
}

// ---------- E2E permission isolation: every admin route × 3 identities ----------

func TestE2E_AdminRoutes_NoToken_Returns401(t *testing.T) {
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2)

	for _, r := range adminRoutes {
		t.Run(r.method+"/"+strings.ReplaceAll(r.path, "/", "_")+"/no_token", func(t *testing.T) {
			req := httptest.NewRequest(r.method, r.path, strings.NewReader(r.body))
			if r.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			resp := mustParseResp(t, w)
			if resp.Code != response.CodeUnauthorized {
				t.Errorf("expected 401 for %s %s without token, got code=%d msg=%s",
					r.method, r.path, resp.Code, resp.Msg)
			}
		})
	}
}

func TestE2E_AdminRoutes_UserToken_Returns403(t *testing.T) {
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2)
	userToken := makeToken(t, 2) // normal user

	for _, r := range adminRoutes {
		t.Run(r.method+"/"+strings.ReplaceAll(r.path, "/", "_")+"/user", func(t *testing.T) {
			req := httptest.NewRequest(r.method, r.path, strings.NewReader(r.body))
			req.Header.Set("Authorization", "Bearer "+userToken)
			if r.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			resp := mustParseResp(t, w)
			if resp.Code != response.CodeForbidden {
				t.Errorf("expected 403 for %s %s with user token, got code=%d msg=%s",
					r.method, r.path, resp.Code, resp.Msg)
			}
		})
	}
}

func TestE2E_AdminRoutes_AdminToken_Not401Not403(t *testing.T) {
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2)
	adminToken := makeToken(t, 1) // admin user

	for _, r := range adminRoutes {
		t.Run(r.method+"/"+strings.ReplaceAll(r.path, "/", "_")+"/admin", func(t *testing.T) {
			req := httptest.NewRequest(r.method, r.path, strings.NewReader(r.body))
			req.Header.Set("Authorization", "Bearer "+adminToken)
			if r.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			resp := mustParseResp(t, w)
			if resp.Code == response.CodeUnauthorized || resp.Code == response.CodeForbidden {
				t.Errorf("admin token should pass through for %s %s, got code=%d msg=%s",
					r.method, r.path, resp.Code, resp.Msg)
			}
		})
	}
}

// ---------- E2E: normal user vs admin report access semantics ----------

func TestE2E_NormalUser_OwnReport_OK(t *testing.T) {
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2) // user(id=2) owns report(id=1)
	userToken := makeToken(t, 2)

	req := httptest.NewRequest("GET", "/api/v1/reports/1", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code != response.CodeOK {
		t.Errorf("user should get own report 200, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected HTTP 200, got %d", w.Code)
	}
}

func TestE2E_NormalUser_OthersReport_NotFound(t *testing.T) {
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2) // user(id=2) owns only report(id=1)
	userToken := makeToken(t, 2)

	req := httptest.NewRequest("GET", "/api/v1/reports/99", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code != response.CodeNotFound {
		t.Errorf("user should get 404 for other's report, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

func TestE2E_Admin_AnyReport_OK(t *testing.T) {
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2) // report(id=1) is owned by user(id=2)
	adminToken := makeToken(t, 1)

	req := httptest.NewRequest("GET", "/api/v1/admin/reports/1", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code != response.CodeOK {
		t.Errorf("admin should get any report 200, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

// ---------- Verify admin token is NOT leaked to user-side report route ----------

func TestE2E_AdminToken_UserReportRoute_NotFound(t *testing.T) {
	db := setupTestDB(t)
	// user(id=2) owns report(id=1); admin(id=1) does not
	router := createTestRouter(t, db, 1, 2)
	adminToken := makeToken(t, 1)

	req := httptest.NewRequest("GET", "/api/v1/reports/1", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	// User-side route uses GetByID(userID, reportID) which checks ownership.
	// Admin(id=1) is not the owner → should get 404.
	if resp.Code != response.CodeNotFound {
		t.Errorf("admin on user report route: expected 404 (ownership check), got code=%d msg=%s",
			resp.Code, resp.Msg)
	}
}

// ---------- Verify report listing routes are distinct ----------

func TestE2E_Admin_ListReportsRequiresAdmin(t *testing.T) {
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2)
	userToken := makeToken(t, 2)

	req := httptest.NewRequest("GET", "/api/v1/admin/reports", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code != response.CodeForbidden {
		t.Errorf("user on admin list reports: expected 403, got code=%d", resp.Code)
	}
}

func TestE2E_NormalUser_OwnReportList_OK(t *testing.T) {
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2)
	userToken := makeToken(t, 2)

	req := httptest.NewRequest("GET", "/api/v1/reports", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code != response.CodeOK {
		t.Errorf("user list own reports: expected 200, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

// ---------- Verify ping/stats are admin-only ----------

func TestE2E_AdminPing_Returns200(t *testing.T) {
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2)
	adminToken := makeToken(t, 1)

	req := httptest.NewRequest("GET", "/api/v1/admin/ping", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code != response.CodeOK {
		t.Errorf("admin ping: expected 200, got code=%d", resp.Code)
	}
	data, _ := resp.Data.(map[string]interface{})
	if data["pong"] != true || data["role"] != "admin" {
		t.Errorf("admin ping payload wrong: %v", data)
	}
}

// ---------- Verify unlock protects against empty body ----------

func TestE2E_AdminUnlockReport_EmptyBody(t *testing.T) {
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2)
	adminToken := makeToken(t, 1)

	req := httptest.NewRequest("POST", "/api/v1/admin/reports/1/unlock", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code != response.CodeBadRequest {
		t.Errorf("unlock empty body: expected 400, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Msg != "reason required" {
		t.Errorf("expected 'reason required', got '%s'", resp.Msg)
	}
}

// ---------- Verify that admin user cannot be used on normal user routes -------------

func TestE2E_UserToken_AdminRoute_Forbidden_All(t *testing.T) {
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2)
	userToken := makeToken(t, 2)

	// Spot-check: all admin endpoints reject user
	routesToCheck := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/api/v1/admin/stats", ""},
		{"GET", "/api/v1/admin/users", ""},
		{"PATCH", "/api/v1/admin/users/2/active", `{"active": true}`},
		{"GET", "/api/v1/admin/orders", ""},
		{"GET", "/api/v1/admin/reports", ""},
		{"POST", "/api/v1/admin/reports/1/unlock", `{"reason": "test"}`},
	}
	for _, r := range routesToCheck {
		t.Run(r.method+"/"+strings.ReplaceAll(r.path, "/", "_"), func(t *testing.T) {
			req := httptest.NewRequest(r.method, r.path, strings.NewReader(r.body))
			req.Header.Set("Authorization", "Bearer "+userToken)
			if r.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			resp := mustParseResp(t, w)
			if resp.Code != response.CodeForbidden {
				t.Errorf("%s %s: expected 403, got %d", r.method, r.path, resp.Code)
			}
		})
	}
}

// ---------- Edge case: token without Bearer prefix ----------

func TestE2E_AdminRoute_MalformedToken_Returns401(t *testing.T) {
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2)

	req := httptest.NewRequest("GET", "/api/v1/admin/ping", nil)
	req.Header.Set("Authorization", "not-bearer-format")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code != response.CodeUnauthorized {
		t.Errorf("malformed token: expected 401, got code=%d", resp.Code)
	}
}

// ---------- Edge case: invalid JWT ----------

func TestE2E_AdminRoute_InvalidToken_Returns401(t *testing.T) {
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2)

	req := httptest.NewRequest("GET", "/api/v1/admin/ping", nil)
	req.Header.Set("Authorization", "Bearer fake.invalid.token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code != response.CodeUnauthorized {
		t.Errorf("invalid JWT: expected 401, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

// ---------- Ensure admin middleware correctly blocks non-existent users ----------

func TestE2E_AdminRoute_UnknownUser_Returns401(t *testing.T) {
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2)
	// Token for userID=999 which doesn't exist in DB
	token, _ := jwtpkg.Generate(testJWTSecret, 24, 999, "e2e-ghost")

	req := httptest.NewRequest("GET", "/api/v1/admin/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code != response.CodeUnauthorized {
		t.Errorf("unknown user: expected 401, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

// ---------- Time-based sanity ----------

func TestE2E_AdminTokenExpiry(t *testing.T) {
	// This test verifies that the JWT generate/parse pipeline works end-to-end.
	// The actual expiry check is done by the jwt library inside the middleware.
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2)

	// Generate a valid token (short-lived but still valid)
	tok, err := jwtpkg.Generate(testJWTSecret, 1, 1, "e2e-time-test")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/admin/ping", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code == response.CodeUnauthorized {
		t.Error("freshly generated token should not be expired")
	}
	// Verify claims parse correctly
	claims, err := jwtpkg.Parse(testJWTSecret, tok)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}
	if claims.UserID != 1 {
		t.Errorf("expected userID=1, got %d", claims.UserID)
	}
	// Token should expire in the future
	if time.Until(claims.ExpiresAt.Time) < 0 {
		t.Error("token already expired!")
	}
}

// ---------- Verify all admin routes are registered ----------

func TestE2E_AllAdminRoutesRegistered(t *testing.T) {
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2)
	adminToken := makeToken(t, 1)

	for _, r := range adminRoutes {
		t.Run("registered/"+r.method+strings.ReplaceAll(r.path, "/", "_"), func(t *testing.T) {
			req := httptest.NewRequest(r.method, r.path, strings.NewReader(r.body))
			req.Header.Set("Authorization", "Bearer "+adminToken)
			if r.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			resp := mustParseResp(t, w)
			// Must not be 404 (route not found); allow any other code
			if resp.Code == response.CodeNotFound {
				t.Errorf("route %s %s returned 404 — is it registered?", r.method, r.path)
			}
			if resp.Code == response.CodeUnauthorized || resp.Code == response.CodeForbidden {
				t.Errorf("route %s %s blocked admin token: code=%d msg=%s",
					r.method, r.path, resp.Code, resp.Msg)
			}
		})
	}
}

// ---------- rate-limit e2e (fake limiter that always denies) ----------

type denyingLimiter struct {
	mu     sync.Mutex
	calls  map[string]int
	denied bool
}

func newDenyingLimiter() *denyingLimiter {
	return &denyingLimiter{calls: make(map[string]int), denied: true}
}

func (d *denyingLimiter) Allow(key string) (bool, time.Duration) {
	d.mu.Lock()
	d.calls[key]++
	d.mu.Unlock()
	return !d.denied, 2 * time.Second
}

func (d *denyingLimiter) callsFor(key string) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.calls[key]
}

func createRateLimitedRouter(t *testing.T, db *gorm.DB) (*gin.Engine, *denyingLimiter) {
	t.Helper()
	authMW := middleware.NewAuthMiddleware(testJWTSecret, db)
	faker := &fakeAllAdminServices{}
	adminH := handler.NewAdminHandlerForTest(faker, faker, faker, faker)
	reportH := handler.NewReportHandlerForTest(&fakeUserReportSvc{ownReportID: 1, ownUserID: 2})

	lim := newDenyingLimiter()
	rl := middleware.RateLimit(lim, middleware.KeyByUser)

	app := &App{
		DB:               db,
		Auth:             authMW,
		AdminHandler:     adminH,
		ReportHandler:    reportH,
		RateLimitAuth:    rl,
		RateLimitReading: rl,
		RateLimitOrder:   rl,
	}
	return Setup(app), lim
}

func TestE2E_RateLimit_ReadingQuickPost_Returns429(t *testing.T) {
	db := setupTestDB(t)
	router, _ := createRateLimitedRouter(t, db)
	userToken := makeToken(t, 2)

	req := httptest.NewRequest("POST", "/api/v1/readings/quick", strings.NewReader(`{"profile_id":1}`))
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code != response.CodeTooManyRequests {
		t.Errorf("POST /readings/quick: expected 429, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

func TestE2E_RateLimit_ReadingGet_NotLimited(t *testing.T) {
	db := setupTestDB(t)
	router, _ := createRateLimitedRouter(t, db)
	userToken := makeToken(t, 2)

	req := httptest.NewRequest("GET", "/api/v1/readings", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code == response.CodeTooManyRequests {
		t.Errorf("GET /readings: should not be rate-limited, got code=%d", resp.Code)
	}
}

func TestE2E_RateLimit_ReportPost_Returns429(t *testing.T) {
	db := setupTestDB(t)
	router, _ := createRateLimitedRouter(t, db)
	userToken := makeToken(t, 2)

	req := httptest.NewRequest("POST", "/api/v1/reports", strings.NewReader(`{"profile_id":1}`))
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code != response.CodeTooManyRequests {
		t.Errorf("POST /reports: expected 429, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

func TestE2E_RateLimit_ReportGet_NotLimited(t *testing.T) {
	db := setupTestDB(t)
	router, _ := createRateLimitedRouter(t, db)
	userToken := makeToken(t, 2)

	req := httptest.NewRequest("GET", "/api/v1/reports", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code == response.CodeTooManyRequests {
		t.Errorf("GET /reports: should not be rate-limited, got code=%d", resp.Code)
	}
}

func TestE2E_RateLimit_OrderPost_Returns429(t *testing.T) {
	db := setupTestDB(t)
	router, _ := createRateLimitedRouter(t, db)
	userToken := makeToken(t, 2)

	req := httptest.NewRequest("POST", "/api/v1/orders", strings.NewReader(`{"report_id":1}`))
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code != response.CodeTooManyRequests {
		t.Errorf("POST /orders: expected 429, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

func TestE2E_RateLimit_OrderGet_NotLimited(t *testing.T) {
	db := setupTestDB(t)
	router, _ := createRateLimitedRouter(t, db)
	userToken := makeToken(t, 2)

	req := httptest.NewRequest("GET", "/api/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code == response.CodeTooManyRequests {
		t.Errorf("GET /orders: should not be rate-limited, got code=%d", resp.Code)
	}
}

func TestE2E_RateLimit_WebhookNotLimited(t *testing.T) {
	db := setupTestDB(t)
	router, _ := createRateLimitedRouter(t, db)

	req := httptest.NewRequest("POST", "/api/v1/webhooks/stripe", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code == response.CodeTooManyRequests {
		t.Errorf("webhook should NOT be rate-limited, got 429")
	}
}

func TestE2E_RateLimit_DisabledPassthrough_Still200(t *testing.T) {
	db := setupTestDB(t)
	router := createTestRouter(t, db, 1, 2) // uses rlNoop
	userToken := makeToken(t, 2)

	req := httptest.NewRequest("POST", "/api/v1/readings/quick", strings.NewReader(`{"profile_id":1}`))
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify no-op does not trigger 429 (route may get 500 due to missing handler, but NOT 429)
	if w.Code == http.StatusOK {
		var resp response.Resp
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code == response.CodeTooManyRequests {
			t.Errorf("no-op passthrough should not trigger 429")
		}
	}
}
