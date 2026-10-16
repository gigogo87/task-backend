package repository

import (
	"context"

	"taskapi/internal/model"
)

//go:generate go tool mockgen -typed -source=task_repository.go -destination=mocks/mock_task_repository.go -package=mocks

// TaskRepository is the storage interface.
type TaskRepository interface {
	List(ctx context.Context) ([]model.Task, error)
	Create(ctx context.Context, t model.Task) (model.Task, error)
	Update(ctx context.Context, id int, t model.Task) (model.Task, error)
	Delete(ctx context.Context, id int) error
}
