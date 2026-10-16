package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"taskapi/internal/middleware"
)

// newTestRouter builds the fully wired router and releases the provider's resources when the test ends.
func newTestRouter(t *testing.T, cfg Config) http.Handler {
	t.Helper()
	dp := NewDependencyProvider(cfg, io.Discard)
	t.Cleanup(dp.Close)
	return dp.Router
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

// createTask creates a task through the API and returns the id assigned by the server.
// Like a real client, the tests take the id from the response instead of assuming where ids start.
func createTask(t *testing.T, h http.Handler, name string) int {
	t.Helper()
	rec := doRequest(t, h, http.MethodPost, "/tasks", fmt.Sprintf(`{"name":%q}`, name))
	require.Equal(t, http.StatusCreated, rec.Code, "create task, body: %s", rec.Body.String())
	var resp struct {
		Result struct {
			ID int `json:"id"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp.Result.ID
}

func taskPath(id int) string {
	return fmt.Sprintf("/tasks/%d", id)
}

func errorCode(t *testing.T, body []byte) string {
	t.Helper()
	var resp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(body, &resp), "body: %s", body)
	return resp.Error.Code
}

// ---------- Happy paths ----------

func TestNewRouter_CreateUpdateDeleteThenList_ReturnsRemainingTask(t *testing.T) {
	// Arrange
	h := newTestRouter(t, generousLimits)
	previewID := createTask(t, h, "Preview PR")
	notesID := createTask(t, h, "Write release notes")
	updated := doRequest(t, h, http.MethodPut, taskPath(previewID), `{"name":"Preview PR","status":1}`)
	require.Equal(t, http.StatusOK, updated.Code, "arrange: update, body: %s", updated.Body.String())
	deleted := doRequest(t, h, http.MethodDelete, taskPath(notesID), "")
	require.Equal(t, http.StatusNoContent, deleted.Code, "arrange: delete")

	// Act
	rec := doRequest(t, h, http.MethodGet, "/tasks", "")

	// Assert
	want := fmt.Sprintf(`{"result":[{"id":%d,"name":"Preview PR","status":1}]}`, previewID)
	assert.JSONEq(t, want, rec.Body.String())
}

func TestNewRouter_DeleteSameTaskTwice_Returns204BothTimes(t *testing.T) {
	// Arrange
	h := newTestRouter(t, generousLimits)
	id := createTask(t, h, "x")

	// Act
	first := doRequest(t, h, http.MethodDelete, taskPath(id), "")
	second := doRequest(t, h, http.MethodDelete, taskPath(id), "")

	// Assert
	assert.Equal(t, http.StatusNoContent, first.Code)
	assert.Equal(t, http.StatusNoContent, second.Code)
	assert.Empty(t, second.Body.String())
}

func TestNewRouter_DeleteNeverCreatedTask_Returns204(t *testing.T) {
	// Arrange
	h := newTestRouter(t, generousLimits)

	// Act
	rec := doRequest(t, h, http.MethodDelete, "/tasks/999", "")

	// Assert
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// ---------- trace id ----------

func TestNewRouter_SuccessfulRequest_SetsTraceIDHeader(t *testing.T) {
	// Arrange
	h := newTestRouter(t, generousLimits)

	// Act
	rec := doRequest(t, h, http.MethodGet, "/tasks", "")

	// Assert
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get(middleware.TraceIDHeader), "successful responses should also carry X-Trace-Id")
}

func TestNewRouter_TaskNotFound_ReturnsErrorWithTraceIDMatchingHeader(t *testing.T) {
	// Arrange
	h := newTestRouter(t, generousLimits)

	// Act
	rec := doRequest(t, h, http.MethodPut, "/tasks/999", `{"name":"x"}`)

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

// ---------- Routing and middleware ----------

func TestNewRouter_WrongMethod_Returns405WithAllowHeader(t *testing.T) {
	tests := []struct {
		method, path, wantAllow string
	}{
		{http.MethodPatch, "/tasks", "GET, HEAD, POST"}, // ServeMux sorts methods alphabetically
		{http.MethodGet, "/tasks/1", "DELETE, PUT"},
	}
	for _, tc := range tests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			// Arrange
			h := newTestRouter(t, generousLimits)

			// Act
			rec := doRequest(t, h, tc.method, tc.path, "")

			// Assert
			require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
			assert.Equal(t, tc.wantAllow, rec.Header().Get("Allow"))
			assert.Equal(t, "METHOD_NOT_ALLOWED", errorCode(t, rec.Body.Bytes()))
		})
	}
}

func TestNewRouter_UnknownPath_Returns404RouteNotFound(t *testing.T) {
	// Arrange
	h := newTestRouter(t, generousLimits)

	// Act
	rec := doRequest(t, h, http.MethodGet, "/nope", "")

	// Assert
	require.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "ROUTE_NOT_FOUND", errorCode(t, rec.Body.Bytes()))
}

func TestNewRouter_BurstExceeded_Returns429(t *testing.T) {
	// Arrange
	h := newTestRouter(t, Config{RateLimitRPS: 0.001, RateLimitBurst: 1})
	first := doRequest(t, h, http.MethodGet, "/tasks", "")
	require.Equal(t, http.StatusOK, first.Code, "arrange: first request")

	// Act
	rec := doRequest(t, h, http.MethodGet, "/tasks", "")

	// Assert
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}
