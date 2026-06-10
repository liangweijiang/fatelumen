package logger

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

func TestWithTraceID_RoundTrip(t *testing.T) {
	ctx := context.Background()
	ctx = WithTraceID(ctx, "abc123")

	tid := TraceIDFromCtx(ctx)
	if tid != "abc123" {
		t.Fatalf("expected 'abc123', got '%s'", tid)
	}
}

func TestTraceIDFromCtx_Empty(t *testing.T) {
	ctx := context.Background()
	tid := TraceIDFromCtx(ctx)
	if tid != "" {
		t.Fatalf("expected empty, got '%s'", tid)
	}
}

func TestFromCtx_WithTraceID(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	defaultLogger = slog.New(handler)

	ctx := WithTraceID(context.Background(), "trace-456")
	logger := FromCtx(ctx)
	logger.Info("test message", "key", "val")

	output := buf.String()
	if output == "" {
		t.Fatal("expected log output, got empty")
	}
	if !bytes.Contains([]byte(output), []byte("trace_id")) || !bytes.Contains([]byte(output), []byte("trace-456")) {
		t.Fatalf("expected trace_id=trace-456 in output, got: %s", output)
	}
}

func TestFromCtx_NoTraceID(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	defaultLogger = slog.New(handler)

	ctx := context.Background()
	logger := FromCtx(ctx)
	logger.Info("no trace message", "foo", "bar")

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("no trace message")) {
		t.Fatalf("expected message in output, got: %s", output)
	}
}
