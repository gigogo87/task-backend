package dto

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"taskapi/internal/model"
)

// NewValidator 建立專案共用的 validator。
// validator 會快取 struct 的解析結果，所以整個程式只建一個，由 dependency provider 注入。
func NewValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	// 錯誤訊息的欄位名稱改用 json tag（name），而不是 Go 的欄位名稱（Name），client 才看得懂。
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		if name == "" {
			return f.Name
		}
		return name
	})
	return v
}

// ValidationError 把 validator 的錯誤轉成 VALIDATION_FAILED，並附上每個欄位的錯誤。
// 不是 ValidationErrors 的錯誤（例如傳錯型別給 validator）代表程式寫錯，原樣回傳讓上層當成 500。
func ValidationError(err error) error {
	var ves validator.ValidationErrors
	if !errors.As(err, &ves) {
		return err
	}
	details := make([]model.FieldError, 0, len(ves))
	for _, fe := range ves {
		details = append(details, model.FieldError{
			Field:  fe.Field(),
			Reason: fe.Tag(),
			Param:  fe.Param(),
		})
	}
	return model.ErrValidationFailed.WithDetails(details)
}
