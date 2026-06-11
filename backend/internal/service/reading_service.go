package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"fatelumen/backend/internal/llm"
	"fatelumen/backend/internal/llm/prompts"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/renderer"
	"fatelumen/backend/internal/repository"
	"fatelumen/backend/internal/storage"
)

// 私有接口：ReadingService 内部仅依赖接口，便于单测 fake
type readingStore interface {
	Create(*model.Reading) error
	GetByID(uint64, uint64) (*model.Reading, error)
	ListByUser(uint64, int, int) ([]model.Reading, error)
	UpdateImageURL(uint64, string) error
}

type profileGetter interface {
	FindByID(uint64) (*model.BirthProfile, error)
}

type chartProvider interface {
	Calculate(ctx context.Context, userID uint64, in CreateChartInput) (*model.Chart, error)
}

type quotaChecker interface {
	CheckAndConsume(ctx context.Context, userID uint64) error
}

// ReadingService 简单测算业务编排。
type ReadingService struct {
	readingRepo    readingStore
	profileRepo    profileGetter
	chartService   chartProvider
	quotaService   quotaChecker
	llmProvider    llm.LLMProvider
	imgRenderer    renderer.Renderer
	fileStorage    storage.Storage
}

func NewReadingService(
	readingRepo *repository.ReadingRepo,
	profileRepo *repository.ProfileRepo,
	chartService *ChartService,
	quotaService *QuotaService,
	llmProvider llm.LLMProvider,
	imgRenderer renderer.Renderer,
	fileStorage storage.Storage,
) *ReadingService {
	return &ReadingService{
		readingRepo:  readingRepo,
		profileRepo:  profileRepo,
		chartService: chartService,
		quotaService: quotaService,
		llmProvider:  llmProvider,
		imgRenderer:  imgRenderer,
		fileStorage:  fileStorage,
	}
}

// CreateQuickInput 简单测算请求参数。
type CreateQuickInput struct {
	ProfileID uint64 `json:"profile_id"`
	Locale    string `json:"locale"`
	IsAdmin   bool   `json:"-"` // admin 豁免额度检查
}

// CreateQuick 编排完整简单测算链路：额度→排盘→LLM→渲染→存储→落库。
func (s *ReadingService) CreateQuick(ctx context.Context, userID uint64, in CreateQuickInput) (*model.Reading, error) {
	// 1. 额度校验（admin 豁免）
	if !in.IsAdmin {
		if err := s.quotaService.CheckAndConsume(ctx, userID); err != nil {
			return nil, err
		}
	} else {
		logger.FromCtx(ctx).Info("admin bypassed quota", "user_id", userID)
	}

	// 2. 取 Profile + 排盘（确定性，绝不走 LLM — P1）
	chart, err := s.chartService.Calculate(ctx, userID, CreateChartInput{ProfileID: in.ProfileID})
	if err != nil {
		return nil, fmt.Errorf("chart calculation: %w", err)
	}

	// 3. 组装 prompt，调 LLM 解读
	locale := in.Locale
	if locale == "" {
		locale = "en"
	}

	userPrompt, err := prompts.BuildQuickUserPrompt(locale, &chart.ChartData)
	if err != nil {
		return nil, fmt.Errorf("build prompt: %w", err)
	}

	llmResult, err := s.llmProvider.GenerateJSON(ctx, prompts.QuickSystemPrompt, userPrompt)
	if err != nil {
		logger.FromCtx(ctx).Error("llm generate failed", "err", err, "provider", s.llmProvider.Name())
		return nil, fmt.Errorf("llm generate: %w", err)
	}

	// 4. 严格解析 LLM JSON 到 QuickContent
	var content model.QuickContent
	if err := json.Unmarshal([]byte(llmResult), &content); err != nil {
		logger.FromCtx(ctx).Error("llm JSON parse failed", "err", err)
		return nil, fmt.Errorf("llm JSON parse: %w", err)
	}

	// 5. 渲染图片 PNG
	imgData := renderer.BuildQuickImageData(content, &chart.ChartData, locale, time.Now().UTC().Format("2006-01-02"))
	html, err := renderer.RenderTemplate(imgData)
	if err != nil {
		logger.FromCtx(ctx).Error("render template failed", "err", err)
		return nil, fmt.Errorf("render template: %w", err)
	}

	png, err := s.imgRenderer.Render(ctx, html, renderer.FormatPNG)
	if err != nil {
		logger.FromCtx(ctx).Error("render png failed", "err", err)
		return nil, fmt.Errorf("render png: %w", err)
	}

	// 6. 先落库获取 ID，再上传存储（需要 reading.ID 生成 key）
	reading := &model.Reading{
		UserID:    userID,
		ProfileID: in.ProfileID,
		ChartID:   chart.ID,
		Locale:    locale,
		Content:   content,
		Status:    "done",
		CreatedAt: time.Now(),
	}
	if err := s.readingRepo.Create(reading); err != nil {
		logger.FromCtx(ctx).Error("reading create failed", "err", err, "user_id", userID)
		return nil, fmt.Errorf("reading create: %w", err)
	}

	key := storage.ReadingKey(userID, reading.ID)
	imageURL, err := s.fileStorage.Put(ctx, key, png, "image/png")
	if err != nil {
		logger.FromCtx(ctx).Error("storage upload failed", "err", err, "key", key)
		return nil, fmt.Errorf("storage upload: %w", err)
	}

	// 7. 更新 ImageURL
	if err := s.readingRepo.UpdateImageURL(reading.ID, imageURL); err != nil {
		logger.FromCtx(ctx).Error("reading image_url update failed", "err", err, "reading_id", reading.ID)
		return nil, fmt.Errorf("reading image_url update: %w", err)
	}
	reading.ImageURL = imageURL

	return reading, nil
}

// GetByID 获取测算详情（带归属校验）。
func (s *ReadingService) GetByID(ctx context.Context, userID, readingID uint64) (*model.Reading, error) {
	reading, err := s.readingRepo.GetByID(readingID, userID)
	if err != nil {
		return nil, err
	}
	return reading, nil
}

// ListByUser 列出用户测算记录。
func (s *ReadingService) ListByUser(ctx context.Context, userID uint64, limit, offset int) ([]model.Reading, error) {
	return s.readingRepo.ListByUser(userID, limit, offset)
}
