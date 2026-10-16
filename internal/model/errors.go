package model

// FieldError is a validation error for one field, i.e. one entry of the details array in an error response.
// Reason is the validator tag name (required, max, oneof, ...) and Param is the tag's argument.
type FieldError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
	Param  string `json:"param,omitempty"`
}

// AppError is the error type.
// Code is a stable, public string for programs to act on.
// Message is a human-readable description.
type AppError struct {
	Code    string
	Message string
	Details []FieldError
}

func (e *AppError) Error() string {
	return e.Code + ": " + e.Message
}

// WithDetails returns a copy with field errors attached.
func (e *AppError) WithDetails(details []FieldError) *AppError {
	cp := *e
	cp.Details = details
	return &cp
}

// Business errors returned by the service and repository layers.
var (
	ErrValidationFailed = &AppError{Code: "VALIDATION_FAILED", Message: "request validation failed"}
	ErrTaskNotFound     = &AppError{Code: "TASK_NOT_FOUND", Message: "task not found"}
)
