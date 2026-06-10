package renderer

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"time"

	"fatelumen/backend/internal/pkg/logger"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

//go:embed templates/*
var templateFS embed.FS

var quickTmpl *template.Template

func init() {
	var err error
	quickTmpl, err = template.ParseFS(templateFS, "templates/quick_image.html")
	if err != nil {
		panic(fmt.Sprintf("renderer: failed to parse quick_image.html: %v", err))
	}
}

// ChromedpRenderer 用 headless Chrome 渲染 HTML 为图片/PDF。
// 运行环境要求：安装 chromium + fonts-noto-cjk，否则中日韩文渲染成方块。
type ChromedpRenderer struct {
	chromiumPath string
}

func NewChromedpRenderer(chromiumPath string) *ChromedpRenderer {
	return &ChromedpRenderer{chromiumPath: chromiumPath}
}

func (r *ChromedpRenderer) Render(ctx context.Context, html string, format Format) ([]byte, error) {
	switch format {
	case FormatPNG:
		return r.renderPNG(ctx, html)
	case FormatPDF:
		return r.renderPDF(ctx, html)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func (r *ChromedpRenderer) renderPNG(ctx context.Context, html string) ([]byte, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.DisableGPU,
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.WindowSize(1080, 1350),
	)
	if r.chromiumPath != "" {
		opts = append(opts, chromedp.ExecPath(r.chromiumPath))
	}

	actx, acancel := chromedp.NewExecAllocator(ctx, opts...)
	defer acancel()

	ctx, cancel := chromedp.NewContext(actx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.EmulateViewport(1080, 1350),
		chromedp.ActionFunc(func(ctx context.Context) error {
			tree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(tree.Frame.ID, html).Do(ctx)
		}),
		chromedp.WaitReady("body"),
		chromedp.FullScreenshot(&buf, 100),
	)
	if err != nil {
		logger.FromCtx(ctx).Error("chromedp screenshot failed", "err", err, "format", "png")
		return nil, fmt.Errorf("chromedp screenshot: %w", err)
	}
	if len(buf) == 0 {
		logger.FromCtx(ctx).Error("chromedp produced empty screenshot", "format", "png")
		return nil, errors.New("chromedp produced empty screenshot")
	}
	return buf, nil
}

func (r *ChromedpRenderer) renderPDF(ctx context.Context, html string) ([]byte, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.DisableGPU,
		chromedp.Flag("disable-dev-shm-usage", true),
	)
	if r.chromiumPath != "" {
		opts = append(opts, chromedp.ExecPath(r.chromiumPath))
	}

	actx, acancel := chromedp.NewExecAllocator(ctx, opts...)
	defer acancel()

	ctx, cancel := chromedp.NewContext(actx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			tree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(tree.Frame.ID, html).Do(ctx)
		}),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			buf, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.5).
				WithPaperHeight(11).
				Do(ctx)
			return err
		}),
	)
	if err != nil {
		logger.FromCtx(ctx).Error("chromedp pdf failed", "err", err, "format", "pdf")
		return nil, fmt.Errorf("chromedp pdf: %w", err)
	}
	if len(buf) == 0 {
		logger.FromCtx(ctx).Error("chromedp produced empty pdf", "format", "pdf")
		return nil, errors.New("chromedp produced empty pdf")
	}
	return buf, nil
}

// RenderTemplate 将 QuickImageData 填入 quick_image.html → 完整 HTML 字符串。
func RenderTemplate(data *QuickImageData) (string, error) {
	var buf bytes.Buffer
	if err := quickTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}
	return buf.String(), nil
}
