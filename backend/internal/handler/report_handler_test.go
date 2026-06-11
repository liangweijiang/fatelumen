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

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type fakeReportSvc struct {
	createFn func(ctx context.Context, userID, profileID uint64, locale string) (*model.Report, error)
	getFn    func(ctx context.Context, userID, reportID uint64) (*model.Report, error)
	listFn   func(ctx context.Context, userID uint64, limit, offset int) ([]model.Report, error)
}

func (f *fakeReportSvc) CreateReport(ctx context.Context, userID, profileID uint64, locale string) (*model.Report, error) {
	return f.createFn(ctx, userID, profileID, locale)
}

func (f *fakeReportSvc) GetReport(ctx context.Context, userID, reportID uint64) (*model.Report, error) {
	return f.getFn(ctx, userID, reportID)
}

func (f *fakeReportSvc) ListReports(ctx context.Context, userID uint64, limit, offset int) ([]model.Report, error) {
	return f.listFn(ctx, userID, limit, offset)
}

func testHandler(svc reportSvc) *ReportHandler {
	return &ReportHandler{svc: svc}
}

// setupAuthedRouter 注册路由并注入认证中间件（设置 user_id=1）。
func setupAuthedRouter(h *ReportHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		c.Next()
	})

	r.POST("/api/v1/reports", h.Create)
	r.GET("/api/v1/reports/:id", h.Get)
	r.GET("/api/v1/reports", h.List)

	return r
}

// setupNoAuthRouter 注册路由但不注入认证中间件。
func setupNoAuthRouter(h *ReportHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/api/v1/reports", h.Create)
	r.GET("/api/v1/reports/:id", h.Get)
	r.GET("/api/v1/reports", h.List)

	return r
}

