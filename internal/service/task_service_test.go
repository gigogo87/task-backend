package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"taskapi/internal/model"
	"taskapi/internal/repository/mocks"
)

// taskServiceFixture 裝著一個測試需要的所有東西：待測對象、它依賴的 mock、共用的 ctx。
// 之後 service 多了依賴（例如 cache、外部 API），只要在這裡加欄位，測試本身不用改簽名。
type taskServiceFixture struct {
	ctx      context.Context
	mockCtrl *gomock.Controller
	repo     *mocks.MockTaskRepository
	svc      TaskService // 待測對象
}

// setUp 在每個測試的 Arrange 開頭呼叫：建立 mock，並把它們注入待測的 service。
// mock 由 mockgen 依 repository.TaskRepository 產生（見 task_repository.go 的 go:generate）。
func setUp(t *testing.T) *taskServiceFixture {
	t.Helper()
	mockCtrl := gomock.NewController(t)
	f := &taskServiceFixture{
		ctx:      context.Background(),
		mockCtrl: mockCtrl,
		repo:     mocks.NewMockTaskRepository(mockCtrl),
	}
	f.svc = NewTaskService(f.repo, slog.New(slog.DiscardHandler))
	t.Cleanup(f.tearDown) // 測試結束一定會執行，包含 t.Fatal 中途結束
	return f
}

// tearDown 驗證所有 EXPECT() 都如預期被呼叫（參數、次數），否則測試失敗。
// 之後若 fixture 有需要釋放的資源，也在這裡處理。
func (f *taskServiceFixture) tearDown() {
	f.mockCtrl.Finish()
}

var errRepository = errors.New("repository failure")

func TestList_RepositorySucceeds_ReturnsRepositoryTasks(t *testing.T) {
	// Arrange
	f := setUp(t)
	tasks := []model.Task{{ID: 1, Name: "x"}}
	f.repo.EXPECT().List(f.ctx).Return(tasks, nil)

	// Act
	got, err := f.svc.List(f.ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, tasks, got)
}

func TestList_RepositoryFails_ReturnsSameError(t *testing.T) {
	// Arrange
	f := setUp(t)
	f.repo.EXPECT().List(f.ctx).Return(nil, errRepository)

	// Act
	_, err := f.svc.List(f.ctx)

	// Assert
	assert.ErrorIs(t, err, errRepository)
}

func TestCreate_ValidTask_ReturnsTaskWithRepositoryID(t *testing.T) {
	// Arrange
	f := setUp(t)
	input := model.Task{Name: "買牛奶"}
	f.repo.EXPECT().Create(f.ctx, input).Return(model.Task{ID: 42, Name: "買牛奶"}, nil)

	// Act
	got, err := f.svc.Create(f.ctx, input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, model.Task{ID: 42, Name: "買牛奶"}, got)
}

func TestCreate_RepositoryFails_ReturnsSameError(t *testing.T) {
	// Arrange
	f := setUp(t)
	f.repo.EXPECT().Create(f.ctx, gomock.Any()).Return(model.Task{}, errRepository)

	// Act
	_, err := f.svc.Create(f.ctx, model.Task{Name: "x"})

	// Assert
	assert.ErrorIs(t, err, errRepository)
}

func TestUpdate_ValidTask_PassesIDAndTaskToRepository(t *testing.T) {
	// Arrange
	f := setUp(t)
	task := model.Task{Name: "new", Status: model.StatusComplete}
	f.repo.EXPECT().Update(f.ctx, 7, task).Return(model.Task{ID: 7, Name: "new", Status: model.StatusComplete}, nil)

	// Act
	got, err := f.svc.Update(f.ctx, 7, task)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, model.Task{ID: 7, Name: "new", Status: model.StatusComplete}, got)
	// id 與 task 是否正確傳給 repository，由 EXPECT 的參數比對驗證
}

// repository 回傳的錯誤要原封不動往上傳，controller 才能對應到正確的錯誤碼。
func TestUpdate_TaskNotFound_ReturnsErrTaskNotFound(t *testing.T) {
	// Arrange
	f := setUp(t)
	f.repo.EXPECT().Update(f.ctx, 1, gomock.Any()).Return(model.Task{}, model.ErrTaskNotFound)

	// Act
	_, err := f.svc.Update(f.ctx, 1, model.Task{Name: "x"})

	// Assert
	assert.ErrorIs(t, err, model.ErrTaskNotFound)
}

func TestDelete_ExistingTask_PassesIDToRepository(t *testing.T) {
	// Arrange
	f := setUp(t)
	f.repo.EXPECT().Delete(f.ctx, 5).Return(nil)

	// Act
	err := f.svc.Delete(f.ctx, 5)

	// Assert
	assert.NoError(t, err)
	// id 是否正確傳給 repository，由 EXPECT 的參數比對驗證
}

// Delete 是冪等的：task 本來就不在（從沒建立過或已被刪除）也算成功。
func TestDelete_TaskNotFound_ReturnsNil(t *testing.T) {
	// Arrange
	f := setUp(t)
	f.repo.EXPECT().Delete(f.ctx, 1).Return(model.ErrTaskNotFound)

	// Act
	err := f.svc.Delete(f.ctx, 1)

	// Assert
	assert.NoError(t, err)
}

// 只有「找不到」被視為成功，其他 repository 錯誤仍要原樣往上傳。
func TestDelete_RepositoryFails_ReturnsSameError(t *testing.T) {
	// Arrange
	f := setUp(t)
	f.repo.EXPECT().Delete(f.ctx, 1).Return(errRepository)

	// Act
	err := f.svc.Delete(f.ctx, 1)

	// Assert
	assert.ErrorIs(t, err, errRepository)
}
