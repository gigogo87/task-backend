package response

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"taskapi/internal/model"
)

// Every defined error needs an HTTP status; a missing one would become a 500.
func TestStatusByCode_AllDefinedErrors_HaveHTTPStatus(t *testing.T) {
	// Arrange
	all := []*model.AppError{
		model.ErrValidationFailed, model.ErrTaskNotFound,
		ErrInvalidJSON, ErrInvalidID, ErrRouteNotFound, ErrMethodNotAllowed,
		ErrPayloadTooLarge, ErrRateLimited, ErrInternal,
	}

	for _, e := range all {
		// Act
		_, ok := statusByCode[e.Code]

		// Assert
		assert.True(t, ok, "%s has no HTTP status in statusByCode", e.Code)
	}
}
