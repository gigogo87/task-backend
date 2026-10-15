// Package dto 定義 API 對外的資料格式（Data Transfer Object）與它的驗證規則。
// 刻意跟 model 分開：API 長什麼樣子，和系統內部怎麼表示資料，可以各自演進。
package dto

import "taskapi/internal/model"

// TaskRequest 是 POST /tasks 與 PUT /tasks/{id} 的 body 格式。client 不能自己指定 id。
//
// Status 用 *int 而不是 int：validator 的 required 會把 0 當成「沒填」，
// 但 0（未完成）是合法值，用指標才分得出「沒傳」和「傳了 0」。
type TaskRequest struct {
	Name   string `json:"name"   validate:"required,max=100"`
	Status *int   `json:"status" validate:"omitempty,oneof=0 1"`
}

// ToModel 轉成 model.Task。PUT 是整筆覆蓋，所以沒傳 status 一律視為未完成。
func (r TaskRequest) ToModel() model.Task {
	status := model.StatusIncomplete
	if r.Status != nil {
		status = *r.Status
	}
	return model.Task{Name: r.Name, Status: status}
}
