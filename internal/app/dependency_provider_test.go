package app

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 新增元件時若忘了在 NewDependencyProvider 裡建立，這個測試會抓到。
func TestNewDependencyProvider_ValidConfig_WiresAllComponents(t *testing.T) {
	// Arrange
	cfg := Config{Port: defaultPort, RateLimitRPS: defaultRateLimitRPS, RateLimitBurst: defaultRateLimitBurst}

	// Act
	dp := NewDependencyProvider(cfg, io.Discard)
	t.Cleanup(dp.Close)

	// Assert
	assert.Equal(t, cfg, dp.Config)
	assert.NotNil(t, dp.Logger)
	assert.NotNil(t, dp.Validator)
	assert.NotNil(t, dp.TaskRepository)
	assert.NotNil(t, dp.TaskService)
	assert.NotNil(t, dp.TaskController)
	assert.NotNil(t, dp.RateLimiter)
	assert.NotNil(t, dp.Router)
}

// main 用 defer dp.Close()；之後若其他關閉路徑也呼叫 Close，不能因此出錯。
func TestClose_CalledTwice_DoesNotPanic(t *testing.T) {
	// Arrange
	dp := NewDependencyProvider(generousLimits, io.Discard)

	// Act
	closeTwice := func() {
		dp.Close()
		dp.Close()
	}

	// Assert
	assert.NotPanics(t, closeTwice)
}
