package trace

import (
	"context"
	"log/slog"
)

// LogHandler 包住任何 slog.Handler，自動在每筆 log 加上 context 裡的 trace_id。
// 各層只要用 logger.InfoContext(ctx, ...) 這類帶 ctx 的方法，就不需要自己傳 trace id。
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

// WithAttrs、WithGroup 要回傳包好的 LogHandler，否則 logger.With(...) 之後就會失去 trace_id。
func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *LogHandler) WithGroup(name string) slog.Handler {
	return &LogHandler{Handler: h.Handler.WithGroup(name)}
}
