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

// RenderReportPDF 将 ReportPDFData 填入 A4 多页模板，渲染出 PDF 字节流。
func RenderReportPDF(ctx context.Context, r Renderer, data *ReportPDFData) ([]byte, error) {
	var buf bytes.Buffer
	if err := reportTmpl.Execute(&buf, data); err != nil {
		logger.FromCtx(ctx).Error("report template render failed", "err", err)
		return nil, fmt.Errorf("report template: %w", err)
	}
	html := buf.String()

	pdf, err := r.Render(ctx, html, FormatPDF)
	if err != nil {
		logger.FromCtx(ctx).Error("report pdf render failed", "err", err, "format", "pdf")
		return nil, fmt.Errorf("report pdf render: %w", err)
	}
	return pdf, nil
}
