package renderer

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"fatelumen/backend/internal/pkg/logger"
)

var reportTmpl *template.Template

func init() {
	var err error
	reportTmpl, err = template.ParseFS(templateFS, "templates/report_pdf.html")
	if err != nil {
		panic(fmt.Sprintf("renderer: failed to parse report_pdf.html: %v", err))
	}
}

// RenderReportHTML 将 ReportPDFData 填入模板，仅渲染出 HTML 字符串（供在线报告页直接使用）。
func RenderReportHTML(ctx context.Context, data *ReportPDFData) (string, error) {
	var buf bytes.Buffer
	if err := reportTmpl.Execute(&buf, data); err != nil {
		logger.FromCtx(ctx).Error("report template render failed", "err", err)
		return "", fmt.Errorf("report template: %w", err)
	}
	return buf.String(), nil
}

// RenderReportPDF 复用 RenderReportHTML 产出的 HTML，再经 Renderer 转为 PDF 字节流。
func RenderReportPDF(ctx context.Context, r Renderer, data *ReportPDFData) ([]byte, error) {
	html, err := RenderReportHTML(ctx, data)
	if err != nil {
		return nil, err
	}
	pdf, err := r.Render(ctx, html, FormatPDF)
	if err != nil {
		logger.FromCtx(ctx).Error("report pdf render failed", "err", err, "format", "pdf")
		return nil, fmt.Errorf("report pdf render: %w", err)
	}
	return pdf, nil
}
