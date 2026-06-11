package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"fatelumen/backend/internal/cache"
	"fatelumen/backend/internal/llm"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/renderer"
	"fatelumen/backend/internal/repository"
)

// ---------- fakes ----------

type fakeReadingStore struct {
	mu      sync.Mutex
	items   map[uint64]*model.Reading
	nextID  uint64
	errOn   string // "create" / "update"
}

func newFakeReadingStore() *fakeReadingStore {
	return &fakeReadingStore{items: make(map[uint64]*model.Reading)}
}

func (f *fakeReadingStore) Create(r *model.Reading) error {
	if f.errOn == "create" {
		return errors.New("fake create error")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	r.ID = f.nextID
	copy := *r
	f.items[r.ID] = &copy
	return nil
}

func (f *fakeReadingStore) GetByID(id, userID uint64) (*model.Reading, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.items[id]
	if !ok || r.UserID != userID {
		return nil, fmt.Errorf("not found")
	}
	copy := *r
	return &copy, nil
}

func (f *fakeReadingStore) ListByUser(userID uint64, limit, offset int) ([]model.Reading, error) {
	return nil, nil
}

func (f *fakeReadingStore) UpdateImageURL(id uint64, imageURL string) error {
	if f.errOn == "update" {
		return errors.New("fake update error")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.items[id]; ok {
		r.ImageURL = imageURL
	}
	return nil
}

type fakeProfileGetter struct {
	profile *model.BirthProfile
	err     error
}

func (f *fakeProfileGetter) FindByID(id uint64) (*model.BirthProfile, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.profile, nil
}

func newFakeProfile() *model.BirthProfile {
	return &model.BirthProfile{
		ID:           1,
		UserID:       42,
		Gender:       1,
		CalendarType: 0,
		BirthYear:    1990,
		BirthMonth:   8,
		BirthDay:     15,
		BirthHour:    14,
		BirthMinute:  30,
	}
}

type fakeChartProvider struct {
	chart *model.Chart
	err   error
}

func (f *fakeChartProvider) Calculate(ctx context.Context, userID uint64, in CreateChartInput) (*model.Chart, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.chart, nil
}

func minChart() *model.Chart {
	return &model.Chart{
		ID:        1,
		ProfileID: 1,
		ChartData: model.ChartData{
			DayMaster: model.DayMaster{
				Stem:    "甲",
				Element: "木",
				YinYang: "阳",
			},
			Meta: model.ChartMeta{
				SolarDate: "1990-08-15",
			},
		},
	}
}

type fakeLLM struct {
	result string
	err    error
	name   string
}

func (f *fakeLLM) GenerateJSON(ctx context.Context, system, user string, opts ...llm.Option) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.result, nil
}

func (f *fakeLLM) Name() string {
	if f.name == "" {
		return "fake-llm"
	}
	return f.name
}

func okLLMResult() string {
	c := model.QuickContent{
		SummaryLine: "A bright Wood rising at dawn.",
		Personality: "Adaptable and kind.",
		Strengths:   []string{"Creative", "Resilient"},
		Weaknesses:  []string{"Indecisive"},
		ElementNote: "Wood nourishes Fire.",
	}
	b, _ := json.Marshal(c)
	return string(b)
}

func badJSONLLMResult() string {
	return `not valid json`
}

type fakeRenderer struct {
	png []byte
	err error
}

func (f *fakeRenderer) Render(ctx context.Context, html string, format renderer.Format) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.png, nil
}

type fakeStorage struct {
	url string
	err error
}

func (f *fakeStorage) Put(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.url, nil
}

// ---------- tests ----------

func fakeQuotaService(limit int) *QuotaService {
	mc := cache.NewMemoryCache()
	defer mc.Close()
	// Don't close here — keep it alive for the test
	return NewQuotaService(cache.NewMemoryCache(), limit)
}

