package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"fatelumen/backend/internal/job"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/renderer"
	"fatelumen/backend/internal/repository"
	"fatelumen/backend/internal/storage"
)

const FullReportPDFRenderVersion = "full-report-pdf-v2"

type FullReportPDFScheduler interface {
	PrepareAndEnqueue(ctx context.Context, reportID uint64, result *model.FullReportResult) error
}

type fullReportPDFPipeline struct {
	reports  *repository.FullReportRepo
	queue    job.Queue
	renderer renderer.Renderer
	storage  storage.Storage
	outcomes FullReportOutcomeNotifier
}

func NewFullReportPDFPipeline(reports *repository.FullReportRepo, queue job.Queue, imageRenderer renderer.Renderer, fileStorage storage.Storage, outcomes ...FullReportOutcomeNotifier) *fullReportPDFPipeline {
	var outcomeNotifier FullReportOutcomeNotifier
	if len(outcomes) > 0 {
		outcomeNotifier = outcomes[0]
	}
	return &fullReportPDFPipeline{reports: reports, queue: queue, renderer: imageRenderer, storage: fileStorage, outcomes: outcomeNotifier}
}

func (p *fullReportPDFPipeline) PrepareAndEnqueue(ctx context.Context, reportID uint64, result *model.FullReportResult) error {
	task, err := p.reports.PrepareRendering(ctx, reportID, result, FullReportPDFRenderVersion, 3, time.Now().UTC())
	if err != nil {
		return err
	}
	if err := enqueueFullReportPDFTask(ctx, p.queue, task); err != nil {
		// The durable task remains queued; startup/periodic recovery will deliver it.
		logger.FromCtx(ctx).Error("enqueue pdf render task failed; durable recovery will retry", "err", err, "report_id", reportID, "render_job_id", task.ID)
	}
	return nil
}

func enqueueFullReportPDFTask(ctx context.Context, queue job.Queue, task *model.FullReportRenderJob) error {
	payload, err := json.Marshal(fullReportPDFJobPayload{RenderJobID: task.ID})
	if err != nil {
		return err
	}
	return queue.Enqueue(ctx, &job.Job{Type: FullReportPDFJobType, Lane: job.LanePDFRender, Payload: payload, MaxAttempts: int(task.MaxAttempts)})
}

func (p *fullReportPDFPipeline) Run(ctx context.Context, task *model.FullReportRenderJob) error {
	report, err := p.reports.GetByInternalID(ctx, task.ReportID)
	if err != nil {
		return err
	}
	result, err := p.reports.GetResult(ctx, task.ReportID)
	if err != nil {
		return err
	}
	if report.Status == model.FullReportStatusCompleted && result.PDFStorageKey != "" && result.PDFURL != "" && len(result.PDFHash) == 64 {
		return nil
	}
	payload, err := p.reports.GetExecutionPayload(ctx, task.ReportID)
	if err != nil {
		return err
	}
	var chart model.ChartSnapshot
	if err := json.Unmarshal(payload.ChartSnapshot, &chart); err != nil {
		return fmt.Errorf("decode frozen chart snapshot: %w", err)
	}
	pdfData := renderer.BuildReportPDFData(&chart.Data, result.Content, result.CreatedAt.UTC().Format("2006-01-02"))
	pdfBytes, err := renderer.RenderReportPDF(ctx, p.renderer, pdfData)
	if err != nil {
		return fmt.Errorf("render pdf: %w", err)
	}
	sum := sha256.Sum256(pdfBytes)
	pdfHash := hex.EncodeToString(sum[:])
	key := fmt.Sprintf("reports/%s/%s/%s/report.pdf", report.PublicID, result.ContentHash, task.RenderVersion)
	url, err := p.storage.Put(ctx, key, pdfBytes, "application/pdf")
	if err != nil {
		logger.FromCtx(ctx).Error("full report pdf upload failed", "err", err, "report_id", task.ReportID, "key", key)
		return err
	}
	if err := p.reports.CompleteRendering(ctx, task.ReportID, key, url, pdfHash, time.Now().UTC()); err != nil {
		return err
	}
	if p.outcomes != nil {
		p.outcomes.Completed(context.WithoutCancel(ctx), task.ReportID)
	}
	return nil
}
