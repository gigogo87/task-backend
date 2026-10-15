package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// okHandler 代表限流之後的下一層：只要被呼叫就回 200。
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// newRateLimitTestHandler 建一個 burst 2、幾乎不補 token 的限流器：前兩次通過，第三次一定被擋。
func newRateLimitTestHandler(t *testing.T) http.Handler {
	t.Helper()
	rl := NewRateLimiter(0.001, 2)
	t.Cleanup(rl.Close)
	return Trace(RateLimit(okHandler, rl))
}

func sendFrom(h http.Handler, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRateLimit_WithinBurst_Returns200(t *testing.T) {
	// Arrange
	h := newRateLimitTestHandler(t)

	// Act
	first := sendFrom(h, "10.0.0.1:1234")
	second := sendFrom(h, "10.0.0.1:1234")

	// Assert
	assert.Equal(t, http.StatusOK, first.Code)
	assert.Equal(t, http.StatusOK, second.Code)
}

func TestRateLimit_BurstExceeded_Returns429WithRetryAfterAndTraceID(t *testing.T) {
	// Arrange
	h := newRateLimitTestHandler(t)
	sendFrom(h, "10.0.0.1:1234")
	sendFrom(h, "10.0.0.1:1234")

	// Act
	rec := sendFrom(h, "10.0.0.1:1234")

	// Assert
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
	assert.Contains(t, rec.Body.String(), `"code":"RATE_LIMITED"`)
	assert.Contains(t, rec.Body.String(), `"trace_id":"`+rec.Header().Get(TraceIDHeader)+`"`,
		"429 body should carry the trace id")
}

func TestRateLimit_SameIPDifferentPort_SharesLimit(t *testing.T) {
	// Arrange
	h := newRateLimitTestHandler(t)
	sendFrom(h, "10.0.0.1:1111")
	sendFrom(h, "10.0.0.1:2222")

	// Act
	rec := sendFrom(h, "10.0.0.1:3333")

	// Assert
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimit_DifferentIP_HasSeparateLimit(t *testing.T) {
	// Arrange
	h := newRateLimitTestHandler(t)
	sendFrom(h, "10.0.0.1:1234")
	sendFrom(h, "10.0.0.1:1234")

	// Act
	rec := sendFrom(h, "10.0.0.2:1234")

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
}
