package service

import (
	"context"
	"errors"
	"log/slog"

	"taskapi/internal/model"
	"taskapi/internal/repository"
)

//go:generate go tool mockgen -typed -source=task_service.go -destination=mocks/mock_task_service.go -package=mocks

// TaskService is the interface the controller depends on; controller tests replace it with a mock.
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

// Update replaces the whole task: every field of t overwrites the stored value.
func (s *taskService) Update(ctx context.Context, id int, t model.Task) (model.Task, error) {
	updated, err := s.repo.Update(ctx, id, t)
	if err != nil {
		return model.Task{}, err
	}
	s.logger.InfoContext(ctx, "task updated", "task_id", id)
	return updated, nil
}

// Delete is idempotent: its goal is "make sure this task does not exist", so a task that is already absent (never created or already deleted) is a success.
// The repository still reports ErrTaskNotFound faithfully; this layer decides, by business meaning, to treat it as success.
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
