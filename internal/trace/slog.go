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

func NewLogHandler(h slog.Handler) *LogHandler {
	return &LogHandler{Handler: h}
}

func (h *LogHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := FromContext(ctx); id != "" {
		r.AddAttrs(slog.String("trace_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *LogHandler) WithGroup(name string) slog.Handler {
	return &LogHandler{Handler: h.Handler.WithGroup(name)}
}
