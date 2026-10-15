package app

import (
	"net/http"

	"taskapi/internal/controller"
	"taskapi/internal/middleware"
)

// newRouter 註冊所有路由並串好 middleware，回傳整個 web server 的入口 handler。
// Go 1.22 起，標準庫的 ServeMux 就支援「方法 + 路徑參數」，不用第三方套件。
//
// middleware 由內往外包，請求進來時則由外往內執行：Trace → RateLimit → RouteErrors → 路由 → controller。
// Trace 放最外層，後面每一層（包含限流回 429）的錯誤回應才會有 trace_id。
func newRouter(tasks *controller.TaskController, rateLimiter *middleware.RateLimiter) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /tasks", tasks.List)
	mux.HandleFunc("POST /tasks", tasks.Create)
	mux.HandleFunc("PUT /tasks/{id}", tasks.Update)
	mux.HandleFunc("DELETE /tasks/{id}", tasks.Delete)

	var h http.Handler = middleware.RouteErrors(mux) // 必須直接包住 mux
	h = middleware.RateLimit(h, rateLimiter)
	h = middleware.Trace(h)
	return h
}
