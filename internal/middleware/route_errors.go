package middleware

import (
	"net/http"

	"taskapi/internal/response"
)

// Router 是 RouteErrors 需要的路由器：除了處理請求，還要能回答「這個請求有沒有對應的路由」。
// *http.ServeMux 就符合這個介面。
type Router interface {
	http.Handler
	Handler(r *http.Request) (h http.Handler, pattern string)
}

// RouteErrors 把路由器內建的 404 / 405 純文字回應，改成統一格式的 JSON 錯誤。
// 必須直接包在路由器外面（最內層），因為它要問路由器有沒有符合的路由。
//
// 有符合的路由：照常交給路由器，controller 自己回的 404（例如 TASK_NOT_FOUND）不受影響。
// 沒有符合的路由：讓路由器跑它內建的處理，只記下 status 與 Allow header，再改回 JSON。
// Allow 由 ServeMux 依照實際登記的路由算出，不需要手寫。
func RouteErrors(next Router) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, pattern := next.Handler(r)
		if pattern != "" {
			next.ServeHTTP(w, r)
			return
		}

		rec := &statusRecorder{header: make(http.Header)}
		h.ServeHTTP(rec, r)

		if rec.status == http.StatusMethodNotAllowed {
			w.Header().Set("Allow", rec.header.Get("Allow"))
			response.WriteError(w, r, response.ErrMethodNotAllowed)
			return
		}
		response.WriteError(w, r, response.ErrRouteNotFound)
	})
}

// statusRecorder 只記錄 status 與 header，丟掉 body（我們要換成 JSON）。
type statusRecorder struct {
	header http.Header
	status int // 0 代表還沒寫入 status
}

func (s *statusRecorder) Header() http.Header { return s.header }

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return len(b), nil
}
