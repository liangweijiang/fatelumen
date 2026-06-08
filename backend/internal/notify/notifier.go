package notify

import "context"

// Message 通知消息。
type Message struct {
	To       string                 // 收件人邮箱（或用户 ID）
	Template string                 // 模板标识，如 "report_ready" / "payment_succeeded"
	Locale   string                 // 多语言模板
	Data     map[string]interface{} // 模板变量
}

// Notifier 抽象对外通知（邮件/站内信/未来短信）。
// MVP 接 Resend；未配置时用 noop 实现（仅打日志），不阻塞主流程。
type Notifier interface {
	Send(ctx context.Context, msg Message) error
	Channel() string // "email" / "inapp"
}
