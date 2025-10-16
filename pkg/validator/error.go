package validator

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidationError 统一的验证错误结构
// 包含详细的错误信息，支持国际化，不会因一个错误就中断
type ValidationError struct {
	// Errors 所有验证错误的集合
	Errors []*FieldError `json:"errors"`
	// Scene 验证场景
	Scene ValidateScene `json:"scene,omitempty"`
	// Message 总体错误消息（可选）
	Message string `json:"message,omitempty"`
}

// FieldError 单个字段的验证错误
type FieldError struct {
	// Field 字段名（JSON tag 名称）
	Field string `json:"field"`
	// StructField 结构体字段名
	StructField string `json:"struct_field,omitempty"`
	// Tag 验证标签（如 required, email, min 等）
	Tag string `json:"tag"`
	// Param 验证参数（如 min=3 中的 3）
	Param string `json:"param,omitempty"`
	// Value 字段的实际值（可选，用于调试）
	Value interface{} `json:"value,omitempty"`
	// Message 错误消息（已国际化）
	Message string `json:"message"`
	// Code 错误代码（用于国际化查找）
	Code string `json:"code,omitempty"`
	// Namespace 字段的完整命名空间（如 User.Profile.Email）
	Namespace string `json:"namespace,omitempty"`
}

// Error 实现 error 接口
func (ve *ValidationError) Error() string {
	if len(ve.Errors) == 0 {
		return "validation failed"
	}

	var builder strings.Builder
	builder.Grow(len(ve.Errors) * errorMessageEstimateLen)

	for i, err := range ve.Errors {
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
		return fmt.Sprintf("field '%s': %s", fe.Field, fe.Message)
	}
	return fmt.Sprintf("field '%s' validation failed on tag '%s'", fe.Field, fe.Tag)
}

// HasErrors 检查是否有验证错误
func (ve *ValidationError) HasErrors() bool {
	return len(ve.Errors) > 0
}

// AddError 添加一个字段错误
func (ve *ValidationError) AddError(err *FieldError) {
	if err != nil {
		ve.Errors = append(ve.Errors, err)
	}
}

// AddErrors 批量添加字段错误
func (ve *ValidationError) AddErrors(errors []*FieldError) {
	ve.Errors = append(ve.Errors, errors...)
}

// ToJSON 转换为 JSON 格式
func (ve *ValidationError) ToJSON() ([]byte, error) {
	return json.Marshal(ve)
}

// GetErrorsByField 按字段名获取错误
func (ve *ValidationError) GetErrorsByField(field string) []*FieldError {
	var errors []*FieldError
	for _, err := range ve.Errors {
		if err.Field == field {
			errors = append(errors, err)
		}
	}
	return errors
}

// GetErrorsByTag 按验证标签获取错误
func (ve *ValidationError) GetErrorsByTag(tag string) []*FieldError {
	var errors []*FieldError
	for _, err := range ve.Errors {
		if err.Tag == tag {
			errors = append(errors, err)
		}
	}
	return errors
}

// ErrorMessageProvider 错误详情提供者接口
// 模型实现此接口可以提供更详细的错误信息
type ErrorMessageProvider interface {
	// GetErrorMessage 获取字段验证错误的详细信息
	// fieldName: 字段名
	// tag: 验证标签
	// param: 验证参数
	// value: 字段实际值
	// 返回 FieldError 结构，包含完整的错误详情
	GetErrorMessage(fieldName, tag, param string, value interface{}) *FieldError
}

// ValidationContext 验证上下文，用于传递验证环境信息
type ValidationContext struct {
	// Scene 验证场景
	Scene ValidateScene
	// ParentNamespace 父级命名空间（用于嵌套对象）
	ParentNamespace string
}

// NewValidationContext 创建验证上下文
func NewValidationContext(scene ValidateScene) *ValidationContext {
	return &ValidationContext{
		Scene: scene,
	}
}

// NewValidationError 创建一个新的验证错误
func NewValidationError(scene ValidateScene) *ValidationError {
	return &ValidationError{
		Errors: make([]*FieldError, 0),
		Scene:  scene,
	}
}

// NewFieldError 创建一个新的字段错误
func NewFieldError(field, tag, param, message string) *FieldError {
	return &FieldError{
		Field:   field,
		Tag:     tag,
		Param:   param,
		Message: message,
	}
}

// NewFieldErrorWithDetail 创建一个带详细信息的字段错误
func NewFieldErrorWithDetail(field, structField, tag, param, message, code, namespace string, value interface{}) *FieldError {
	return &FieldError{
		Field:       field,
		StructField: structField,
		Tag:         tag,
		Param:       param,
		Message:     message,
		Code:        code,
		Namespace:   namespace,
		Value:       value,
	}
}

// MergeValidationErrors 合并多个验证错误
func MergeValidationErrors(errors ...*ValidationError) *ValidationError {
	merged := NewValidationError("")
	for _, err := range errors {
		if err != nil && err.HasErrors() {
			merged.AddErrors(err.Errors)
		}
	}
	return merged
}

// BuildFieldError 构建字段错误（供模型使用的辅助方法）
// 参数：
//   - field: 字段名
//   - tag: 验证标签
//   - message: 错误消息
//   - opts: 可选参数
func BuildFieldError(field, tag, message string, opts ...FieldErrorOption) *FieldError {
	fe := &FieldError{
		Field:   field,
		Tag:     tag,
		Message: message,
	}

	for _, opt := range opts {
		opt(fe)
	}

	return fe
}

// FieldErrorOption 字段错误选项函数
type FieldErrorOption func(*FieldError)

// WithParam 设置验证参数
func WithParam(param string) FieldErrorOption {
	return func(fe *FieldError) {
		fe.Param = param
	}
}

// WithValue 设置字段值
func WithValue(value interface{}) FieldErrorOption {
	return func(fe *FieldError) {
		fe.Value = value
	}
}

// WithCode 设置错误代码
func WithCode(code string) FieldErrorOption {
	return func(fe *FieldError) {
		fe.Code = code
	}
}

// WithStructField 设置结构体字段名
func WithStructField(structField string) FieldErrorOption {
	return func(fe *FieldError) {
		fe.StructField = structField
	}
}

// WithNamespace 设置命名空间
func WithNamespace(namespace string) FieldErrorOption {
	return func(fe *FieldError) {
		fe.Namespace = namespace
	}
}
