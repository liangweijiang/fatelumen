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

type fakeAdminReportStore struct {
	reports     []model.Report
	total       int64
	listErr     error
	report      *model.Report
	getErr      error
	filter      repository.ReportFilter
	limit       int
	offset      int
	markPaidCalls   int
	markPaidErr     error
	lastReportID    uint64
	lastOrderID     uint64
}

func newFakeAdminReportStore() *fakeAdminReportStore {
	return &fakeAdminReportStore{}
}

func (f *fakeAdminReportStore) AdminListReports(filter repository.ReportFilter, limit, offset int) ([]model.Report, int64, error) {
	f.filter = filter
	f.limit = limit
	f.offset = offset
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	return f.reports, f.total, nil
}

func (f *fakeAdminReportStore) AdminGetReportByID(id uint64) (*model.Report, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.report, nil
}

func (f *fakeAdminReportStore) MarkPaid(reportID, orderID uint64) error {
	f.markPaidCalls++
	f.lastReportID = reportID
	f.lastOrderID = orderID
	if f.markPaidErr != nil {
		return f.markPaidErr
	}
	if f.report != nil {
		f.report.Paid = true
	}
	return nil
}

// ---------- Pagination ----------

func TestAdminListReports_Pagination(t *testing.T) {
	store := newFakeAdminReportStore()
	store.reports = []model.Report{
		{ID: 1, UserID: 10, ProfileID: 100, Status: "done", Paid: true},
	}
	store.total = 30

	svc := &AdminReportService{reportRepo: store}

	page, err := svc.ListReports(context.Background(), "", nil, 0, 2, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Total != 30 {
		t.Errorf("total: want 30, got %d", page.Total)
	}
	if page.Page != 2 {
		t.Errorf("page: want 2, got %d", page.Page)
	}
	if page.PageSize != 20 {
		t.Errorf("pageSize: want 20, got %d", page.PageSize)
	}
}

func TestAdminListReports_OffsetCalculation(t *testing.T) {
	store := newFakeAdminReportStore()
	svc := &AdminReportService{reportRepo: store}

	svc.ListReports(context.Background(), "", nil, 0, 3, 20)
	if store.offset != 40 {
		t.Errorf("offset: want 40, got %d", store.offset)
	}
}

func TestAdminListReports_PageSizeCap(t *testing.T) {
	store := newFakeAdminReportStore()
	svc := &AdminReportService{reportRepo: store}

	page, err := svc.ListReports(context.Background(), "", nil, 0, 1, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.PageSize != 20 {
		t.Errorf("expected capped to 20, got %d", page.PageSize)
	}
}

func TestAdminListReports_PaidFilterPassthrough(t *testing.T) {
	store := newFakeAdminReportStore()
	svc := &AdminReportService{reportRepo: store}

	paidTrue := true
	svc.ListReports(context.Background(), "done", &paidTrue, 42, 1, 10)

	if store.filter.Status != "done" {
		t.Errorf("filter status: want 'done', got '%s'", store.filter.Status)
	}
	if store.filter.Paid == nil || *store.filter.Paid != true {
		t.Errorf("filter paid: want true, got %v", store.filter.Paid)
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

func TestAdminListReports_PaidFilterNil(t *testing.T) {
	store := newFakeAdminReportStore()
	svc := &AdminReportService{reportRepo: store}

	svc.ListReports(context.Background(), "", nil, 0, 1, 10)
	if store.filter.Paid != nil {
		t.Errorf("filter paid: want nil when not set, got %v", *store.filter.Paid)
	}
}

// ---------- GetReportDetail ----------

func TestAdminGetReportDetail_Success(t *testing.T) {
	store := newFakeAdminReportStore()
	store.report = &model.Report{
		ID:        1,
		UserID:    10,
		ProfileID: 100,
		PayMethod: "order",
		Status:    "done",
		Paid:      true,
		Content: model.ReportContent{
			SummaryLine: "test summary line",
			Summary:     "test summary body",
		},
		PDFURL: "/reports/1.pdf",
	}

	svc := &AdminReportService{reportRepo: store}
	detail, err := svc.GetReportDetail(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.SummaryLine != "test summary line" {
		t.Errorf("summary_line: want 'test summary line', got '%s'", detail.SummaryLine)
	}
	if detail.Summary != "test summary body" {
		t.Errorf("summary: want 'test summary body', got '%s'", detail.Summary)
	}
	if detail.PDFURL != "/reports/1.pdf" {
		t.Errorf("pdf_url: want '/reports/1.pdf', got '%s'", detail.PDFURL)
	}
	if detail.Type != "order" {
		t.Errorf("type: want 'order', got '%s'", detail.Type)
	}
}

func TestAdminGetReportDetail_NotFound(t *testing.T) {
	store := newFakeAdminReportStore()
	store.getErr = errors.New("not found")
	svc := &AdminReportService{reportRepo: store}

	_, err := svc.GetReportDetail(context.Background(), 99)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

// ---------- UnlockReport ----------

func TestUnlockReport_Success(t *testing.T) {
	store := newFakeAdminReportStore()
	store.report = &model.Report{
		ID:        1,
		Paid:      false,
		PayMethod: "order",
	}

	svc := &AdminReportService{reportRepo: store}
	err := svc.UnlockReport(context.Background(), 1, 1, "customer service requested")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.markPaidCalls != 1 {
		t.Errorf("markPaidCalls: want 1, got %d", store.markPaidCalls)
	}
	if store.lastOrderID != 0 {
		t.Errorf("orderID passed to MarkPaid: want 0 (no order touched), got %d", store.lastOrderID)
	}
	if !store.report.Paid {
		t.Error("report.Paid should be true after unlock")
	}
}

func TestUnlockReport_EmptyReason(t *testing.T) {
	store := newFakeAdminReportStore()

	svc := &AdminReportService{reportRepo: store}
	err := svc.UnlockReport(context.Background(), 1, 1, "  ")
	if err == nil {
		t.Fatal("expected error for empty reason")
	}
	if err.Error() != "reason required" {
		t.Errorf("error: want 'reason required', got '%s'", err.Error())
	}
	if store.markPaidCalls != 0 {
		t.Error("MarkPaid should not be called when reason is empty")
	}
}

func TestUnlockReport_Idempotent(t *testing.T) {
	store := newFakeAdminReportStore()
	store.report = &model.Report{
		ID:        1,
		Paid:      true,
		PayMethod: "credit",
	}

	svc := &AdminReportService{reportRepo: store}
	err := svc.UnlockReport(context.Background(), 1, 1, "already unlocked")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.markPaidCalls != 0 {
		t.Error("MarkPaid should not be called when already paid")
	}
}

func TestUnlockReport_NotFound(t *testing.T) {
	store := newFakeAdminReportStore()
	store.getErr = errors.New("not found")

	svc := &AdminReportService{reportRepo: store}
	err := svc.UnlockReport(context.Background(), 1, 99, "not found reason")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

// ---------- DTO ----------

func TestAdminReportItem_NoContentField(t *testing.T) {
	item := AdminReportItem{
		ID:        1,
		UserID:    10,
		ProfileID: 100,
		Type:      "order",
		Status:    "done",
		Paid:      true,
	}

	b, _ := json.Marshal(item)
	s := string(b)

	if strings.Contains(s, `"content"`) || strings.Contains(s, `"Content"`) {
		t.Errorf("AdminReportItem JSON should not contain content field, got: %s", s)
	}
	if strings.Contains(s, `"pdf_url"`) || strings.Contains(s, `"pdfURL"`) || strings.Contains(s, `"PdfUrl"`) {
		t.Errorf("AdminReportItem JSON should not contain pdf_url field, got: %s", s)
	}
	// Verify expected fields are present
	if !strings.Contains(s, `"id":1`) {
		t.Error("missing id field")
	}
	if !strings.Contains(s, `"type":"order"`) {
		t.Error("missing type field")
	}
}

func TestAdminReportDetail_HasNoChapters(t *testing.T) {
	detail := AdminReportDetail{
		ID:          1,
		SummaryLine: "line",
		Summary:     "body",
	}

	b, _ := json.Marshal(detail)
	s := string(b)

	if strings.Contains(s, `"chapters"`) || strings.Contains(s, `"Chapters"`) {
		t.Errorf("AdminReportDetail JSON should not contain chapters field, got: %s", s)
	}
	if strings.Contains(s, `"content"`) || strings.Contains(s, `"Content"`) {
		t.Errorf("AdminReportDetail JSON should not contain content field, got: %s", s)
	}
}
