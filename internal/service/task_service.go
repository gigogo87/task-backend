// Package service 是商業邏輯層：決定「要做什麼」，透過 repository 存取資料，完全不知道 HTTP 的存在。
package service

import (
	"context"
	"errors"
	"log/slog"

	"taskapi/internal/model"
	"taskapi/internal/repository"
)

//go:generate go tool mockgen -typed -source=task_service.go -destination=mocks/mock_task_service.go -package=mocks

// TaskService 是 controller 依賴的介面，測試 controller 時可以換成 mock。
type TaskService interface {
	List(ctx context.Context) ([]model.Task, error)
	Create(ctx context.Context, t model.Task) (model.Task, error)
	Update(ctx context.Context, id int, t model.Task) (model.Task, error)
	Delete(ctx context.Context, id int) error
}

type taskService struct {
	repo   repository.TaskRepository
	logger *slog.Logger
}

// NewTaskService 透過參數注入所有依賴（依賴注入），不在裡面自己 new 任何東西。
func NewTaskService(repo repository.TaskRepository, logger *slog.Logger) TaskService {
	return &taskService{repo: repo, logger: logger}
}

func (s *taskService) List(ctx context.Context) ([]model.Task, error) {
	s.logger.InfoContext(ctx, "task listed")
	return s.repo.List(ctx)
}

func (s *taskService) Create(ctx context.Context, t model.Task) (model.Task, error) {
	created, err := s.repo.Create(ctx, t)
	if err != nil {
		return model.Task{}, err
	}
	s.logger.InfoContext(ctx, "task created", "task_id", created.ID)
	return created, nil
}

// Update 是整筆覆蓋：t 的所有欄位都會取代原本的值。
func (s *taskService) Update(ctx context.Context, id int, t model.Task) (model.Task, error) {
	updated, err := s.repo.Update(ctx, id, t)
	if err != nil {
		return model.Task{}, err
	}
	s.logger.InfoContext(ctx, "task updated", "task_id", id)
	return updated, nil
}

// Delete 是冪等的：目的是「讓這筆 task 不存在」，所以 task 本來就不在（從沒建立過或已被刪除）也算成功。
// 這樣 client 在逾時後重試，不會因為第一次其實已經刪除成功而收到錯誤。
// repository 仍如實回報 ErrTaskNotFound，由這一層依業務語意決定視為成功；
// 對照 Update：更新不存在的 task 仍然是錯誤。
func (s *taskService) Delete(ctx context.Context, id int) error {
	err := s.repo.Delete(ctx, id)
	if errors.Is(err, model.ErrTaskNotFound) {
		s.logger.InfoContext(ctx, "task already absent", "task_id", id)
		return nil
	}
	if err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "task deleted", "task_id", id)
	return nil
}
