package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"taskapi/internal/model"
	"taskapi/internal/trace"
)

type decoded struct {
	Error struct {
		Code    string             `json:"code"`
		Message string             `json:"message"`
		Details []model.FieldError `json:"details"`
		TraceID string             `json:"trace_id"`
	} `json:"error"`
}

// writeError 帶著 trace id "abc123" 呼叫 WriteError，回傳錄下的回應與解析後的 body。
func writeError(t *testing.T, err error) (*httptest.ResponseRecorder, decoded) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(trace.NewContext(req.Context(), "abc123"))
	rec := httptest.NewRecorder()
	WriteError(rec, req, err)

	var body decoded
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "body: %s", rec.Body.String())
	return rec, body
}

func TestWriteError_AppErrorWithDetails_WritesMappedStatusDetailsAndTraceID(t *testing.T) {
	// Arrange
	details := []model.FieldError{{Field: "name", Reason: "required"}}
	err := model.ErrValidationFailed.WithDetails(details)

	// Act
	rec, body := writeError(t, err)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "VALIDATION_FAILED", body.Error.Code)
	assert.Equal(t, details, body.Error.Details)
	assert.Equal(t, "abc123", body.Error.TraceID)
}

func TestWriteError_NonAppError_Returns500GenericMessage(t *testing.T) {
	// Arrange
	err := errors.New("db password is hunter2")

	// Act
	rec, body := writeError(t, err)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "INTERNAL_ERROR", body.Error.Code)
	assert.Equal(t, "internal server error", body.Error.Message)
}

func TestWriteError_UnmappedCode_Returns500GenericMessage(t *testing.T) {
	// Arrange
	err := &model.AppError{Code: "SOMETHING_NEW", Message: "secret detail"}

	// Act
	rec, body := writeError(t, err)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "INTERNAL_ERROR", body.Error.Code)
	assert.Equal(t, "internal server error", body.Error.Message)
}
