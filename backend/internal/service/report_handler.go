package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"fatelumen/backend/internal/bazi"
	"fatelumen/backend/internal/job"
	"fatelumen/backend/internal/llm"
	"fatelumen/backend/internal/llm/prompts"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/hash"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/renderer"
	"fatelumen/backend/internal/repository"
	"fatelumen/backend/internal/storage"
)

// 私有接口：reportHandler 内部仅依赖接口，便于单测 fake
type reportGetter interface {
	GetByID(uint64, uint64) (*model.Report, error)
	UpdateStatus(uint64, string, string) error
	UpdateResult(uint64, model.ReportContent, string) error
	UpdateChartID(uint64, uint64) error
}

type chartSaver interface {
	FindByHash(string) (*model.Chart, error)
	Create(*model.Chart) error
}

// reportHandler 实现 job.JobHandler，Type="report"。
// 按顺序编排：排盘 → LLM → PDF → R2 → 落库。
type reportHandler struct {
	profileRepo profileGetter
	chartRepo   chartSaver
	llmProvider llm.LLMProvider
	renderer    renderer.Renderer
	storage     storage.Storage
	reportRepo  reportGetter
}

func NewReportHandler(
	profileRepo *repository.ProfileRepo,
	chartRepo *repository.ChartRepo,
	llmProvider llm.LLMProvider,
	renderer renderer.Renderer,
	storage storage.Storage,
	reportRepo *repository.ReportRepo,
) job.JobHandler {
	return &reportHandler{
		profileRepo: profileRepo,
		chartRepo:   chartRepo,
		llmProvider: llmProvider,
		renderer:    renderer,
		storage:     storage,
		reportRepo:  reportRepo,
	}
}

