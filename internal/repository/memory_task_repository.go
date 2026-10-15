package repository

import (
	"context"
	"sort"
	"sync"

	"taskapi/internal/model"
)

// 編譯期檢查：確保 MemoryTaskRepository 有實作 TaskRepository，少一個方法就會編譯失敗。
var _ TaskRepository = (*MemoryTaskRepository)(nil)

// MemoryTaskRepository 用 map 把資料存在記憶體裡（程式重啟資料就會消失）。
type MemoryTaskRepository struct {
	mu     sync.RWMutex // HTTP server 會同時處理多個請求，map 不是並發安全的，所以要加鎖
	tasks  map[int]model.Task
	nextID int
}

// firstTaskID 是第一筆任務的 ID；之後每新增一筆就加 1（概念同 DB 的 auto increment）。
const firstTaskID = 1

// NewMemoryTaskRepository 建立一個空的 MemoryTaskRepository。Go 慣例用 NewXxx 當建構函式。
func NewMemoryTaskRepository() *MemoryTaskRepository {
	return &MemoryTaskRepository{
		tasks:  make(map[int]model.Task),
		nextID: firstTaskID,
	}
}

// List 回傳所有任務，依 ID 排序（map 的順序是隨機的，排序後結果才穩定、好測試）。
func (r *MemoryTaskRepository) List(_ context.Context) ([]model.Task, error) {
	r.mu.RLock() // 讀取用讀鎖，多個讀者可以同時讀
	defer r.mu.RUnlock()

	result := make([]model.Task, 0, len(r.tasks))
	for _, t := range r.tasks {
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// Create 新增任務，ID 由儲存層產生（概念同 DB 的 auto increment）。
func (r *MemoryTaskRepository) Create(_ context.Context, t model.Task) (model.Task, error) {
	r.mu.Lock() // 寫入用寫鎖，同一時間只有一個人能寫
	defer r.mu.Unlock()

	t.ID = r.nextID
	r.nextID++
	r.tasks[t.ID] = t
	return t, nil
}

// Update 以整筆覆蓋的方式更新指定 ID 的任務。
func (r *MemoryTaskRepository) Update(_ context.Context, id int, t model.Task) (model.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tasks[id]; !ok {
		return model.Task{}, model.ErrTaskNotFound
	}
	t.ID = id // 以參數的 id 為準，忽略 t 裡的 id
	r.tasks[id] = t
	return t, nil
}

// Delete 刪除指定 ID 的任務。
func (r *MemoryTaskRepository) Delete(_ context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tasks[id]; !ok {
		return model.ErrTaskNotFound
	}
	delete(r.tasks, id)
	return nil
}
