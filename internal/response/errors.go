package response

import (
	"net/http"

	"taskapi/internal/model"
)

// Errors that only exist at the HTTP level: returned by controllers, the router and middleware, never by the service or repository.
var (
	ErrInvalidJSON      = &model.AppError{Code: "INVALID_JSON", Message: "invalid JSON body"}
	ErrInvalidID        = &model.AppError{Code: "INVALID_ID", Message: "id must be a positive integer"}
	ErrRouteNotFound    = &model.AppError{Code: "ROUTE_NOT_FOUND", Message: "route not found"}
	ErrMethodNotAllowed = &model.AppError{Code: "METHOD_NOT_ALLOWED", Message: "method not allowed"}
	ErrPayloadTooLarge  = &model.AppError{Code: "PAYLOAD_TOO_LARGE", Message: "request body too large"}
	ErrRateLimited      = &model.AppError{Code: "RATE_LIMITED", Message: "rate limit exceeded"}
	ErrInternal         = &model.AppError{Code: "INTERNAL_ERROR", Message: "internal server error"}
)

// statusByCode is the single table mapping error codes to HTTP statuses.
// Add every new error code here, or it becomes a 500 (TestStatusByCode_AllDefinedErrors_HaveHTTPStatus catches omissions).
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
