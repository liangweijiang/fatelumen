package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"fatelumen/backend/internal/job"
	"fatelumen/backend/internal/llm"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/renderer"
)

// ---------- fakes for report handler tests ----------

type fakeReportGetter struct {
	mu      sync.Mutex
	reports map[uint64]*model.Report
	errOn   string
}

func newFakeReportGetter() *fakeReportGetter {
	return &fakeReportGetter{reports: make(map[uint64]*model.Report)}
}

func (f *fakeReportGetter) addReport(r *model.Report) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reports[r.ID] = r
}

func (f *fakeReportGetter) getReport(id uint64) *model.Report {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reports[id]
}

func (f *fakeReportGetter) GetByID(id, userID uint64) (*model.Report, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.reports[id]
	if !ok || r.UserID != userID {
		return nil, errors.New("not found")
	}
	return r, nil
}

func (f *fakeReportGetter) UpdateStatus(id uint64, status, errMsg string) error {
	if f.errOn == "update_status" {
		return errors.New("fake update status error")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.reports[id]; ok {
		r.Status = status
		r.ErrorMsg = errMsg
	}
	return nil
}

func (f *fakeReportGetter) UpdateResult(id uint64, content model.ReportContent, pdfURL string) error {
	if f.errOn == "update_result" {
		return errors.New("fake update result error")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.reports[id]; ok {
		r.Content = content
		r.PDFURL = pdfURL
	}
	return nil
}

func (f *fakeReportGetter) UpdateChartID(id uint64, chartID uint64) error {
	if f.errOn == "update_chart_id" {
		return errors.New("fake update chart id error")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.reports[id]; ok {
		r.ChartID = chartID
	}
	return nil
}

type fakeChartSaver struct {
	mu     sync.Mutex
	charts map[string]*model.Chart // keyed by hash
	nextID uint64
}

func newFakeChartSaver() *fakeChartSaver {
	return &fakeChartSaver{charts: make(map[string]*model.Chart)}
}

func (f *fakeChartSaver) FindByHash(hash string) (*model.Chart, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.charts[hash]
	if !ok {
		return nil, errors.New("not found")
	}
	return c, nil
}

func (f *fakeChartSaver) Create(chart *model.Chart) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	chart.ID = f.nextID
	copy := *chart
	f.charts[chart.ChartHash] = &copy
	return nil
}

func fakeReportContentJSON() string {
	c := model.ReportContent{
		Locale:      "en",
		SummaryLine: "A strong Wood pillar rising at dawn.",
		Summary:     "Comprehensive overview of this chart's destiny configuration.",
		Personality: "The Day Master 甲 Wood represents growth, compassion, and leadership.",
		Career:      "Favorable elements support career in creative and service-oriented fields.",
		Relationship: "Peach blossom indicators suggest warm romantic prospects.",
		Health:      "Elemental balance points to strong constitution; stay hydrated.",
		YearlyFortune: []model.YearlyFortuneItem{
			{Year: 2026, Note: "A year of steady progress and recognition."},
			{Year: 2027, Note: "Career breakthrough likely with proper planning."},
			{Year: 2028, Note: "Nurture close relationships for emotional support."},
		},
		Suggestions: []string{
			"Pursue creative projects with confidence",
			"Build a consistent health routine",
			"Cultivate patience in relationships",
			"Explore educational opportunities",
		},
		Chapters: []model.Chapter{
			{No: 1, Key: "chart_detail", Title: "Refined Chart Reading", Body: "chapter 1 body..."},
			{No: 2, Key: "destiny_depth", Title: "In-Depth Destiny Reading", Body: "chapter 2 body..."},
			{No: 3, Key: "ten_gods_full", Title: "Full Ten-Gods Panorama", Body: "chapter 3 body..."},
			{No: 4, Key: "luck_cycle", Title: "Lifelong Luck-Cycle Trend", Body: "chapter 4 body..."},
			{No: 5, Key: "ten_year_years", Title: "Next Ten Years", Body: "chapter 5 body..."},
			{No: 6, Key: "career_depth", Title: "Career Deep Dive", Body: "chapter 6 body..."},
			{No: 7, Key: "wealth_depth", Title: "Wealth Deep Dive", Body: "chapter 7 body..."},
			{No: 8, Key: "love_depth", Title: "Relationship Deep Dive", Body: "chapter 8 body..."},
			{No: 9, Key: "health_depth", Title: "Health Deep Dive", Body: "chapter 9 body..."},
			{No: 10, Key: "remedies", Title: "Remedies", Body: "chapter 10 body..."},
			{No: 11, Key: "fortune_guide", Title: "Fortune Guide", Body: "chapter 11 body..."},
			{No: 12, Key: "life_plan", Title: "Lifelong Guidance", Body: "chapter 12 body..."},
		},
	}
	b, _ := json.Marshal(c)
	return string(b)
}

func buildReportJob(reportID, userID, profileID uint64, locale string) *job.Job {
	payloadBytes, _ := json.Marshal(reportPayload{
		ReportID:  reportID,
		UserID:    userID,
		ProfileID: profileID,
		Locale:    locale,
	})
	return &job.Job{
		ID:      job.NewJobID(),
		Type:    "report",
		Status:  job.StatusPending,
		Payload: payloadBytes,
	}
}

func newHandlerWithFakes() (*reportHandler, *fakeProfileGetter, *fakeChartSaver, *fakeReportGetter, *fakeLLM, *fakeRenderer, *fakeStorage) {
	profileGetter := &fakeProfileGetter{profile: newFakeProfile()}
	chartSaver := newFakeChartSaver()
	reportGetter := newFakeReportGetter()
	llmProvider := &fakeLLM{result: fakeReportContentJSON()}
	renderer := &fakeRenderer{png: []byte("fake-pdf-bytes")}
	storage := &fakeStorage{url: "https://cdn.example.com/reports/42/1.pdf"}

	// Inject preregistered report
	reportGetter.addReport(&model.Report{
		ID:        1,
		UserID:    42,
		ProfileID: 1,
		Status:    "pending",
	})

	h := &reportHandler{
		profileRepo: profileGetter,
		chartRepo:   chartSaver,
		llmProvider: llmProvider,
		renderer:    renderer,
		storage:     storage,
		reportRepo:  reportGetter,
	}
	return h, profileGetter, chartSaver, reportGetter, llmProvider, renderer, storage
}

// ---------- handler tests ----------

func TestReportHandler_Success(t *testing.T) {
	h, _, chartSaver, reportGetter, _, _, _ := newHandlerWithFakes()

	job := buildReportJob(1, 42, 1, "en")
	_, err := h.Handle(context.Background(), job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 验证 report 状态为 done
	r := reportGetter.getReport(1)
	if r == nil {
		t.Fatal("report not found")
	}
	if r.Status != "done" {
		t.Fatalf("expected status 'done', got '%s'", r.Status)
	}
	if r.Content.SummaryLine == "" {
		t.Fatal("expected content with summary_line")
	}
	if len(r.Content.Chapters) != 12 {
		t.Fatalf("expected 12 chapters in content, got %d", len(r.Content.Chapters))
	}
	for i, ch := range r.Content.Chapters {
		if ch.No != i+1 {
			t.Errorf("chapter %d: expected No=%d, got %d", i, i+1, ch.No)
		}
		if ch.Key == "" {
			t.Errorf("chapter %d: Key is empty", i)
		}
		if ch.Body == "" {
			t.Errorf("chapter %d: Body is empty", i)
		}
	}
	if r.ChartID == 0 {
		t.Fatal("expected chart_id > 0")
	}

	// 验证 chart 已落库
	if len(chartSaver.charts) == 0 {
		t.Fatal("expected chart to be saved")
	}
}

func TestReportHandler_InvalidPayload(t *testing.T) {
	h, _, _, _, _, _, _ := newHandlerWithFakes()

	job := &job.Job{
		ID:      job.NewJobID(),
		Type:    "report",
		Payload: json.RawMessage(`not json`),
	}

	_, err := h.Handle(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for invalid payload")
	}
}

func TestReportHandler_ProfileNotFound(t *testing.T) {
	h, profileGetter, _, _, _, _, _ := newHandlerWithFakes()
	profileGetter.err = errors.New("profile not found")

	job := buildReportJob(1, 42, 999, "en")
	_, err := h.Handle(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for profile not found")
	}
}

func TestReportHandler_LLMJSONParseFailure(t *testing.T) {
	h, _, _, _, llmProvider, _, _ := newHandlerWithFakes()

	// Set LLM to return invalid JSON
	llmProvider.result = `not valid json`

	job := buildReportJob(1, 42, 1, "en")
	_, err := h.Handle(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for LLM JSON parse failure")
	}
}

func TestReportHandler_LLMCallFailure(t *testing.T) {
	h, _, _, _, llmProvider, _, _ := newHandlerWithFakes()
	llmProvider.err = errors.New("llm timeout")

	job := buildReportJob(1, 42, 1, "en")
	_, err := h.Handle(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for LLM call failure")
	}
}

func TestReportHandler_ChartReuse(t *testing.T) {
	h, _, chartSaver, _, _, _, _ := newHandlerWithFakes()

	// First run creates a chart
	job1 := buildReportJob(1, 42, 1, "en")
	_, err := h.Handle(context.Background(), job1)
	if err != nil {
		t.Fatalf("first run: unexpected error: %v", err)
	}

	chartCount := len(chartSaver.charts)

	// Second run should reuse the same chart (same profile)
	reportGetter := newFakeReportGetter()
	reportGetter.addReport(&model.Report{ID: 2, UserID: 42, ProfileID: 1, Status: "pending"})
	h.reportRepo = reportGetter

	job2 := buildReportJob(2, 42, 1, "en")
	_, err = h.Handle(context.Background(), job2)
	if err != nil {
		t.Fatalf("second run: unexpected error: %v", err)
	}

	if len(chartSaver.charts) != chartCount {
		t.Fatalf("expected chart count %d (reused), got %d", chartCount, len(chartSaver.charts))
	}
}

// Ensure concrete types satisfy interfaces (compile-time check)
var _ reportStore = (*fakeReportStore)(nil)
var _ reportQueue = (*fakeReportQueue)(nil)
var _ reportGetter = (*fakeReportGetter)(nil)
var _ chartSaver = (*fakeChartSaver)(nil)
var _ llm.LLMProvider = (*fakeLLM)(nil)
var _ renderer.Renderer = (*fakeRenderer)(nil)
var _ job.JobHandler = (*reportHandler)(nil)
