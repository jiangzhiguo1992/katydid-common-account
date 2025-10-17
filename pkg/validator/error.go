package validator

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidationContext 验证上下文，用于传递验证环境信息
type ValidationContext struct {
	// Scene 验证场景
	Scene ValidateScene `json:"scene"`
	// Message 总体错误消息（可选）
	Message string `json:"message,omitempty"`
	// Errors 所有验证错误的集合（可选）
	Errors []*FieldError `json:"errors,omitempty"`
}

// FieldError 单个字段的验证错误
// 包含了 sl.ReportError 所需的所有参数
// 国际化时，可以通过 Namespace + Tag 和 Params 字段查找对应的翻译
// 如User.Profile.Email_regex_len + params=["3", "100"]
type FieldError struct {
	// FieldName 结构体字段名（对应 sl.ReportError 的 fieldName 参数）
	FieldName string `json:"field_name,omitempty"`
	// JsonName JSON 字段名（对应 sl.ReportError 的 jsonName 参数）
	JsonName string `json:"json_name"`
	// Tag 验证标签（如 required, email, min 等）
	Tag string `json:"tag"`
	// Param 验证参数（如 min=3 中的 "3"）
	Param string `json:"param,omitempty"`
	// Value 字段的实际值（用于 sl.ReportError 的 value 参数）
	Value any `json:"value,omitempty"`
	// Message 友好的错误消息（可选，用于直接显示给用户）
	Message string `json:"message,omitempty"`
	// Namespace 字段的完整命名空间（如 User.Profile.Email）
	Namespace string `json:"namespace,omitempty"`
}

// NewValidationContext 创建验证上下文
func NewValidationContext(scene ValidateScene) *ValidationContext {
	return &ValidationContext{
		Scene:  scene,
		Errors: make([]*FieldError, 0),
	}
}

// NewFieldError 创建字段错误
// value: 字段值
// fieldName: 结构体字段名
// jsonName: JSON 字段名
// tag: 验证标签
// param: 验证参数
func NewFieldError(value any, fieldName, jsonName, tag, param string) *FieldError {
	return &FieldError{
		FieldName: fieldName,
		JsonName:  jsonName,
		Tag:       tag,
		Param:     param,
		Value:     value,
		Namespace: jsonName,
	}
}

// Error 实现 error 接口
func (vc *ValidationContext) Error() string {
	if len(vc.Errors) == 0 {
		if len(vc.Message) == 0 {
			return "validation passed: no errors"
		}
		return fmt.Sprintf("validation failed: %s", vc.Message)
	}

	var builder strings.Builder
	builder.Grow(len(vc.Errors) * errorMessageEstimateLen)

	for i, err := range vc.Errors {
		if i > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString(err.String())
	}

	return builder.String()
}

// String 返回友好的错误信息
func (fe *FieldError) String() string {
	if fe.Message != "" {
		return fmt.Sprintf("field '%s': %s", fe.JsonName, fe.Message)
	}
	return fmt.Sprintf("field '%s' validation failed on tag '%s'", fe.JsonName, fe.Tag)
}

// HasErrors 检查是否有验证错误
func (vc *ValidationContext) HasErrors() bool {
	return len(vc.Errors) > 0
}

// AddError 通过 FieldError 添加字段错误
func (vc *ValidationContext) AddError(err *FieldError) {
	if err != nil {
		vc.Errors = append(vc.Errors, err)
	}
}

// AddErrorByValidator 通过 validator.FieldError 添加字段错误
func (vc *ValidationContext) AddErrorByValidator(error validator.FieldError) {
	vc.Errors = append(vc.Errors, &FieldError{
		FieldName: error.StructField(),
		JsonName:  error.Field(),
		Tag:       error.Tag(),
		Param:     error.Param(),
		Value:     error.Value(),
		Message:   error.Error(),
		Namespace: error.Namespace(),
	})
}

// AddErrorByDetail 通过详细信息添加字段错误
func (vc *ValidationContext) AddErrorByDetail(value any, field, json, tag, param, message, namespace string) {
	vc.Errors = append(vc.Errors, &FieldError{
		FieldName: field,
		JsonName:  json,
		Tag:       tag,
		Param:     param,
		Value:     value,
		Message:   message,
		Namespace: namespace,
	})
}

// AddErrors 批量添加字段错误
func (vc *ValidationContext) AddErrors(errors []*FieldError) {
	vc.Errors = append(vc.Errors, errors...)
}

// ToJSON 转换为 JSON 格式
func (vc *ValidationContext) ToJSON() ([]byte, error) {
	return json.Marshal(vc)
}

// GetErrorsByNamespace 按命名空间获取错误
func (vc *ValidationContext) GetErrorsByNamespace(namespace string) []*FieldError {
	var errors []*FieldError
	for _, err := range vc.Errors {
		if err.Namespace == namespace {
			errors = append(errors, err)
		}
	}
	return errors
}

// GetErrorsByTag 按验证标签获取错误
func (vc *ValidationContext) GetErrorsByTag(tag string) []*FieldError {
	var errors []*FieldError
	for _, err := range vc.Errors {
		if err.Tag == tag {
			errors = append(errors, err)
		}
	}
	return errors
}

func (fe *FieldError) WithMessage(message string) *FieldError {
	fe.Message = message
	return fe
}

func (fe *FieldError) WithNamespace(namespace string) *FieldError {
	fe.Namespace = namespace
	return fe
}

// ToReportErrorArgs 转换为 sl.ReportError 所需的参数
// 返回值对应 sl.ReportError(value, fieldName, jsonName, tag, param)
func (fe *FieldError) ToReportErrorArgs() (value interface{}, fieldName, jsonName, tag, param string) {
	return fe.Value, fe.FieldName, fe.JsonName, fe.Tag, fe.Param
}
