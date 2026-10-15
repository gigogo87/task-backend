// Package middleware 放包在所有路由外面的 HTTP middleware，簽名統一是 func(http.Handler) http.Handler。
package middleware

import (
	"net/http"

	"taskapi/internal/trace"
)

// TraceIDHeader 是帶 trace id 的 header：請求帶了就沿用，回應（成功和失敗）一律會帶。
const TraceIDHeader = "X-Trace-Id"

// Trace 替每個請求決定 trace id 並放進 context：
// 請求的 X-Trace-Id header 有合法的值就沿用（才能和上游串成同一條鏈），沒帶或格式不合法就自己產生。
// 必須放在最外層，後面的 middleware（例如限流回 429）才拿得到 trace id。
func Trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(TraceIDHeader)
		if !trace.IsValidID(id) {
			id = trace.NewID()
		}
		w.Header().Set(TraceIDHeader, id)
		next.ServeHTTP(w, r.WithContext(trace.NewContext(r.Context(), id)))
	})
}
