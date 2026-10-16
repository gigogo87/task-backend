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

// DependencyProvider is the only place in the program that constructs objects (dependency injection).
type DependencyProvider struct {
	Config    Config
	Logger    *slog.Logger
	Validator *validator.Validate

	TaskRepository repository.TaskRepository
	TaskService    service.TaskService
	TaskController *controller.TaskController
	RateLimiter    *middleware.RateLimiter
	Router         http.Handler // entry point of the whole HTTP API: routes + middleware (see router.go)
}

// NewDependencyProvider builds every component bottom-up: repository → service → controller → middleware → router.
// logOut is where logs are written.
func NewDependencyProvider(cfg Config, logOut io.Writer) *DependencyProvider {
	dp := &DependencyProvider{Config: cfg}

	// Shared building blocks
	dp.Logger = slog.New(trace.NewLogHandler(slog.NewJSONHandler(logOut, nil)))
	dp.Validator = dto.NewValidator()

	// Layers, bottom-up
	dp.TaskRepository = repository.NewMemoryTaskRepository()
	dp.TaskService = service.NewTaskService(dp.TaskRepository, dp.Logger)
	dp.TaskController = controller.NewTaskController(dp.TaskService, dp.Validator, dp.Logger)
	dp.RateLimiter = middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)
	dp.Router = newRouter(dp.TaskController, dp.RateLimiter)

	return dp
}

// Close releases resources that need closing.
func (dp *DependencyProvider) Close() {
	dp.RateLimiter.Close()
}
