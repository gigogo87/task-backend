package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testBurst = 2     // 每個 IP 瞬間最多可以連打幾次
	testRPS   = 0.001 // 幾乎不補 token：用完 burst 之後，測試期間不會再被放行
)

// okHandler 代表限流之後的下一層：只要被呼叫就回 200。
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// newRateLimitTestHandler 建一個 burst 為 testBurst、幾乎不補 token 的限流器：
// 前 testBurst 次通過，之後一定被擋。
func newRateLimitTestHandler(t *testing.T) http.Handler {
	t.Helper()
	rl := NewRateLimiter(testRPS, testBurst)
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

// exhaustBurst 從同一個位址送出 testBurst 次請求，把該 IP 的額度用完。
func exhaustBurst(h http.Handler, remoteAddr string) {
	for i := 0; i < testBurst; i++ {
		sendFrom(h, remoteAddr)
	}
}

func TestRateLimit_WithinBurst_Returns200(t *testing.T) {
	// Arrange
	h := newRateLimitTestHandler(t)

	for i := 0; i < testBurst; i++ {
		// Act
		rec := sendFrom(h, "10.0.0.1:1234")

		// Assert
		assert.Equal(t, http.StatusOK, rec.Code, "request %d of %d", i+1, testBurst)
	}
}

func TestRateLimit_BurstExceeded_Returns429WithRetryAfterAndTraceID(t *testing.T) {
	// Arrange
	h := newRateLimitTestHandler(t)
	exhaustBurst(h, "10.0.0.1:1234")

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
	for i := 0; i < testBurst; i++ {
		sendFrom(h, fmt.Sprintf("10.0.0.1:%d", 1000+i)) // 每次都換一個 port
	}

	// Act
	rec := sendFrom(h, "10.0.0.1:9999")

	// Assert
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimit_DifferentIP_HasSeparateLimit(t *testing.T) {
	// Arrange
	h := newRateLimitTestHandler(t)
	exhaustBurst(h, "10.0.0.1:1234")

	// Act
	rec := sendFrom(h, "10.0.0.2:1234")

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
}
