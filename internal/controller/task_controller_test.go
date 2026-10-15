package controller

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"taskapi/internal/dto"
	"taskapi/internal/model"
	"taskapi/internal/service/mocks"
)

// errorResponse 對應統一的錯誤格式。
type errorResponse struct {
	Error struct {
		Code    string             `json:"code"`
		Message string             `json:"message"`
		Details []model.FieldError `json:"details"`
		TraceID string             `json:"trace_id"`
	} `json:"error"`
}

// taskControllerFixture 裝著一個測試需要的所有東西：待測對象與它依賴的 mock。
// 之後 controller 多了依賴，只要在這裡加欄位，測試本身不用改簽名。
type taskControllerFixture struct {
	mockCtrl   *gomock.Controller
	service    *mocks.MockTaskService
	controller *TaskController // 待測對象
}

// setUp 在每個測試的 Arrange 開頭呼叫：建立 mock，並把它們注入待測的 controller。
// mock 由 mockgen 依 service.TaskService 產生（見 task_service.go 的 go:generate）。
// 沒有設定 EXPECT() 的測試，只要 service 被呼叫就會失敗，
// 因此「格式錯誤時不應呼叫 service」也會被自動驗證。
func setUp(t *testing.T) *taskControllerFixture {
	t.Helper()
	mockCtrl := gomock.NewController(t)
	f := &taskControllerFixture{
		mockCtrl: mockCtrl,
		service:  mocks.NewMockTaskService(mockCtrl),
	}
	f.controller = NewTaskController(f.service, dto.NewValidator(), slog.New(slog.DiscardHandler))
	t.Cleanup(f.tearDown) // 測試結束一定會執行，包含 t.Fatal 中途結束
	return f
}

// tearDown 驗證所有 EXPECT() 都如預期被呼叫（參數、次數），否則測試失敗。
func (f *taskControllerFixture) tearDown() {
	f.mockCtrl.Finish()
}

// serve 直接呼叫 controller 的方法，不經過路由。
// 路由會把 URL 的 {id} 放進 request，這裡用 SetPathValue 模擬；id 為空字串代表這個路由沒有 {id}。
func serve(t *testing.T, h http.HandlerFunc, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if id != "" {
		req.SetPathValue("id", id)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) errorResponse {
	t.Helper()
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var resp errorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp), "body: %s", rec.Body.String())
	return resp
}

// assertError 檢查 HTTP status 與錯誤碼。
func assertError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	require.Equal(t, wantStatus, rec.Code, "body: %s", rec.Body.String())
	assert.Equal(t, wantCode, decodeError(t, rec).Error.Code)
}

// ---------- List ----------

func TestList_NoTasks_ReturnsEmptyArray(t *testing.T) {
	// Arrange
	f := setUp(t)
	f.service.EXPECT().List(gomock.Any()).Return([]model.Task{}, nil)

	// Act
	rec := serve(t, f.controller.List, "", "")

	// Assert
	require.Equal(t, http.StatusOK, rec.Code)
	// 空的時候要回 [] 而不是 null，前端比較好處理
	assert.JSONEq(t, `{"result":[]}`, rec.Body.String())
}

// 非預期的錯誤要回 500，而且不能把原始錯誤訊息洩漏給 client。
func TestList_UnexpectedServiceError_Returns500WithoutLeakingDetails(t *testing.T) {
	// Arrange
	f := setUp(t)
	f.service.EXPECT().List(gomock.Any()).Return(nil, errors.New("db password is hunter2"))

	// Act
	rec := serve(t, f.controller.List, "", "")

	// Assert
	assertError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
	assert.NotContains(t, rec.Body.String(), "hunter2", "internal error message leaked to client")
}

// ---------- Create ----------

func TestCreate_ValidBody_Returns201AndPassesTaskToService(t *testing.T) {
	// Arrange
	f := setUp(t)
	f.service.EXPECT().
		Create(gomock.Any(), model.Task{Name: "買牛奶", Status: model.StatusIncomplete}).
		Return(model.Task{ID: 1, Name: "買牛奶", Status: model.StatusIncomplete}, nil)

	// Act
	rec := serve(t, f.controller.Create, "", `{"name":"買牛奶"}`)

	// Assert
	require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
	assert.JSONEq(t, `{"result":{"id":1,"name":"買牛奶","status":0}}`, rec.Body.String())
}

func TestCreate_InvalidFields_Returns400WithFieldDetails(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantDetails []model.FieldError
	}{
		{"missing name", `{}`,
			[]model.FieldError{{Field: "name", Reason: "required"}}},
		{"empty name", `{"name":""}`,
			[]model.FieldError{{Field: "name", Reason: "required"}}},
		{"name too long", `{"name":"` + strings.Repeat("a", 101) + `"}`,
			[]model.FieldError{{Field: "name", Reason: "max", Param: "100"}}},
		{"status out of range", `{"name":"x","status":5}`,
			[]model.FieldError{{Field: "status", Reason: "oneof", Param: "0 1"}}},
		{"multiple invalid fields", `{"name":"","status":5}`,
			[]model.FieldError{{Field: "name", Reason: "required"}, {Field: "status", Reason: "oneof", Param: "0 1"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			f := setUp(t) // 不設定 EXPECT：驗證失敗時不應呼叫 service

			// Act
			rec := serve(t, f.controller.Create, "", tc.body)

			// Assert
			assertError(t, rec, http.StatusBadRequest, "VALIDATION_FAILED")
			assert.Equal(t, tc.wantDetails, decodeError(t, rec).Error.Details)
		})
	}
}

