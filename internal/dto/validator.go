package dto

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"taskapi/internal/model"
)

// NewValidator creates the validator shared by the whole project.
func NewValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	// Report field names from the json tag (name) instead of the Go field name (Name), so clients understand them.
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		if name == "" {
			return f.Name
		}
		return name
	})
	return v
}

// ValidationError turns validator errors into VALIDATION_FAILED with one detail per field.
// Any other error (e.g. passing the wrong type to the validator) is a programming bug and is returned as-is, becoming a 500 upstream.
func ValidationError(err error) error {
	var ves validator.ValidationErrors
	if !errors.As(err, &ves) {
		return err
	}
	details := make([]model.FieldError, 0, len(ves))
	for _, fe := range ves {
		details = append(details, model.FieldError{
			Field:  fe.Field(),
			Reason: fe.Tag(),
			Param:  fe.Param(),
		})
	}
	return model.ErrValidationFailed.WithDetails(details)
}
