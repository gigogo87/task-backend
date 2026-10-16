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
	testBurst = 2     // maximum burst of requests per IP
	testRPS   = 0.001 // almost no refill: once the burst is used, nothing more gets through during the test
)

// okHandler stands for the next layer after the rate limiter: it returns 200 whenever called.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func newRateLimitTestHandler(t *testing.T) http.Handler {
	t.Helper()
	rl := NewRateLimiter(testRPS, testBurst)
	t.Cleanup(rl.Close)
	return RateLimit(okHandler, rl)
}

func sendFrom(h http.Handler, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// exhaustBurst sends testBurst requests from one address, using up that IP's quota.
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

func TestRateLimit_BurstExceeded_Returns429WithRetryAfter(t *testing.T) {
	// Arrange
	h := newRateLimitTestHandler(t)
	exhaustBurst(h, "10.0.0.1:1234")

	// Act
	rec := sendFrom(h, "10.0.0.1:1234")

	// Assert
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
	assert.Contains(t, rec.Body.String(), `"code":"RATE_LIMITED"`)
}

func TestRateLimit_SameIPDifferentPort_SharesLimit(t *testing.T) {
	// Arrange
	h := newRateLimitTestHandler(t)
	for i := 0; i < testBurst; i++ {
		sendFrom(h, fmt.Sprintf("10.0.0.1:%d", 1000+i)) // a different port each time
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
