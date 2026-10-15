package app

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			h := newTestHandler(t, generousLimits)

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
	h := newTestHandler(t, generousLimits)

	// Act
	rec := doRequest(t, h, http.MethodGet, "/nope", "")

	// Assert
	require.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "ROUTE_NOT_FOUND", errorCode(t, rec.Body.Bytes()))
}
