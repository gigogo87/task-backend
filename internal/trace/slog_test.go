package trace

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"

func newTestLogger(buf *bytes.Buffer, level slog.Level) *slog.Logger {
	inner := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: level})
	return slog.New(NewLogHandler(inner))
}

func decodeLogLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry), "log line: %s", buf.String())
	return entry
}

func TestHandle_ContextHasTraceID_AddsTraceIDAttr(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	logger := newTestLogger(&buf, slog.LevelInfo)
	ctx := NewContext(context.Background(), testTraceID)

	// Act
	logger.InfoContext(ctx, "task created", "task_id", 1)

	// Assert
	entry := decodeLogLine(t, &buf)
	assert.Equal(t, testTraceID, entry["trace_id"])
	assert.Equal(t, "task created", entry["msg"])
	assert.Equal(t, float64(1), entry["task_id"]) // JSON numbers decode to float64
}

func TestHandle_ContextWithoutTraceID_OmitsTraceIDAttr(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	logger := newTestLogger(&buf, slog.LevelInfo)

	// Act
	logger.InfoContext(context.Background(), "server listening")

	// Assert
	entry := decodeLogLine(t, &buf)
	assert.NotContains(t, entry, "trace_id")
	assert.Equal(t, "server listening", entry["msg"])
}

func TestWithAttrs_LoggerWithAttrs_KeepsTraceIDAndAttrs(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	logger := newTestLogger(&buf, slog.LevelInfo).With("component", "service")
	ctx := NewContext(context.Background(), testTraceID)

	// Act
	logger.InfoContext(ctx, "task deleted")

	// Assert
	entry := decodeLogLine(t, &buf)
	assert.Equal(t, testTraceID, entry["trace_id"])
	assert.Equal(t, "service", entry["component"])
}

func TestWithGroup_LoggerWithGroup_KeepsTraceIDInsideGroup(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	logger := newTestLogger(&buf, slog.LevelInfo).WithGroup("request")
	ctx := NewContext(context.Background(), testTraceID)

	// Act
	logger.InfoContext(ctx, "handled", "method", "GET")

	// Assert
	entry := decodeLogLine(t, &buf)
	group, ok := entry["request"].(map[string]any)
	require.True(t, ok, "want a request group, got %v", entry)
	assert.Equal(t, testTraceID, group["trace_id"])
	assert.Equal(t, "GET", group["method"])
}

func TestEnabled_LevelBelowInnerHandlerLevel_WritesNothing(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	logger := newTestLogger(&buf, slog.LevelWarn)
	ctx := NewContext(context.Background(), testTraceID)

	// Act
	logger.InfoContext(ctx, "should be filtered")

	// Assert
	assert.Empty(t, buf.String())
}
