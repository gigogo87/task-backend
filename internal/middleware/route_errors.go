package middleware

import (
	"net/http"

	"taskapi/internal/response"
)

// Router is what RouteErrors needs: besides serving requests, it must tell whether a request matches a route.
// *http.ServeMux satisfies this interface.
type Router interface {
	http.Handler
	Handler(r *http.Request) (h http.Handler, pattern string)
}

// RouteErrors turns the router's built-in plain-text 404 / 405 responses into the common JSON error format.
// It must wrap the router directly (innermost), because it asks the router whether a route matches.
//
// Matching route: pass through to the router; a 404 returned by a controller (e.g. TASK_NOT_FOUND) is untouched.
// No matching route: let the router run its built-in handler, record only the status and Allow header, then answer in JSON.
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

// statusRecorder records only the status and headers and discards the body (it is replaced by JSON).
type statusRecorder struct {
	header http.Header
	status int // 0 means no status written yet
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
