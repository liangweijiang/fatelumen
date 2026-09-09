package notify

import (
	"context"

	"fatelumen/backend/internal/pkg/logger"
)

// NoopNotifier keeps notification call sites active when no external channel
// is configured. It deliberately logs metadata only, never message contents.
type NoopNotifier struct{}

func NewNoopNotifier() *NoopNotifier { return &NoopNotifier{} }

func (n *NoopNotifier) Send(ctx context.Context, msg Message) error {
	logger.FromCtx(ctx).Info("notification skipped by noop channel", "template", msg.Template, "locale", msg.Locale)
	return nil
}

func (n *NoopNotifier) Channel() string { return "noop" }
