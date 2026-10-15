package repository

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"taskapi/internal/model"
)

func TestCreate_NewTasks_AssignsSequentialIDs(t *testing.T) {
	// Arrange
	ctx := context.Background()
	r := NewMemoryTaskRepository()

	// Act
	first, err1 := r.Create(ctx, model.Task{Name: "買牛奶"})
	second, err2 := r.Create(ctx, model.Task{Name: "寫作業"})

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, firstTaskID, first.ID)
	assert.Equal(t, firstTaskID+1, second.ID)
}

func TestList_MultipleTasks_ReturnsSortedByID(t *testing.T) {
	// Arrange
	ctx := context.Background()
	r := NewMemoryTaskRepository()
	for _, name := range []string{"a", "b", "c"} {
		_, _ = r.Create(ctx, model.Task{Name: name})
	}

	// Act
	got, err := r.List(ctx)

	// Assert
	require.NoError(t, err)
	want := []model.Task{
		{ID: firstTaskID, Name: "a"},
		{ID: firstTaskID + 1, Name: "b"},
		{ID: firstTaskID + 2, Name: "c"},
	}
	assert.Equal(t, want, got)
}

func TestList_NoTasks_ReturnsEmptyNonNilSlice(t *testing.T) {
	// Arrange
	r := NewMemoryTaskRepository()

	// Act
	got, err := r.List(context.Background())

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, got, "nil slice would encode as null instead of []")
	assert.Empty(t, got)
}

func TestUpdate_ExistingTask_ReplacesFieldsAndKeepsID(t *testing.T) {
	// Arrange
	ctx := context.Background()
	r := NewMemoryTaskRepository()
	created, _ := r.Create(ctx, model.Task{Name: "old"})

	// Act
	updated, err := r.Update(ctx, created.ID, model.Task{ID: 999, Name: "new", Status: model.StatusComplete})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, model.Task{ID: created.ID, Name: "new", Status: model.StatusComplete}, updated)
}

func TestUpdate_NonExistentID_ReturnsErrTaskNotFound(t *testing.T) {
	// Arrange
	r := NewMemoryTaskRepository()

	// Act
	_, err := r.Update(context.Background(), 999, model.Task{Name: "x"})

	// Assert
	assert.ErrorIs(t, err, model.ErrTaskNotFound)
}

func TestDelete_ExistingTask_RemovesTask(t *testing.T) {
	// Arrange
	ctx := context.Background()
	r := NewMemoryTaskRepository()
	created, _ := r.Create(ctx, model.Task{Name: "x"})

	// Act
	err := r.Delete(ctx, created.ID)

	// Assert
	require.NoError(t, err)
	got, _ := r.List(ctx)
	assert.Empty(t, got)
}

func TestDelete_NonExistentID_ReturnsErrTaskNotFound(t *testing.T) {
	// Arrange
	r := NewMemoryTaskRepository()

	// Act
	err := r.Delete(context.Background(), 999)

	// Assert
	assert.ErrorIs(t, err, model.ErrTaskNotFound)
}

// 配合 `go test -race` 可抓出沒加鎖的問題。
func TestCreate_ConcurrentCalls_StoresAllTasks(t *testing.T) {
	// Arrange
	const goroutines = 100
	ctx := context.Background()
	r := NewMemoryTaskRepository()
	var wg sync.WaitGroup

	// Act
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = r.Create(ctx, model.Task{Name: "x"})
		}()
	}
	wg.Wait()

	// Assert
	got, _ := r.List(ctx)
	assert.Len(t, got, goroutines)
}
