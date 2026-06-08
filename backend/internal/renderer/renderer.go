package renderer

import "context"

// Format 渲染输出格式。
type Format string

const (
	FormatPNG Format = "png"
	FormatPDF Format = "pdf"
)

// Renderer 抽象「HTML → 二进制图片/PDF」。业务层只依赖此接口，
// 未来可从 chromedp 换成 wkhtmltopdf / 云渲染服务，不动业务代码。
type Renderer interface {
	// Render 输入已渲染好的 HTML 字符串，返回目标格式的字节流。
	Render(ctx context.Context, html string, format Format) ([]byte, error)
}