// Handle 执行异步报告生成全链路。每步失败 return err 让 worker 走重试/降级。
func (h *reportHandler) Handle(ctx context.Context, j *job.Job) (string, error) {
	// 1. 解析 payload
	var payload reportPayload
	if err := json.Unmarshal([]byte(j.Payload), &payload); err != nil {
		logger.FromCtx(ctx).Error("report payload parse failed", "err", err, "job_id", j.ID)
		return "", fmt.Errorf("parse payload: %w", err)
	}

	reportID := payload.ReportID
	userID := payload.UserID
	profileID := payload.ProfileID
	locale := payload.Locale
	if locale == "" {
		locale = "en"
	}

	// 幂等保护：report 已完成则直接返回已有结果，避免重复生成。
	// 场景：进程崩溃后 ReclaimStale 将孤儿 job 重置为 pending 重新调度，
	// 但原 handler 已达成 done 状态（report 已写库），此时无需重新跑全链路。
	existingReport, reportErr := h.reportRepo.GetByID(reportID, userID)
	if reportErr == nil && existingReport != nil && existingReport.Status == "done" {
		logger.FromCtx(ctx).Info("report already completed, idempotent skip",
			"report_id", reportID)
		return existingReport.PDFURL, nil
	}

	// 2. 取 Profile → bazi.Calculate 排盘（确定性，P1）
	profile, err := h.profileRepo.FindByID(profileID)
	if err != nil {
		logger.FromCtx(ctx).Error("profile not found for report", "err", err,
			"report_id", reportID, "profile_id", profileID)
		return "", fmt.Errorf("find profile: %w", err)
	}

	isLeap := profile.IsLeapMonth == 1
	chartData, err := bazi.Calculate(bazi.BirthInput{
		Gender:       profile.Gender,
		CalendarType: profile.CalendarType,
		Year:         int(profile.BirthYear),
		Month:        int(profile.BirthMonth),
		Day:          int(profile.BirthDay),
		Hour:         int(profile.BirthHour),
		Minute:       int(profile.BirthMinute),
		IsLeapMonth:  isLeap,
		Longitude:    profile.Longitude,
	})
	if err != nil {
		logger.FromCtx(ctx).Error("bazi calculate failed", "err", err,
			"report_id", reportID, "profile_id", profileID)
		return "", fmt.Errorf("bazi calculate: %w", err)
	}

	// 排盘落库（可复用）
	chartHash := hash.CalcChartHash(
		profile.Gender, profile.CalendarType,
		int(profile.BirthYear), int(profile.BirthMonth), int(profile.BirthDay),
		int(profile.BirthHour), int(profile.BirthMinute),
		isLeap, profile.Timezone,
	)

	var chartID uint64
	existing, err := h.chartRepo.FindByHash(chartHash)
	if err == nil && existing != nil {
		chartID = existing.ID
	} else {
		chart := &model.Chart{
			ProfileID: profileID,
			ChartHash: chartHash,
			ChartData: *chartData,
			CreatedAt: time.Now(),
		}
		if err := h.chartRepo.Create(chart); err != nil {
			logger.FromCtx(ctx).Error("chart create failed", "err", err,
				"report_id", reportID, "profile_id", profileID)
			return "", fmt.Errorf("chart create: %w", err)
		}
		chartID = chart.ID
	}

	if err := h.reportRepo.UpdateChartID(reportID, chartID); err != nil {
		logger.FromCtx(ctx).Error("report chart_id update failed", "err", err,
			"report_id", reportID, "chart_id", chartID)
		return "", fmt.Errorf("update chart_id: %w", err)
	}

	// 3. UpdateStatus → processing
	if err := h.reportRepo.UpdateStatus(reportID, "processing", ""); err != nil {
		logger.FromCtx(ctx).Error("report status update to processing failed", "err", err,
			"report_id", reportID)
		return "", fmt.Errorf("update status processing: %w", err)
	}

	// 4. LLM 解读
	userPrompt, err := prompts.BuildReportUserPrompt(locale, chartData)
	if err != nil {
		logger.FromCtx(ctx).Error("build report prompt failed", "err", err,
			"report_id", reportID)
		return "", fmt.Errorf("build prompt: %w", err)
	}

	llmStart := time.Now()
	llmResult, err := h.llmProvider.GenerateJSON(ctx, prompts.ReportSystemPrompt, userPrompt,
		llm.WithMaxTokens(4000),
		llm.WithTemperature(0.5),
	)
	if err != nil {
		logger.FromCtx(ctx).Error("llm generate failed", "err", err,
			"provider", h.llmProvider.Name(), "report_id", reportID,
			"elapsed_ms", time.Since(llmStart).Milliseconds())
		return "", fmt.Errorf("llm generate: %w", err)
	}
	logger.FromCtx(ctx).Info("llm generate completed",
		"report_id", reportID,
		"elapsed_ms", time.Since(llmStart).Milliseconds(),
	)

	var content model.ReportContent
	if err := json.Unmarshal([]byte(llmResult), &content); err != nil {
		logger.FromCtx(ctx).Error("llm JSON parse failed", "err", err, "report_id", reportID)
		return "", fmt.Errorf("llm JSON parse: %w", err)
	}

	// 5. 渲染 PDF
	pdfData := renderer.BuildReportPDFData(chartData, content, time.Now().UTC().Format("2006-01-02"))
	pdf, err := renderer.RenderReportPDF(ctx, h.renderer, pdfData)
	if err != nil {
		logger.FromCtx(ctx).Error("report pdf render failed", "err", err, "report_id", reportID)
		return "", fmt.Errorf("render pdf: %w", err)
	}

	// 6. 上传 R2
	key := storage.ReportKey(userID, reportID)
	pdfURL, err := h.storage.Put(ctx, key, pdf, "application/pdf")
	if err != nil {
		logger.FromCtx(ctx).Error("storage upload failed", "err", err,
			"key", key, "report_id", reportID)
		return "", fmt.Errorf("storage upload: %w", err)
	}

	// 7. 落库结果 + 状态 done
	if err := h.reportRepo.UpdateResult(reportID, content, pdfURL); err != nil {
		logger.FromCtx(ctx).Error("report result update failed", "err", err, "report_id", reportID)
		return "", fmt.Errorf("update result: %w", err)
	}

	if err := h.reportRepo.UpdateStatus(reportID, "done", ""); err != nil {
		logger.FromCtx(ctx).Error("report status update to done failed", "err", err, "report_id", reportID)
		return "", fmt.Errorf("update status done: %w", err)
	}

	logger.FromCtx(ctx).Info("report generation completed",
		"report_id", reportID,
		"user_id", userID,
		"profile_id", profileID,
	)
	return pdfURL, nil
}

// 编译期接口校验
var _ job.JobHandler = (*reportHandler)(nil)
