package repository

import (
	"context"
	"sort"
	"sync"

	"taskapi/internal/model"
)

// Compile-time check that MemoryTaskRepository implements TaskRepository.
var _ TaskRepository = (*MemoryTaskRepository)(nil)

// MemoryTaskRepository keeps tasks in an in-memory map (data is lost when the program restarts).
type MemoryTaskRepository struct {
	mu     sync.RWMutex // maps are not goroutine-safe
	tasks  map[int]model.Task
	nextID int
}

// firstTaskID is the ID of the first task; each new task gets the next number.
const firstTaskID = 1

// NewMemoryTaskRepository creates an empty MemoryTaskRepository.
func NewMemoryTaskRepository() *MemoryTaskRepository {
	return &MemoryTaskRepository{
		tasks:  make(map[int]model.Task),
		nextID: firstTaskID,
	}
}

// List returns all tasks sorted by ID.
func (r *MemoryTaskRepository) List(_ context.Context) ([]model.Task, error) {
	r.mu.RLock() // read lock: multiple readers may read at the same time
	defer r.mu.RUnlock()

	result := make([]model.Task, 0, len(r.tasks))
	for _, t := range r.tasks {
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// Create adds a task; the storage layer assigns the ID.
func (r *MemoryTaskRepository) Create(_ context.Context, t model.Task) (model.Task, error) {
	r.mu.Lock() // write lock: only one writer at a time
	defer r.mu.Unlock()

	t.ID = r.nextID
	r.nextID++
	r.tasks[t.ID] = t
	return t, nil
}

// Update replaces the whole task with the given ID.
func (r *MemoryTaskRepository) Update(_ context.Context, id int, t model.Task) (model.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tasks[id]; !ok {
		return model.Task{}, model.ErrTaskNotFound
	}
	t.ID = id
	r.tasks[id] = t
	return t, nil
}

// Delete removes the task with the given ID.
func (r *MemoryTaskRepository) Delete(_ context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tasks[id]; !ok {
		return model.ErrTaskNotFound
	}
	delete(r.tasks, id)
	return nil
}
