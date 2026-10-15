// Package app 負責把所有元件組裝起來（composition root）。
package app

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"

	"taskapi/internal/controller"
	"taskapi/internal/dto"
	"taskapi/internal/middleware"
	"taskapi/internal/repository"
	"taskapi/internal/service"
	"taskapi/internal/trace"
)

// DependencyProvider 是整個程式唯一 new 物件的地方（依賴注入）。
// 規則：每個 NewXxx 都只透過參數拿到依賴，不在裡面自己 new 其他層的東西。
// 好處是：要換 DB 只改這裡一行；測試時可以直接把假的實作傳給 NewXxx，不需要經過 DependencyProvider。
type DependencyProvider struct {
	Config    Config
	Logger    *slog.Logger
	Validator *validator.Validate

	TaskRepository repository.TaskRepository
	TaskService    service.TaskService
	TaskController *controller.TaskController
	RateLimiter    *middleware.RateLimiter
	Router         http.Handler // 整個 HTTP API 的入口：路由 + middleware（見 router.go）
}

// NewDependencyProvider 依「由下往上」的順序建立所有元件：repository → service → controller → middleware → router。
// logOut 是 log 要寫到哪裡（正式環境是 os.Stdout，測試可以傳 io.Discard）。
func NewDependencyProvider(cfg Config, logOut io.Writer) *DependencyProvider {
	dp := &DependencyProvider{Config: cfg}

	// 共用的基礎元件
	dp.Logger = slog.New(trace.NewLogHandler(slog.NewJSONHandler(logOut, nil)))
	dp.Validator = dto.NewValidator()

	// 各層，由下往上
	dp.TaskRepository = repository.NewMemoryTaskRepository()
	dp.TaskService = service.NewTaskService(dp.TaskRepository, dp.Logger)
	dp.TaskController = controller.NewTaskController(dp.TaskService, dp.Validator, dp.Logger)
	dp.RateLimiter = middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)
	dp.Router = newRouter(dp.TaskController, dp.RateLimiter)

	return dp
}

// Close 釋放需要關閉的資源（目前是限流器的背景 goroutine；之後的 DB 連線池也在這裡關）。
func (dp *DependencyProvider) Close() {
	dp.RateLimiter.Close()
}
