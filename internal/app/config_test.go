package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_NoEnvVars_ReturnsDefaults(t *testing.T) {
	// Arrange
	t.Setenv(envPort, "")
	t.Setenv(envRateLimitRPS, "")
	t.Setenv(envRateLimitBurst, "")

	// Act
	cfg, err := LoadConfig()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, Config{Port: defaultPort, RateLimitRPS: defaultRateLimitRPS, RateLimitBurst: defaultRateLimitBurst}, cfg)
}

func TestLoadConfig_ValidEnvVars_OverridesDefaults(t *testing.T) {
	// Arrange
	t.Setenv(envPort, "9090")
	t.Setenv(envRateLimitRPS, "2.5")
	t.Setenv(envRateLimitBurst, "5")

	// Act
	cfg, err := LoadConfig()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, Config{Port: "9090", RateLimitRPS: 2.5, RateLimitBurst: 5}, cfg)
}

func TestLoadConfig_InvalidEnvVar_ReturnsError(t *testing.T) {
	tests := []struct{ key, value string }{
		{envRateLimitRPS, "abc"},
		{envRateLimitRPS, "0"},
		{envRateLimitRPS, "NaN"},
		{envRateLimitBurst, "0"},
		{envRateLimitBurst, "1.5"},
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
