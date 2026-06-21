package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"fatelumen/backend/internal/job"
	"fatelumen/backend/internal/model"
)

// ---------- fakes for report service tests ----------

type fakeReportStore struct {
	mu       sync.Mutex
	reports  map[uint64]*model.Report
	nextID   uint64
	errOn    string // "create"
	createFn func(*model.Report)
}

func newFakeReportStore() *fakeReportStore {
	return &fakeReportStore{reports: make(map[uint64]*model.Report)}
}

func (f *fakeReportStore) Create(r *model.Report) error {
	if f.errOn == "create" {
		return errors.New("fake create error")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	r.ID = f.nextID
	copy := *r
	f.reports[r.ID] = &copy
	if f.createFn != nil {
		f.createFn(r)
	}
	return nil
}

func (f *fakeReportStore) GetByID(id, userID uint64) (*model.Report, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.reports[id]
	if !ok || r.UserID != userID {
		return nil, errors.New("not found")
	}
	copy := *r
	return &copy, nil
}

func (f *fakeReportStore) ListByUser(userID uint64, limit, offset int) ([]model.Report, error) {
	return nil, nil
}

func (f *fakeReportStore) UpdateStatus(id uint64, status, errMsg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.reports[id]; ok {
		r.Status = status
		r.ErrorMsg = errMsg
	}
	return nil
}

func (f *fakeReportStore) UpdateResult(id uint64, content model.ReportContent, pdfURL string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.reports[id]; ok {
		r.Content = content
		r.PDFURL = pdfURL
	}
	return nil
}

func (f *fakeReportStore) UnlockReportWithCredits(userID, reportID uint64, cost int) error {
	return nil
}

type fakeReportQueue struct {
	mu   sync.Mutex
	jobs []*job.Job
	err  error
}

func newFakeReportQueue() *fakeReportQueue {
	return &fakeReportQueue{}
}

func (q *fakeReportQueue) Enqueue(ctx context.Context, j *job.Job) error {
	if q.err != nil {
		return q.err
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	copy := *j
	q.jobs = append(q.jobs, &copy)
	return nil
}

func (q *fakeReportQueue) lastJob() *job.Job {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.jobs) == 0 {
		return nil
	}
	return q.jobs[len(q.jobs)-1]
}

// ---------- report service tests ----------

func TestCreateReport_Success(t *testing.T) {
	store := newFakeReportStore()
	queue := newFakeReportQueue()
	svc := &ReportService{
		reportRepo: store,
		queue:      queue,
	}

	report, err := svc.CreateReport(context.Background(), 42, 1, "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report == nil {
		t.Fatal("expected report, got nil")
	}
	if report.ID == 0 {
		t.Fatal("expected report ID > 0")
	}
	if report.UserID != 42 {
		t.Fatalf("expected userID 42, got %d", report.UserID)
	}
	if report.ProfileID != 1 {
		t.Fatalf("expected profileID 1, got %d", report.ProfileID)
	}
	if report.Status != "pending" {
		t.Fatalf("expected status 'pending', got '%s'", report.Status)
	}

	// 验证队列收到 job
	lastJob := queue.lastJob()
	if lastJob == nil {
		t.Fatal("expected a job in the queue")
	}
	if lastJob.Type != "report" {
		t.Fatalf("expected job type 'report', got '%s'", lastJob.Type)
	}

	// 验证 payload 含 report_id
	var payload reportPayload
	if err := json.Unmarshal([]byte(lastJob.Payload), &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if payload.ReportID != report.ID {
		t.Fatalf("expected report_id %d in payload, got %d", report.ID, payload.ReportID)
	}
	if payload.UserID != 42 {
		t.Fatalf("expected user_id 42, got %d", payload.UserID)
	}
}

func TestCreateReport_DefaultLocale(t *testing.T) {
	store := newFakeReportStore()
	queue := newFakeReportQueue()
	svc := &ReportService{
		reportRepo: store,
		queue:      queue,
	}

	report, err := svc.CreateReport(context.Background(), 42, 1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Locale != "en" {
		t.Fatalf("expected default locale 'en', got '%s'", report.Locale)
	}
}

func TestCreateReport_CreateFails(t *testing.T) {
	store := newFakeReportStore()
	store.errOn = "create"
	svc := &ReportService{
		reportRepo: store,
		queue:      newFakeReportQueue(),
	}

	_, err := svc.CreateReport(context.Background(), 42, 1, "en")
	if err == nil {
		t.Fatal("expected error for create failure")
	}
}

func TestCreateReport_EnqueueFails(t *testing.T) {
	store := newFakeReportStore()
	queue := newFakeReportQueue()
	queue.err = errors.New("queue full")
	svc := &ReportService{
		reportRepo: store,
		queue:      queue,
	}

	_, err := svc.CreateReport(context.Background(), 42, 1, "en")
	if err == nil {
		t.Fatal("expected error for enqueue failure")
	}
}

func TestGetReport_Success(t *testing.T) {
	store := newFakeReportStore()
	store.Create(&model.Report{UserID: 42, Status: "pending"})
	store.Create(&model.Report{UserID: 99, Status: "pending"})

	svc := &ReportService{reportRepo: store}

	// User 42 can access their own report
	r, err := svc.GetReport(context.Background(), 42, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.UserID != 42 {
		t.Fatalf("expected userID 42, got %d", r.UserID)
	}

	// User 42 cannot access user 99's report
	_, err = svc.GetReport(context.Background(), 42, 2)
	if err == nil {
		t.Fatal("expected error for wrong user access")
	}
}
