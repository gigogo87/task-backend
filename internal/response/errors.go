package response

import (
	"net/http"

	"taskapi/internal/model"
)

// 只會在 HTTP 層發生的錯誤：由 controller、router、middleware 回傳，service / repository 不會用到。
var (
	ErrInvalidJSON      = &model.AppError{Code: "INVALID_JSON", Message: "invalid JSON body"}
	ErrInvalidID        = &model.AppError{Code: "INVALID_ID", Message: "id must be a positive integer"}
	ErrRouteNotFound    = &model.AppError{Code: "ROUTE_NOT_FOUND", Message: "route not found"}
	ErrMethodNotAllowed = &model.AppError{Code: "METHOD_NOT_ALLOWED", Message: "method not allowed"}
	ErrPayloadTooLarge  = &model.AppError{Code: "PAYLOAD_TOO_LARGE", Message: "request body too large"}
	ErrRateLimited      = &model.AppError{Code: "RATE_LIMITED", Message: "rate limit exceeded"}
	ErrInternal         = &model.AppError{Code: "INTERNAL_ERROR", Message: "internal server error"}
)

// statusByCode 是錯誤碼對應 HTTP status 的唯一一張表。
// 新增錯誤碼時記得加在這裡，否則會被當成 500（TestEveryErrorHasStatus 會抓到漏加的）。
var statusByCode = map[string]int{
	model.ErrValidationFailed.Code: http.StatusBadRequest,
	model.ErrTaskNotFound.Code:     http.StatusNotFound,

	ErrInvalidJSON.Code:      http.StatusBadRequest,
	ErrInvalidID.Code:        http.StatusBadRequest,
	ErrRouteNotFound.Code:    http.StatusNotFound,
	ErrMethodNotAllowed.Code: http.StatusMethodNotAllowed,
	ErrPayloadTooLarge.Code:  http.StatusRequestEntityTooLarge,
	ErrRateLimited.Code:      http.StatusTooManyRequests,
	ErrInternal.Code:         http.StatusInternalServerError,
}
