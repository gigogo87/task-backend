// Package trace 管理每個請求的 trace id。
// 之後接 OpenTelemetry 等工具時，只需改這個 package。
package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type ctxKey struct{}

// NewContext 把 trace id 放進 context，讓之後的每一層都拿得到。
func NewContext(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, traceID)
}

// FromContext 取出 trace id；沒有的話回傳空字串。
func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

// NewID 產生一個隨機的 trace id（16 bytes → 32 字元小寫 hex）。
func NewID() string {
	var b [16]byte
	_, _ = rand.Read(b[:]) // crypto/rand.Read 不會回傳錯誤
	return hex.EncodeToString(b[:])
}

const maxIDLength = 128

// IsValidID 檢查從外部（例如 request header）收到的 trace id 能不能沿用：
// 不可為空、最長 128 字元，只能包含英數字與 '-'、'_'、'.'（UUID、hex 等常見格式都符合）。
// trace id 會原樣寫進 log 與回應 header，限制格式可以避免被塞入換行偽造 log，或塞入超長字串。
func IsValidID(id string) bool {
	if id == "" || len(id) > maxIDLength {
		return false
	}
	for _, c := range id {
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}
