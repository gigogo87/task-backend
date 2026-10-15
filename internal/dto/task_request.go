// Package dto 定義 API 對外的資料格式（Data Transfer Object）與它的驗證規則。
// 刻意跟 model 分開：API 長什麼樣子，和系統內部怎麼表示資料，可以各自演進。
package dto

import "taskapi/internal/model"

// TaskRequest 是 POST /tasks 與 PUT /tasks/{id} 的 body 格式。client 不能自己指定 id。
//
// Status 用 *int 而不是 int：validator 的 required 會把 0 當成「沒填」，
// 但 0（未完成）是合法值，用指標才分得出「沒傳」和「傳了 0」。
//
// validate tag 裡的數字無法改用常數（Go 的 struct tag 只能寫字串字面值），因此在這裡說明：
//   - max=100：任務名稱最多 100 字元。
//   - oneof=0 1：對應 model.StatusIncomplete 與 model.StatusComplete，新增狀態時兩邊要一起改。
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