func TestCreateQuick_Success(t *testing.T) {
	mc := cache.NewMemoryCache()
	quota := NewQuotaService(mc, 3)

	svc := &ReadingService{
		readingRepo:  newFakeReadingStore(),
		profileRepo:  &fakeProfileGetter{profile: newFakeProfile()},
		chartService: &fakeChartProvider{chart: minChart()},
		quotaService: quota,
		llmProvider:  &fakeLLM{result: okLLMResult()},
		imgRenderer:  &fakeRenderer{png: []byte("fake-png-data")},
		fileStorage:  &fakeStorage{url: "https://cdn.example.com/reading.png"},
	}

	reading, err := svc.CreateQuick(context.Background(), 42, CreateQuickInput{
		ProfileID: 1,
		Locale:    "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reading == nil {
		t.Fatal("expected reading, got nil")
	}
	if reading.UserID != 42 {
		t.Fatalf("expected userID 42, got %d", reading.UserID)
	}
	if reading.Content.SummaryLine == "" {
		t.Fatal("expected summary_line, got empty")
	}
	if reading.ImageURL != "https://cdn.example.com/reading.png" {
		t.Fatalf("expected image URL, got '%s'", reading.ImageURL)
	}
	if reading.Status != "done" {
		t.Fatalf("expected status 'done', got '%s'", reading.Status)
	}
	if reading.ChartID != 1 {
		t.Fatalf("expected chartID 1, got %d", reading.ChartID)
	}
}

func TestCreateQuick_QuotaExceeded(t *testing.T) {
	mc := cache.NewMemoryCache()
	quota := NewQuotaService(mc, 1)

	svc := &ReadingService{
		readingRepo:  newFakeReadingStore(),
		profileRepo:  &fakeProfileGetter{profile: newFakeProfile()},
		chartService: &fakeChartProvider{chart: minChart()},
		quotaService: quota,
		llmProvider:  &fakeLLM{result: okLLMResult()},
		imgRenderer:  &fakeRenderer{png: []byte("fake-png-data")},
		fileStorage:  &fakeStorage{url: "https://cdn.example.com/reading.png"},
	}

	ctx := context.Background()

	// First call succeeds
	_, err := svc.CreateQuick(ctx, 99, CreateQuickInput{ProfileID: 1})
	if err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}

	// Second call exceeds quota
	_, err = svc.CreateQuick(ctx, 99, CreateQuickInput{ProfileID: 1})
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestCreateQuick_AdminBypassesQuota(t *testing.T) {
	mc := cache.NewMemoryCache()
	quota := NewQuotaService(mc, 1)

	svc := &ReadingService{
		readingRepo:  newFakeReadingStore(),
		profileRepo:  &fakeProfileGetter{profile: newFakeProfile()},
		chartService: &fakeChartProvider{chart: minChart()},
		quotaService: quota,
		llmProvider:  &fakeLLM{result: okLLMResult()},
		imgRenderer:  &fakeRenderer{png: []byte("fake-png-data")},
		fileStorage:  &fakeStorage{url: "https://cdn.example.com/reading.png"},
	}

	ctx := context.Background()

	// First call as admin (bypasses quota)
	_, err := svc.CreateQuick(ctx, 99, CreateQuickInput{ProfileID: 1, IsAdmin: true})
	if err != nil {
		t.Fatalf("admin first call: unexpected error: %v", err)
	}

	// Second call as admin still works (no quota deducted)
	_, err = svc.CreateQuick(ctx, 99, CreateQuickInput{ProfileID: 1, IsAdmin: true})
	if err != nil {
		t.Fatalf("admin second call: unexpected error: %v", err)
	}

	// Third call as normal user should still have quota=1 available
	_, err = svc.CreateQuick(ctx, 99, CreateQuickInput{ProfileID: 1, IsAdmin: false})
	if err != nil {
		t.Fatalf("normal user first call after admin: unexpected error: %v", err)
	}

	// Normal user second call exceeds quota
	_, err = svc.CreateQuick(ctx, 99, CreateQuickInput{ProfileID: 1, IsAdmin: false})
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("expected ErrQuotaExceeded for normal user, got %v", err)
	}
}

func TestCreateQuick_LLMJSONParseFailure(t *testing.T) {
	mc := cache.NewMemoryCache()
	quota := NewQuotaService(mc, 3)

	svc := &ReadingService{
		readingRepo:  newFakeReadingStore(),
		profileRepo:  &fakeProfileGetter{profile: newFakeProfile()},
		chartService: &fakeChartProvider{chart: minChart()},
		quotaService: quota,
		llmProvider:  &fakeLLM{result: badJSONLLMResult()},
		imgRenderer:  &fakeRenderer{png: []byte("fake-png-data")},
		fileStorage:  &fakeStorage{url: "https://cdn.example.com/reading.png"},
	}

	_, err := svc.CreateQuick(context.Background(), 42, CreateQuickInput{ProfileID: 1})
	if err == nil {
		t.Fatal("expected error for bad JSON, got nil")
	}
}