func TestCreate_MalformedBody_Returns400InvalidJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"syntax error", `{name:`},
		{"empty body", ``},
		{"unknown field", `{"name":"x","foo":1}`},
		{"wrong field type", `{"name":"x","status":"1"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			f := setUp(t) // 不設定 EXPECT：JSON 壞掉時不應呼叫 service

			// Act
			rec := serve(t, f.controller.Create, "", tc.body)

			// Assert
			assertError(t, rec, http.StatusBadRequest, "INVALID_JSON")
		})
	}
}

func TestCreate_BodyOver1MB_Returns413PayloadTooLarge(t *testing.T) {
	// Arrange
	f := setUp(t)
	body := `{"name":"` + strings.Repeat("a", maxBodyBytes) + `"}`

	// Act
	rec := serve(t, f.controller.Create, "", body)

	// Assert
	assertError(t, rec, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE")
}

// ---------- Update ----------

func TestUpdate_ValidBody_Returns200WithUpdatedTask(t *testing.T) {
	// Arrange
	f := setUp(t)
	task := model.Task{Name: "new", Status: model.StatusComplete}
	f.service.EXPECT().Update(gomock.Any(), 3, task).Return(model.Task{ID: 3, Name: "new", Status: model.StatusComplete}, nil)

	// Act
	rec := serve(t, f.controller.Update, "3", `{"name":"new","status":1}`)

	// Assert
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	assert.JSONEq(t, `{"result":{"id":3,"name":"new","status":1}}`, rec.Body.String())
}

// PUT 是整筆覆蓋：沒傳 status 就是設成未完成，而不是保留原本的值。
func TestUpdate_StatusOmitted_PassesIncompleteStatusToService(t *testing.T) {
	// Arrange
	f := setUp(t)
	f.service.EXPECT().
		Update(gomock.Any(), 1, model.Task{Name: "x", Status: model.StatusIncomplete}).
		Return(model.Task{ID: 1, Name: "x", Status: model.StatusIncomplete}, nil)

	// Act
	rec := serve(t, f.controller.Update, "1", `{"name":"x"}`)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	// status 是否以 0 傳給 service，由 EXPECT 的參數比對驗證
}

func TestUpdate_TaskNotFound_Returns404TaskNotFound(t *testing.T) {
	// Arrange
	f := setUp(t)
	f.service.EXPECT().Update(gomock.Any(), 999, gomock.Any()).Return(model.Task{}, model.ErrTaskNotFound)

	// Act
	rec := serve(t, f.controller.Update, "999", `{"name":"x"}`)

	// Assert
	assertError(t, rec, http.StatusNotFound, "TASK_NOT_FOUND")
}

func TestUpdate_InvalidID_Returns400InvalidID(t *testing.T) {
	for _, id := range []string{"abc", "0", "-1"} {
		t.Run(id, func(t *testing.T) {
			// Arrange
			f := setUp(t) // 不設定 EXPECT：id 不合法時不應呼叫 service

			// Act
			rec := serve(t, f.controller.Update, id, `{"name":"x"}`)

			// Assert
			assertError(t, rec, http.StatusBadRequest, "INVALID_ID")
		})
	}
}

// ---------- Delete ----------

func TestDelete_ExistingTask_Returns204AndPassesIDToService(t *testing.T) {
	// Arrange
	f := setUp(t)
	f.service.EXPECT().Delete(gomock.Any(), 5).Return(nil)

	// Act
	rec := serve(t, f.controller.Delete, "5", "")

	// Assert
	assert.Equal(t, http.StatusNoContent, rec.Code)
	// id 是否正確傳給 service，由 EXPECT 的參數比對驗證
}

// task 不存在時 service 會回 nil（Delete 是冪等的），所以 controller 只會在非預期的錯誤時失敗。
func TestDelete_UnexpectedServiceError_Returns500(t *testing.T) {
	// Arrange
	f := setUp(t)
	f.service.EXPECT().Delete(gomock.Any(), 5).Return(errors.New("storage unavailable"))

	// Act
	rec := serve(t, f.controller.Delete, "5", "")

	// Assert
	assertError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
}

func TestDelete_InvalidID_Returns400InvalidID(t *testing.T) {
	// Arrange
	f := setUp(t) // 不設定 EXPECT：id 不合法時不應呼叫 service

	// Act
	rec := serve(t, f.controller.Delete, "abc", "")

	// Assert
	assertError(t, rec, http.StatusBadRequest, "INVALID_ID")
}
