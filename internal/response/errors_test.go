package response

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"taskapi/internal/model"
)

// 每個定義好的錯誤都要有對應的 HTTP status，漏加就會被當成 500。
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
