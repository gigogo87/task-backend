package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRouteErrorsTestHandler builds a mux with two routes and wraps it in RouteErrors.
func newRouteErrorsTestHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /x", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /missing/{id}", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "handler 404", http.StatusNotFound)
	})
	return RouteErrors(mux)
}

func serveRoute(h http.Handler, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestRouteErrors_MatchedRoute_PassesThroughToHandler(t *testing.T) {
	// Arrange
	h := newRouteErrorsTestHandler()

	// Act
	rec := serveRoute(h, http.MethodGet, "/x")

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestRouteErrors_HandlerReturns404_KeepsHandlerResponse(t *testing.T) {
	// Arrange
	h := newRouteErrorsTestHandler()

	// Act
	rec := serveRoute(h, http.MethodGet, "/missing/1")

	// Assert
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "handler 404")
}

func TestRouteErrors_UnknownPath_Returns404RouteNotFoundJSON(t *testing.T) {
	// Arrange
	h := newRouteErrorsTestHandler()

	// Act
	rec := serveRoute(h, http.MethodGet, "/nope")

	// Assert
	require.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code":"ROUTE_NOT_FOUND"`)
}

func TestRouteErrors_WrongMethod_Returns405JSONWithAllowHeader(t *testing.T) {
	// Arrange
	h := newRouteErrorsTestHandler()

	// Act
	rec := serveRoute(h, http.MethodPost, "/x")

	// Assert
	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	assert.Equal(t, "GET, HEAD", rec.Header().Get("Allow"))
	assert.Contains(t, rec.Body.String(), `"code":"METHOD_NOT_ALLOWED"`)
}
