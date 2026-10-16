package middleware

import (
	"net/http"

	"taskapi/internal/trace"
)

// TraceIDHeader carries the trace id: it is reused when the request sends it and always set on responses.
const TraceIDHeader = "X-Trace-Id"

// Trace decides the trace id of each request and puts it in the context:
// a valid X-Trace-Id request header is reused; otherwise a new id is generated.
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
