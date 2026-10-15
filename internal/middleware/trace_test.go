package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"taskapi/internal/trace"
)

// echoTraceIDHandler 把 context 裡的 trace id 寫進 body，用來檢查 Trace 有沒有把 id 放進 context。
var echoTraceIDHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(trace.FromContext(r.Context())))
})

func TestTrace_NoTraceIDHeader_GeneratesTraceIDInHeaderAndContext(t *testing.T) {
	// Arrange
	h := Trace(echoTraceIDHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	// Act
	h.ServeHTTP(rec, req)

	// Assert
	id := rec.Header().Get(TraceIDHeader)
	require.Len(t, id, trace.IDLength)
	assert.Equal(t, id, rec.Body.String(), "trace id in context should match header")
}

func TestTrace_ValidTraceIDHeader_ReusesItInHeaderAndContext(t *testing.T) {
	// Arrange
	h := Trace(echoTraceIDHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(TraceIDHeader, "3fa85f64-5717-4562-b3fc-2c963f66afa6")
	rec := httptest.NewRecorder()

	// Act
	h.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, "3fa85f64-5717-4562-b3fc-2c963f66afa6", rec.Header().Get(TraceIDHeader))
	assert.Equal(t, "3fa85f64-5717-4562-b3fc-2c963f66afa6", rec.Body.String())
}

func TestTrace_InvalidTraceIDHeader_GeneratesNewTraceID(t *testing.T) {
	// Arrange
	h := Trace(echoTraceIDHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(TraceIDHeader, "bad id with spaces")
	rec := httptest.NewRecorder()

	// Act
	h.ServeHTTP(rec, req)

	// Assert
	assert.Regexp(t, fmt.Sprintf(`^[0-9a-f]{%d}$`, trace.IDLength), rec.Header().Get(TraceIDHeader))
}
