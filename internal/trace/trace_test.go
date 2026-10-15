package trace

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidID_AcceptableID_ReturnsTrue(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"32-char hex", "4bf92f3577b34da6a3ce929d0e0e4736"},
		{"uuid", "3fa85f64-5717-4562-b3fc-2c963f66afa6"},
		{"letters digits dot underscore", "Req_2026.10.13-abc"},
		{"exactly max length", strings.Repeat("a", maxIDLength)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			id := tc.id

			// Act
			ok := IsValidID(id)

			// Assert
			assert.True(t, ok)
		})
	}
}

func TestIsValidID_UnacceptableID_ReturnsFalse(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"empty", ""},
		{"one over max length", strings.Repeat("a", maxIDLength+1)},
		{"contains space", "abc def"},
		{"contains newline", "abc\ninjected log line"},
		{"contains colon", "abc:def"},
		{"non-ASCII", "追蹤123"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			id := tc.id

			// Act
			ok := IsValidID(id)

			// Assert
			assert.False(t, ok)
		})
	}
}

func TestNewID_CalledTwice_ReturnsDistinctValidHexIDs(t *testing.T) {
	// Arrange
	idPattern := fmt.Sprintf(`^[0-9a-f]{%d}$`, IDLength)

	// Act
	a, b := NewID(), NewID()

	// Assert
	assert.Regexp(t, idPattern, a)
	assert.Regexp(t, idPattern, b)
	assert.NotEqual(t, a, b)
	assert.True(t, IsValidID(a), "generated ids must pass our own validation")
}
