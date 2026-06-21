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
	exportFn func(ctx context.Context, userID, reportID uint64) (string, error)
	htmlFn   func(ctx context.Context, userID, reportID uint64) (string, error)
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

func (f *fakeReportSvc) ExportReportPDF(ctx context.Context, userID, reportID uint64) (string, error) {
	return f.exportFn(ctx, userID, reportID)
}

func (f *fakeReportSvc) RenderReportHTML(ctx context.Context, userID, reportID uint64) (string, error) {
	return f.htmlFn(ctx, userID, reportID)
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
	r.GET("/api/v1/reports/:id/pdf", h.ExportPDF)
	r.GET("/api/v1/reports/:id/html", h.ViewHTML)

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
				Status:  "done",
				Paid:    true,
				PDFURL:  "https://cdn.example.com/report/1.pdf",
				Content: model.ReportContent{Locale: "en", SummaryLine: "preview line", Summary: "preview text", Chapters: []model.Chapter{{No: 1, Key: "chart_detail", Title: "ch1", Body: "b1"}, {No: 2, Key: "destiny_depth", Title: "ch2", Body: "b2"}, {No: 3, Key: "ten_gods_full", Title: "ch3", Body: "b3"}, {No: 4, Key: "luck_cycle", Title: "ch4", Body: "b4"}, {No: 5, Key: "ten_year_years", Title: "ch5", Body: "b5"}, {No: 6, Key: "career_depth", Title: "ch6", Body: "b6"}, {No: 7, Key: "wealth_depth", Title: "ch7", Body: "b7"}, {No: 8, Key: "love_depth", Title: "ch8", Body: "b8"}, {No: 9, Key: "health_depth", Title: "ch9", Body: "b9"}, {No: 10, Key: "remedies", Title: "ch10", Body: "b10"}, {No: 11, Key: "fortune_guide", Title: "ch11", Body: "b11"}, {No: 12, Key: "life_plan", Title: "ch12", Body: "b12"}}},
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
	if data["paid"] != true {
		t.Errorf("expected paid=true")
	}
	if data["locked"] != false {
		t.Errorf("expected locked=false for paid report")
	}
	if data["pdf_url"] != "https://cdn.example.com/report/1.pdf" {
		t.Errorf("expected pdf_url, got %v", data["pdf_url"])
	}
	if data["content"] == nil {
		t.Error("expected content to be present for paid report")
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

// --- Paid gating tests ---

func TestGetReport_Locked_Unpaid(t *testing.T) {
	svc := &fakeReportSvc{
		getFn: func(ctx context.Context, userID, reportID uint64) (*model.Report, error) {
			return &model.Report{
				ID: reportID, UserID: userID, ProfileID: 1,
				Status: "done",
				Paid:   false,
				PDFURL: "https://cdn.example.com/report/1.pdf",
				Content: model.ReportContent{
					Locale:      "en",
					SummaryLine: "hook line",
					Summary:     "hook summary",
					Chapters:    []model.Chapter{{No: 1, Title: "secret", Body: "secret body"}},
					Personality: "secret",
					Career:      "secret",
					Suggestions: []string{"secret"},
				},
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

	// Must be locked
	if data["locked"] != true {
		t.Errorf("expected locked=true for unpaid report")
	}
	if data["paid"] != false {
		t.Errorf("expected paid=false")
	}

	// Must have summary preview hooks
	if data["summary_line"] != "hook line" {
		t.Errorf("expected summary_line, got %v", data["summary_line"])
	}
	if data["summary"] != "hook summary" {
		t.Errorf("expected summary, got %v", data["summary"])
	}

	// Must NOT leak pdf_url
	if data["pdf_url"] != nil && data["pdf_url"] != "" {
		t.Errorf("expected no pdf_url for locked report, got %v", data["pdf_url"])
	}

	// Must NOT leak full content
	if data["content"] != nil {
		t.Errorf("expected no content for locked report, got %v", data["content"])
	}
}

func TestGetReport_Processing(t *testing.T) {
	svc := &fakeReportSvc{
		getFn: func(ctx context.Context, userID, reportID uint64) (*model.Report, error) {
			return &model.Report{
				ID: reportID, UserID: userID, ProfileID: 1,
				Status: "processing",
				Paid:   false,
				Content: model.ReportContent{
					Locale:    "en",
					Chapters:  nil,
				},
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
	if data["status"] != "processing" {
		t.Errorf("expected status=processing, got %v", data["status"])
	}
	// processing state should not trigger gating
	if data["locked"] != false {
		t.Errorf("expected locked=false for processing report, got locked=%v", data["locked"])
	}
}

// --- buildReportDetail pure function tests ---

func TestBuildReportDetail_Paid(t *testing.T) {
	r := &model.Report{
		ID:      1,
		Status:  "done",
		Paid:    true,
		Locale:  "en",
		PDFURL:  "https://cdn.example.com/r/1.pdf",
		Content: model.ReportContent{
			SummaryLine:   "line",
			Summary:       "sum",
			Personality:   "p",
			Career:        "c",
			Chapters:      []model.Chapter{
				{No: 1, Key: "chart_detail", Title: "t1", Body: "b1"},
				{No: 2, Key: "destiny_depth", Title: "t2", Body: "b2"},
				{No: 3, Key: "ten_gods_full", Title: "t3", Body: "b3"},
				{No: 4, Key: "luck_cycle", Title: "t4", Body: "b4"},
				{No: 5, Key: "ten_year_years", Title: "t5", Body: "b5"},
				{No: 6, Key: "career_depth", Title: "t6", Body: "b6"},
				{No: 7, Key: "wealth_depth", Title: "t7", Body: "b7"},
				{No: 8, Key: "love_depth", Title: "t8", Body: "b8"},
				{No: 9, Key: "health_depth", Title: "t9", Body: "b9"},
				{No: 10, Key: "remedies", Title: "t10", Body: "b10"},
				{No: 11, Key: "fortune_guide", Title: "t11", Body: "b11"},
				{No: 12, Key: "life_plan", Title: "t12", Body: "b12"},
			},
			Suggestions:   []string{"s"},
		},
	}

	resp := buildReportDetail(r, true)
	if resp.Locked {
		t.Error("expected locked=false for paid report")
	}
	if resp.Content == nil {
		t.Fatal("expected content for paid report")
	}
	if resp.PDFURL != "https://cdn.example.com/r/1.pdf" {
		t.Errorf("expected pdf_url, got %s", resp.PDFURL)
	}
	if len(resp.Content.Chapters) != 12 {
		t.Errorf("expected 12 chapters, got %d", len(resp.Content.Chapters))
	}
}

func TestBuildReportDetail_Unpaid(t *testing.T) {
	r := &model.Report{
		ID:      1,
		Status:  "done",
		Paid:    false,
		Locale:  "en",
		PDFURL:  "https://cdn.example.com/r/1.pdf",
		Content: model.ReportContent{
			SummaryLine:   "hook",
			Summary:       "hook summary",
			Personality:   "secret",
			Career:        "secret",
			Chapters:      []model.Chapter{
				{No: 1, Key: "chart_detail", Title: "secret", Body: "s1"},
				{No: 2, Key: "destiny_depth", Title: "secret", Body: "s2"},
				{No: 3, Key: "ten_gods_full", Title: "secret", Body: "s3"},
				{No: 4, Key: "luck_cycle", Title: "secret", Body: "s4"},
				{No: 5, Key: "ten_year_years", Title: "secret", Body: "s5"},
				{No: 6, Key: "career_depth", Title: "secret", Body: "s6"},
				{No: 7, Key: "wealth_depth", Title: "secret", Body: "s7"},
				{No: 8, Key: "love_depth", Title: "secret", Body: "s8"},
				{No: 9, Key: "health_depth", Title: "secret", Body: "s9"},
				{No: 10, Key: "remedies", Title: "secret", Body: "s10"},
				{No: 11, Key: "fortune_guide", Title: "secret", Body: "s11"},
				{No: 12, Key: "life_plan", Title: "secret", Body: "s12"},
			},
			Suggestions:   []string{"secret"},
		},
	}

	resp := buildReportDetail(r, false)
	if !resp.Locked {
		t.Error("expected locked=true for unpaid report")
	}
	if resp.SummaryLine != "hook" {
		t.Errorf("expected summary_line, got %s", resp.SummaryLine)
	}
	if resp.Summary != "hook summary" {
		t.Errorf("expected summary, got %s", resp.Summary)
	}
	if resp.Content != nil {
		t.Error("expected no content for locked report")
	}
	if resp.PDFURL != "" {
		t.Errorf("expected empty pdf_url, got %s", resp.PDFURL)
	}
}

func TestBuildReportDetail_Processing(t *testing.T) {
	r := &model.Report{
		ID:     1,
		Status: "processing",
		Paid:   false,
		Locale: "en",
	}

	resp := buildReportDetail(r, false)
	if resp.Locked {
		t.Error("expected locked=false for non-done report")
	}
	if resp.Content != nil {
		t.Error("expected no content for processing report")
	}
}

func TestBuildReportDetail_AdminBypass(t *testing.T) {
	r := &model.Report{
		ID:      1,
		Status:  "done",
		Paid:    false, // unpaid
		Locale:  "en",
		PDFURL:  "https://cdn.example.com/r/1.pdf",
		Content: model.ReportContent{
			SummaryLine:   "hook",
			Summary:       "hook summary",
			Chapters:      []model.Chapter{
				{No: 1, Key: "chart_detail", Title: "visible", Body: "v1"},
				{No: 2, Key: "destiny_depth", Title: "visible", Body: "v2"},
				{No: 3, Key: "ten_gods_full", Title: "visible", Body: "v3"},
				{No: 4, Key: "luck_cycle", Title: "visible", Body: "v4"},
				{No: 5, Key: "ten_year_years", Title: "visible", Body: "v5"},
				{No: 6, Key: "career_depth", Title: "visible", Body: "v6"},
				{No: 7, Key: "wealth_depth", Title: "visible", Body: "v7"},
				{No: 8, Key: "love_depth", Title: "visible", Body: "v8"},
				{No: 9, Key: "health_depth", Title: "visible", Body: "v9"},
				{No: 10, Key: "remedies", Title: "visible", Body: "v10"},
				{No: 11, Key: "fortune_guide", Title: "visible", Body: "v11"},
				{No: 12, Key: "life_plan", Title: "visible", Body: "v12"},
			},
		},
	}

	// unlocked=true (admin bypass) → should get full content even if unpaid
	resp := buildReportDetail(r, true)
	if resp.Locked {
		t.Error("expected locked=false for admin bypass on unpaid report")
	}
	if resp.Content == nil {
		t.Fatal("expected content for admin bypass")
	}
	if resp.PDFURL != "https://cdn.example.com/r/1.pdf" {
		t.Errorf("expected pdf_url for admin bypass, got %s", resp.PDFURL)
	}
	if resp.Content.Chapters == nil || len(resp.Content.Chapters) != 12 {
		t.Error("expected 12 chapters for admin bypass")
	}
}

// --- PDF / HTML 付费门控测试 ---

func TestExportPDF_Locked_Unpaid_403(t *testing.T) {
	svc := &fakeReportSvc{
		getFn: func(ctx context.Context, userID, reportID uint64) (*model.Report, error) {
			return &model.Report{ID: reportID, UserID: userID, Status: "done", Paid: false}, nil
		},
		exportFn: func(ctx context.Context, userID, reportID uint64) (string, error) {
			t.Fatal("ExportReportPDF should NOT be called when locked")
			return "", nil
		},
	}
	h := testHandler(svc)
	router := setupAuthedRouter(h)

	req := newReq("GET", "/api/v1/reports/1/pdf", "")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResp(t, w)
	if resp.Code != response.CodeForbidden {
		t.Fatalf("expected code %d, got %d", response.CodeForbidden, resp.Code)
	}
}

func TestExportPDF_Paid_OK(t *testing.T) {
	svc := &fakeReportSvc{
		getFn: func(ctx context.Context, userID, reportID uint64) (*model.Report, error) {
			return &model.Report{ID: reportID, UserID: userID, Status: "done", Paid: true}, nil
		},
		exportFn: func(ctx context.Context, userID, reportID uint64) (string, error) {
			return "https://cdn.example.com/report/1.pdf", nil
		},
	}
	h := testHandler(svc)
	router := setupAuthedRouter(h)

	req := newReq("GET", "/api/v1/reports/1/pdf", "")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResp(t, w)
	if resp.Code != response.CodeOK {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestViewHTML_Locked_Unpaid_403(t *testing.T) {
	svc := &fakeReportSvc{
		getFn: func(ctx context.Context, userID, reportID uint64) (*model.Report, error) {
			return &model.Report{ID: reportID, UserID: userID, Status: "done", Paid: false}, nil
		},
		htmlFn: func(ctx context.Context, userID, reportID uint64) (string, error) {
			t.Fatal("RenderReportHTML should NOT be called when locked")
			return "", nil
		},
	}
	h := testHandler(svc)
	router := setupAuthedRouter(h)

	req := newReq("GET", "/api/v1/reports/1/html", "")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := parseResp(t, w)
	if resp.Code != response.CodeForbidden {
		t.Fatalf("expected code %d, got %d", response.CodeForbidden, resp.Code)
	}
}

func TestViewHTML_Paid_OK(t *testing.T) {
	svc := &fakeReportSvc{
		getFn: func(ctx context.Context, userID, reportID uint64) (*model.Report, error) {
			return &model.Report{ID: reportID, UserID: userID, Status: "done", Paid: true}, nil
		},
		htmlFn: func(ctx context.Context, userID, reportID uint64) (string, error) {
			return "<html>full</html>", nil
		},
	}
	h := testHandler(svc)
	router := setupAuthedRouter(h)

	req := newReq("GET", "/api/v1/reports/1/html", "")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected http 200, got %d", w.Code)
	}
}
