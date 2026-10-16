package trace

import (
	"context"
	"log/slog"
)

// LogHandler wraps any slog.Handler and adds the context's trace_id to every log record.
// Layers only need to log with ctx-aware methods such as logger.InfoContext(ctx, ...); they never pass the trace id themselves.
type LogHandler struct {
	slog.Handler
}

// NewLogHandler wraps h so every record gets the context's trace_id.
func NewLogHandler(h slog.Handler) *LogHandler {
	return &LogHandler{Handler: h}
}

// Handle adds the trace_id from ctx, if any, and passes the record on.
func (h *LogHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := FromContext(ctx); id != "" {
		r.AddAttrs(slog.String("trace_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

// WithAttrs returns a wrapped LogHandler, so loggers derived with logger.With keep the trace_id.
func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogHandler{Handler: h.Handler.WithAttrs(attrs)}
}

// WithGroup returns a wrapped LogHandler, so loggers derived with logger.WithGroup keep the trace_id.
func (h *LogHandler) WithGroup(name string) slog.Handler {
	return &LogHandler{Handler: h.Handler.WithGroup(name)}
}