func newReq(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func parseResp(t *testing.T, w *httptest.ResponseRecorder) response.Resp {
	t.Helper()
	var resp response.Resp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	return resp
}

// --- POST create ---

func TestCreateReport_Success(t *testing.T) {
	svc := &fakeReportSvc{
		createFn: func(ctx context.Context, userID, profileID uint64, locale string) (*model.Report, error) {
			if locale == "" {
				locale = "en"
			}
			return &model.Report{ID: 10, UserID: userID, ProfileID: profileID, Locale: locale, Status: "pending"}, nil
		},
	}
	h := testHandler(svc)
	router := setupAuthedRouter(h)

	req := newReq("POST", "/api/v1/reports", `{"profile_id": 42}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResp(t, w)
	if resp.Code != response.CodeOK {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data map, got %T", resp.Data)
	}
	if data["report_id"] != float64(10) {
		t.Errorf("expected report_id=10, got %v", data["report_id"])
	}
	if data["status"] != "pending" {
		t.Errorf("expected status=pending, got %v", data["status"])
	}
}

func TestCreateReport_MissingProfileID(t *testing.T) {
	svc := &fakeReportSvc{}
	h := testHandler(svc)
	router := setupAuthedRouter(h)

	req := newReq("POST", "/api/v1/reports", `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResp(t, w)
	if resp.Code != response.CodeBadRequest {
		t.Fatalf("expected code %d, got %d", response.CodeBadRequest, resp.Code)
	}
}

func TestCreateReport_InvalidJSON(t *testing.T) {
	svc := &fakeReportSvc{}
	h := testHandler(svc)
	router := setupAuthedRouter(h)

	req := newReq("POST", "/api/v1/reports", `not-json`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResp(t, w)
	if resp.Code != response.CodeBadRequest {
		t.Fatalf("expected code %d, got %d", response.CodeBadRequest, resp.Code)
	}
}

func TestCreateReport_ServiceError(t *testing.T) {
	svc := &fakeReportSvc{
		createFn: func(ctx context.Context, userID, profileID uint64, locale string) (*model.Report, error) {
			return nil, errors.New("queue full")
		},
	}
	h := testHandler(svc)
	router := setupAuthedRouter(h)

	req := newReq("POST", "/api/v1/reports", `{"profile_id": 1}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResp(t, w)
	if resp.Code != response.CodeServerError {
		t.Fatalf("expected code %d, got %d", response.CodeServerError, resp.Code)
	}
}

// --- GET /:id ---

func TestGetReport_SuccessDone(t *testing.T) {
	svc := &fakeReportSvc{
		getFn: func(ctx context.Context, userID, reportID uint64) (*model.Report, error) {
			return &model.Report{
				ID: reportID, UserID: userID, ProfileID: 1,
				Status: "done",
				PDFURL: "https://cdn.example.com/report/1.pdf",
				Content: model.ReportContent{Locale: "en", SummaryLine: "test"},
			}, nil
		},
	}
	h := testHandler(svc)
	router := setupAuthedRouter(h)

	req := newReq("GET", "/api/v1/reports/1", "")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResp(t, w)
	if resp.Code != response.CodeOK {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data map, got %T", resp.Data)
	}
	if data["status"] != "done" {
		t.Errorf("expected status=done, got %v", data["status"])
	}
	if data["pdf_url"] != "https://cdn.example.com/report/1.pdf" {
		t.Errorf("expected pdf_url, got %v", data["pdf_url"])
	}
}

func TestGetReport_NotFound_WrongUser(t *testing.T) {
	svc := &fakeReportSvc{
		getFn: func(ctx context.Context, userID, reportID uint64) (*model.Report, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	h := testHandler(svc)
	router := setupAuthedRouter(h)

	req := newReq("GET", "/api/v1/reports/99", "")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResp(t, w)
	if resp.Code != response.CodeNotFound {
		t.Fatalf("expected code %d, got %d", response.CodeNotFound, resp.Code)
	}
}

func TestGetReport_InvalidID(t *testing.T) {
	svc := &fakeReportSvc{}
	h := testHandler(svc)
	router := setupAuthedRouter(h)

	req := newReq("GET", "/api/v1/reports/abc", "")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResp(t, w)
	if resp.Code != response.CodeBadRequest {
		t.Fatalf("expected code %d, got %d", response.CodeBadRequest, resp.Code)
	}
}

// --- GET list ---

func TestListReports_Success(t *testing.T) {
	svc := &fakeReportSvc{
		listFn: func(ctx context.Context, userID uint64, limit, offset int) ([]model.Report, error) {
			return []model.Report{
				{ID: 1, UserID: userID, Status: "done", PDFURL: "https://cdn.example.com/1.pdf"},
				{ID: 2, UserID: userID, Status: "pending"},
			}, nil
		},
	}
	h := testHandler(svc)
	router := setupAuthedRouter(h)

	req := newReq("GET", "/api/v1/reports?limit=10&offset=0", "")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := parseResp(t, w)
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

func TestListReports_EmptyList(t *testing.T) {
	svc := &fakeReportSvc{
		listFn: func(ctx context.Context, userID uint64, limit, offset int) ([]model.Report, error) {
			return []model.Report{}, nil
		},
	}
	h := testHandler(svc)
	router := setupAuthedRouter(h)

	req := newReq("GET", "/api/v1/reports", "")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResp(t, w)
	if resp.Code != response.CodeOK {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	arr, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected array, got %T", resp.Data)
	}
	if len(arr) != 0 {
		t.Fatalf("expected 0 items, got %d", len(arr))
	}
}

// --- Unauthorized tests ---

func TestCreateReport_Unauthorized(t *testing.T) {
	svc := &fakeReportSvc{}
	h := testHandler(svc)
	router := setupNoAuthRouter(h)

	req := newReq("POST", "/api/v1/reports", `{"profile_id": 1}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResp(t, w)
	if resp.Code != response.CodeUnauthorized {
		t.Fatalf("expected code %d, got %d", response.CodeUnauthorized, resp.Code)
	}
}

func TestGetReport_Unauthorized(t *testing.T) {
	svc := &fakeReportSvc{}
	h := testHandler(svc)
	router := setupNoAuthRouter(h)

	req := newReq("GET", "/api/v1/reports/1", "")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResp(t, w)
	if resp.Code != response.CodeUnauthorized {
		t.Fatalf("expected code %d, got %d", response.CodeUnauthorized, resp.Code)
	}
}

func TestListReports_Unauthorized(t *testing.T) {
	svc := &fakeReportSvc{}
	h := testHandler(svc)
	router := setupNoAuthRouter(h)

	req := newReq("GET", "/api/v1/reports", "")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResp(t, w)
	if resp.Code != response.CodeUnauthorized {
		t.Fatalf("expected code %d, got %d", response.CodeUnauthorized, resp.Code)
	}
}
