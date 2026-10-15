// Package response 放 controller、router 與 middleware 共用的 HTTP 回應工具，確保所有回應格式一致。
package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"taskapi/internal/model"
	"taskapi/internal/trace"
)

// errorResponse 是統一的錯誤格式：
//
//	{"error": {"code": "...", "message": "...", "details": [...], "trace_id": "..."}}
type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string             `json:"code"`
	Message string             `json:"message"`
	Details []model.FieldError `json:"details,omitempty"`
	TraceID string             `json:"trace_id,omitempty"`
}

// WriteJSON 把 v 編碼成 JSON 回應。
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError 把錯誤轉成統一格式的 JSON，HTTP status 由 statusByCode 決定。
// 不是 *model.AppError、或錯誤碼沒有對應 status 的錯誤，一律當成 500，
// 且不把原始訊息回給 client，避免洩漏內部資訊。
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	appErr := ErrInternal
	var target *model.AppError
	if errors.As(err, &target) {
		if _, ok := statusByCode[target.Code]; ok {
			appErr = target
		}
	}

	WriteJSON(w, statusByCode[appErr.Code], errorResponse{Error: errorBody{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		TraceID: trace.FromContext(r.Context()),
	}})
}
