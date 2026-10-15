package model

// FieldError 是單一欄位的驗證錯誤，對應錯誤回應裡 details 陣列的每一筆。
// Reason 直接用 validator 的 tag 名稱（required、max、oneof...），Param 是 tag 的參數。
type FieldError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
	Param  string `json:"param,omitempty"`
}

// AppError 是整個專案統一的錯誤型別。
// 刻意不帶 HTTP status：model 不該知道 HTTP 的存在，錯誤碼對應 status 的表在 response package。
// Code 是對外承諾、給程式判斷用的字串，不能隨便改；Message 是給人看的說明。
type AppError struct {
	Code    string
	Message string
	Details []FieldError
}

func (e *AppError) Error() string {
	return e.Code + ": " + e.Message
}

// WithDetails 回傳一份帶有欄位錯誤的複本，不會改到共用的錯誤變數。
func (e *AppError) WithDetails(details []FieldError) *AppError {
	cp := *e
	cp.Details = details
	return &cp
}

// 跟商業邏輯有關、service / repository 會回傳的錯誤。
// 只會在 HTTP 層發生的錯誤（INVALID_JSON、RATE_LIMITED...）定義在 response package。
var (
	ErrValidationFailed = &AppError{Code: "VALIDATION_FAILED", Message: "request validation failed"}
	ErrTaskNotFound     = &AppError{Code: "TASK_NOT_FOUND", Message: "task not found"}
)
