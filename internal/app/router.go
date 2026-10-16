package app

import (
	"net/http"

	"taskapi/internal/controller"
	"taskapi/internal/middleware"
)

// newRouter registers all routes, chains the middleware and returns the entry handler of the web server.
// Since Go 1.22 the standard library ServeMux supports method + path-parameter patterns, so no third-party router is needed.
func newRouter(tasks *controller.TaskController, rateLimiter *middleware.RateLimiter) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /tasks", tasks.List)
	mux.HandleFunc("POST /tasks", tasks.Create)
	mux.HandleFunc("PUT /tasks/{id}", tasks.Update)
	mux.HandleFunc("DELETE /tasks/{id}", tasks.Delete)

	var h http.Handler = middleware.RouteErrors(mux) // must wrap the mux directly
	h = middleware.RateLimit(h, rateLimiter)
	h = middleware.Trace(h)
	return h
}
