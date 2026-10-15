package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"taskapi/internal/middleware"
)

// 整合測試：透過 DependencyProvider 組出完整的 handler（middleware + controller + service + repository），跑一遍真實流程。

func newTestHandler(t *testing.T, cfg Config) http.Handler {
	t.Helper()
	dp := NewDependencyProvider(cfg, io.Discard)
	t.Cleanup(dp.Close)
	return dp.Handler()
}

var generousLimits = Config{RateLimitRPS: 1000, RateLimitBurst: 1000}

func doRequest(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHandler_CreateUpdateDeleteSequence_ListReturnsRemainingTask(t *testing.T) {
	// Arrange
	h := newTestHandler(t, generousLimits)
	steps := []struct {
		method, path, body string
		wantStatus         int
	}{
		{http.MethodPost, "/task", `{"name":"買牛奶"}`, http.StatusCreated},
		{http.MethodPost, "/task", `{"name":"寫作業"}`, http.StatusCreated},
		{http.MethodPut, "/task/1", `{"name":"買牛奶","status":1}`, http.StatusOK},
		{http.MethodDelete, "/task/2", "", http.StatusNoContent},
	}
	for _, s := range steps {
		rec := doRequest(t, h, s.method, s.path, s.body)
		require.Equal(t, s.wantStatus, rec.Code, "arrange %s %s, body: %s", s.method, s.path, rec.Body.String())
	}

	// Act
	rec := doRequest(t, h, http.MethodGet, "/tasks", "")

	// Assert
	assert.JSONEq(t, `{"result":[{"id":1,"name":"買牛奶","status":1}]}`, rec.Body.String())
}

func TestHandler_SuccessfulRequest_SetsTraceIDHeader(t *testing.T) {
	// Arrange
	h := newTestHandler(t, generousLimits)

	// Act
	rec := doRequest(t, h, http.MethodGet, "/tasks", "")

	// Assert
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get(middleware.TraceIDHeader), "successful responses should also carry X-Trace-Id")
}

func TestHandler_TaskNotFound_ReturnsErrorWithTraceIDMatchingHeader(t *testing.T) {
	// Arrange
	h := newTestHandler(t, generousLimits)

	// Act
	rec := doRequest(t, h, http.MethodPut, "/task/999", `{"name":"x"}`)

	// Assert
	require.Equal(t, http.StatusNotFound, rec.Code)
	var resp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			TraceID string `json:"trace_id"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "TASK_NOT_FOUND", resp.Error.Code)
	assert.Equal(t, "task not found", resp.Error.Message)
	assert.NotEmpty(t, resp.Error.TraceID)
	assert.Equal(t, rec.Header().Get(middleware.TraceIDHeader), resp.Error.TraceID, "trace_id in body should match X-Trace-Id header")
}

func TestHandler_BurstExceeded_Returns429(t *testing.T) {
	// Arrange
	h := newTestHandler(t, Config{RateLimitRPS: 0.001, RateLimitBurst: 1})
	first := doRequest(t, h, http.MethodGet, "/tasks", "")
	require.Equal(t, http.StatusOK, first.Code, "arrange: first request")

	// Act
	rec := doRequest(t, h, http.MethodGet, "/tasks", "")

	// Assert
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}
