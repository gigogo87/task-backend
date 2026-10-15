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

// 整合測試：透過 DependencyProvider 組出完整的 router（middleware + controller + service + repository），跑一遍真實流程。

// newTestRouter 建立完整組裝好的 router；測試結束時釋放 provider 的資源。
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

// createTask 透過 API 新增一筆任務，回傳 server 指派的 id。
// 整合測試從回應取得 id，而不是假設 id 從幾開始，就像真正的 client 一樣。
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

// ---------- 正常流程 ----------

func TestNewRouter_CreateUpdateDeleteThenList_ReturnsRemainingTask(t *testing.T) {
	// Arrange
	h := newTestRouter(t, generousLimits)
	milkID := createTask(t, h, "買牛奶")
	homeworkID := createTask(t, h, "寫作業")
	updated := doRequest(t, h, http.MethodPut, taskPath(milkID), `{"name":"買牛奶","status":1}`)
	require.Equal(t, http.StatusOK, updated.Code, "arrange: update, body: %s", updated.Body.String())
	deleted := doRequest(t, h, http.MethodDelete, taskPath(homeworkID), "")
	require.Equal(t, http.StatusNoContent, deleted.Code, "arrange: delete")

	// Act
	rec := doRequest(t, h, http.MethodGet, "/tasks", "")

	// Assert
	want := fmt.Sprintf(`{"result":[{"id":%d,"name":"買牛奶","status":1}]}`, milkID)
	assert.JSONEq(t, want, rec.Body.String())
}

// DELETE 是冪等的：重試（例如第一次的回應在網路上遺失）也要得到成功。
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

// ---------- 路由與 middleware ----------

func TestNewRouter_WrongMethod_Returns405WithAllowHeader(t *testing.T) {
	tests := []struct {
		method, path, wantAllow string
	}{
		{http.MethodPatch, "/tasks", "GET, HEAD, POST"}, // ServeMux 依字母排序
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
