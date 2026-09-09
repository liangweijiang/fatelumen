package service

import (
	"context"
	"strconv"

	"fatelumen/backend/internal/notify"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
)

type FullReportOutcomeNotifier interface {
	Completed(ctx context.Context, reportID uint64)
	Failed(ctx context.Context, reportID uint64, errorCode string)
}

type fullReportOutcomeNotifier struct {
	reports  *repository.FullReportRepo
	notifier notify.Notifier
}

func NewFullReportOutcomeNotifier(reports *repository.FullReportRepo, notifier notify.Notifier) FullReportOutcomeNotifier {
	return &fullReportOutcomeNotifier{reports: reports, notifier: notifier}
}

func (n *fullReportOutcomeNotifier) Completed(ctx context.Context, reportID uint64) {
	report, err := n.reports.GetByInternalID(ctx, reportID)
	if err != nil {
		logger.FromCtx(ctx).Error("load completed report for notification failed", "err", err, "report_id", reportID)
		return
	}
	result, err := n.reports.GetResult(ctx, reportID)
	if err != nil {
		logger.FromCtx(ctx).Error("load completed report result for notification failed", "err", err, "report_id", reportID)
		return
	}
	n.send(ctx, notify.Message{
		To:       strconv.FormatUint(report.UserID, 10),
		Template: "report_ready",
		Locale:   report.Locale,
		Data: map[string]interface{}{
			"report_id": report.PublicID,
			"pdf_url":   result.PDFURL,
		},
	}, reportID)
}

func (n *fullReportOutcomeNotifier) Failed(ctx context.Context, reportID uint64, errorCode string) {
	report, err := n.reports.GetByInternalID(ctx, reportID)
	if err != nil {
		logger.FromCtx(ctx).Error("load failed report for notification failed", "err", err, "report_id", reportID)
		return
	}
	n.send(ctx, notify.Message{
		To:       strconv.FormatUint(report.UserID, 10),
		Template: "report_failed",
		Locale:   report.Locale,
		Data: map[string]interface{}{
			"report_id":  report.PublicID,
			"error_code": errorCode,
		},
	}, reportID)
}

func (n *fullReportOutcomeNotifier) send(ctx context.Context, msg notify.Message, reportID uint64) {
	if n == nil || n.notifier == nil {
		return
	}
	if err := n.notifier.Send(ctx, msg); err != nil {
		logger.FromCtx(ctx).Error("full report notification failed", "err", err, "report_id", reportID, "channel", n.notifier.Channel(), "template", msg.Template)
	}
}
