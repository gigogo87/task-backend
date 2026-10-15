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

// newTestLogger 回傳一個把 JSON log 寫進 buf 的 logger，handler 外層包著待測的 LogHandler。
func newTestLogger(buf *bytes.Buffer, level slog.Level) *slog.Logger {
	inner := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: level})
	return slog.New(NewLogHandler(inner))
}

// decodeLogLine 把 buf 裡唯一的一筆 JSON log 解析成 map。
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
	assert.Equal(t, float64(1), entry["task_id"]) // JSON 數字解析成 float64
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

// logger.With(...) 會走 WithAttrs；若沒有回傳包好的 LogHandler，之後的 log 就會失去 trace_id。
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

// logger.WithGroup(...) 會走 WithGroup；trace_id 跟其他 attr 一樣會被放進 group 裡。
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

// 等級過濾交給內層 handler：低於設定等級的 log 不會輸出。
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
