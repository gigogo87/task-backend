package model

// Status 代表任務狀態：0 = 未完成，1 = 已完成
const (
	StatusIncomplete = 0
	StatusComplete   = 1
)

// Task 是我們 API 的核心資料結構，各層之間都用它傳遞資料。
// 後面的 `json:"..."` 叫做 struct tag，告訴 encoding/json 轉成 JSON 時用什麼欄位名稱。
type Task struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Status int    `json:"status"`
}
