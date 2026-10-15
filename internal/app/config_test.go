package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_NoEnvVars_ReturnsDefaults(t *testing.T) {
	// Arrange
	t.Setenv("PORT", "")
	t.Setenv("RATE_LIMIT_RPS", "")
	t.Setenv("RATE_LIMIT_BURST", "")

	// Act
	cfg, err := LoadConfig()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, Config{Port: "8080", RateLimitRPS: 10, RateLimitBurst: 20}, cfg)
}

func TestLoadConfig_ValidEnvVars_OverridesDefaults(t *testing.T) {
	// Arrange
	t.Setenv("PORT", "9090")
	t.Setenv("RATE_LIMIT_RPS", "2.5")
	t.Setenv("RATE_LIMIT_BURST", "5")

	// Act
	cfg, err := LoadConfig()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, Config{Port: "9090", RateLimitRPS: 2.5, RateLimitBurst: 5}, cfg)
}

func TestLoadConfig_InvalidEnvVar_ReturnsError(t *testing.T) {
	tests := []struct{ key, value string }{
		{"RATE_LIMIT_RPS", "abc"},
		{"RATE_LIMIT_RPS", "0"},
		{"RATE_LIMIT_RPS", "NaN"},
		{"RATE_LIMIT_BURST", "0"},
		{"RATE_LIMIT_BURST", "1.5"},
	}
	for _, tc := range tests {
		t.Run(tc.key+"="+tc.value, func(t *testing.T) {
			// Arrange
			t.Setenv(tc.key, tc.value)

			// Act
			_, err := LoadConfig()

			// Assert
			assert.Error(t, err)
		})
	}
}
