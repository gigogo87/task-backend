// Package controller 負責 HTTP 這一層：解析請求 → 驗證格式 → 呼叫 service → 把結果轉成 JSON 回應。
// 路由註冊在 app/router.go。
package controller

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"

	"taskapi/internal/dto"
	"taskapi/internal/model"
	"taskapi/internal/response"
	"taskapi/internal/service"
)

const maxBodyBytes = 1 << 20 // 限制 body 最大 1MB，避免被塞爆

type TaskController struct {
	service  service.TaskService
	validate *validator.Validate
	logger   *slog.Logger
}

// NewTaskController 透過參數注入所有依賴，測試時可以換成假的 service。
func NewTaskController(svc service.TaskService, validate *validator.Validate, logger *slog.Logger) *TaskController {
	return &TaskController{service: svc, validate: validate, logger: logger}
}

// List 處理 GET /tasks
func (c *TaskController) List(w http.ResponseWriter, r *http.Request) {
	c.logger.InfoContext(r.Context(), "[CTR IN] List")
	tasks, err := c.service.List(r.Context())
	if err != nil {
		c.fail(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"result": tasks})
}

// Create 處理 POST /task
func (c *TaskController) Create(w http.ResponseWriter, r *http.Request) {
	c.logger.InfoContext(r.Context(), "[CTR IN] Create")
	req, err := c.decodeAndValidate(w, r)
	if err != nil {
		c.fail(w, r, err)
		return
	}

	created, err := c.service.Create(r.Context(), req.ToModel())
	if err != nil {
		c.fail(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, map[string]any{"result": created})
}

// Update 處理 PUT /task/{id}
func (c *TaskController) Update(w http.ResponseWriter, r *http.Request) {
	c.logger.InfoContext(r.Context(), "[CTR IN] Update")
	id, err := c.parseID(r)
	if err != nil {
		c.fail(w, r, err)
		return
	}
	req, err := c.decodeAndValidate(w, r)
	if err != nil {
		c.fail(w, r, err)
		return
	}

	updated, err := c.service.Update(r.Context(), id, req.ToModel())
	if err != nil {
		c.fail(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"result": updated})
}

// Delete 處理 DELETE /task/{id}
func (c *TaskController) Delete(w http.ResponseWriter, r *http.Request) {
	c.logger.InfoContext(r.Context(), "[CTR IN] Delete")
	id, err := c.parseID(r)
	if err != nil {
		c.fail(w, r, err)
		return
	}

	if err := c.service.Delete(r.Context(), id); err != nil {
		c.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent) // 204：成功但沒有內容要回
}

// ---------- 以下是共用的小工具函式 ----------

// parseID 從 URL 的 {id} 取出數字，並用 validator 檢查必須是正整數。
func (c *TaskController) parseID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || c.validate.Var(id, "gt=0") != nil {
		return 0, response.ErrInvalidID
	}
	return id, nil
}

// decodeAndValidate 把 JSON body 解析成 dto.TaskRequest，再用 validator 檢查格式。
func (c *TaskController) decodeAndValidate(w http.ResponseWriter, r *http.Request) (dto.TaskRequest, error) {
	var req dto.TaskRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields() // 多送奇怪欄位就報錯
	if err := dec.Decode(&req); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return dto.TaskRequest{}, response.ErrPayloadTooLarge
		}
		return dto.TaskRequest{}, response.ErrInvalidJSON
	}

	if err := c.validate.Struct(req); err != nil {
		return dto.TaskRequest{}, dto.ValidationError(err)
	}
	return req, nil
}

// fail 回傳錯誤回應。非預期的錯誤（不是 AppError）先寫進 log，因為回給 client 的只有 "internal server error"。
func (c *TaskController) fail(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *model.AppError
	if !errors.As(err, &appErr) {
		c.logger.ErrorContext(r.Context(), "unexpected error",
			"error", err, "method", r.Method, "path", r.URL.Path)
	}
	response.WriteError(w, r, err)
}
