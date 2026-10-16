package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"taskapi/internal/model"
	"taskapi/internal/trace"
)

// errorResponse is the common error format:
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

// WriteJSON writes v as a JSON response.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes err in the common JSON format, with the HTTP status taken from statusByCode.
// Anything that is not a *model.AppError, or whose code has no status, becomes a 500,
// and its original message is not returned, so no internal details leak to the client.
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