func TestCreateQuick_LLMCallFailure(t *testing.T) {
	mc := cache.NewMemoryCache()
	quota := NewQuotaService(mc, 3)

	svc := &ReadingService{
		readingRepo:  newFakeReadingStore(),
		profileRepo:  &fakeProfileGetter{profile: newFakeProfile()},
		chartService: &fakeChartProvider{chart: minChart()},
		quotaService: quota,
		llmProvider:  &fakeLLM{err: errors.New("llm timeout")},
		imgRenderer:  &fakeRenderer{png: []byte("fake-png-data")},
		fileStorage:  &fakeStorage{url: "https://cdn.example.com/reading.png"},
	}

	_, err := svc.CreateQuick(context.Background(), 42, CreateQuickInput{ProfileID: 1})
	if err == nil {
		t.Fatal("expected error for LLM failure, got nil")
	}
}

func TestCreateQuick_StorageFailure(t *testing.T) {
	mc := cache.NewMemoryCache()
	quota := NewQuotaService(mc, 3)

	svc := &ReadingService{
		readingRepo:  newFakeReadingStore(),
		profileRepo:  &fakeProfileGetter{profile: newFakeProfile()},
		chartService: &fakeChartProvider{chart: minChart()},
		quotaService: quota,
		llmProvider:  &fakeLLM{result: okLLMResult()},
		imgRenderer:  &fakeRenderer{png: []byte("fake-png-data")},
		fileStorage:  &fakeStorage{err: errors.New("r2 upload failed")},
	}

	_, err := svc.CreateQuick(context.Background(), 42, CreateQuickInput{ProfileID: 1})
	if err == nil {
		t.Fatal("expected error for storage failure, got nil")
	}
}

func TestCreateQuick_RenderFailure(t *testing.T) {
	mc := cache.NewMemoryCache()
	quota := NewQuotaService(mc, 3)

	svc := &ReadingService{
		readingRepo:  newFakeReadingStore(),
		profileRepo:  &fakeProfileGetter{profile: newFakeProfile()},
		chartService: &fakeChartProvider{chart: minChart()},
		quotaService: quota,
		llmProvider:  &fakeLLM{result: okLLMResult()},
		imgRenderer:  &fakeRenderer{err: errors.New("chromedp failed")},
		fileStorage:  &fakeStorage{url: "https://cdn.example.com/reading.png"},
	}

	_, err := svc.CreateQuick(context.Background(), 42, CreateQuickInput{ProfileID: 1})
	if err == nil {
		t.Fatal("expected error for render failure, got nil")
	}
}

func TestCreateQuick_ChartFailure(t *testing.T) {
	mc := cache.NewMemoryCache()
	quota := NewQuotaService(mc, 3)

	svc := &ReadingService{
		readingRepo:  newFakeReadingStore(),
		profileRepo:  &fakeProfileGetter{profile: newFakeProfile()},
		chartService: &fakeChartProvider{err: errors.New("profile not found")},
		quotaService: quota,
		llmProvider:  &fakeLLM{result: okLLMResult()},
		imgRenderer:  &fakeRenderer{png: []byte("fake-png-data")},
		fileStorage:  &fakeStorage{url: "https://cdn.example.com/reading.png"},
	}

	_, err := svc.CreateQuick(context.Background(), 42, CreateQuickInput{ProfileID: 1})
	if err == nil {
		t.Fatal("expected error for chart failure, got nil")
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc := &ReadingService{
		readingRepo: newFakeReadingStore(),
	}
	_, err := svc.GetByID(context.Background(), 1, 999)
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestGetByID_WrongUser(t *testing.T) {
	store := newFakeReadingStore()
	store.Create(&model.Reading{UserID: 1})
	store.Create(&model.Reading{UserID: 2})

	svc := &ReadingService{
		readingRepo: store,
	}

	// User 1 can access their own
	r, err := svc.GetByID(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("user 1 should access record 1: %v", err)
	}
	if r.UserID != 1 {
		t.Fatalf("expected userID 1, got %d", r.UserID)
	}

	// User 3 cannot access user 2's record
	_, err = svc.GetByID(context.Background(), 3, 2)
	if err == nil {
		t.Fatal("user 3 should not access user 2's record")
	}
}

// Ensure concrete types satisfy interfaces (compile-time check)
var _ readingStore = (*repository.ReadingRepo)(nil)
var _ profileGetter = (*repository.ProfileRepo)(nil)
var _ chartProvider = (*ChartService)(nil)
var _ quotaChecker = (*QuotaService)(nil)
