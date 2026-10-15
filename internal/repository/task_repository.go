// Package repository 是儲存層：只負責把資料存進、取出儲存體（記憶體、DB、cache...），不含商業邏輯。
package repository

import (
	"context"

	"taskapi/internal/model"
)

//go:generate go tool mockgen -typed -source=task_repository.go -destination=mocks/mock_task_repository.go -package=mocks

// TaskRepository 是儲存層的介面。
// service 只依賴這個介面，不在乎底下是記憶體、MySQL 還是 Redis。
// 之後要換成資料庫，只要再寫一個實作這四個方法的 struct，並在 dependency_provider.go 換掉即可。
type TaskRepository interface {
	List(ctx context.Context) ([]model.Task, error)
	Create(ctx context.Context, t model.Task) (model.Task, error)
	Update(ctx context.Context, id int, t model.Task) (model.Task, error)
	Delete(ctx context.Context, id int) error
}
