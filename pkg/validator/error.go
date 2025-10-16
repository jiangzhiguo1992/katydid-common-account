package validator

import (
	"encoding/json"
	"fmt"
	"strings"
)

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

// ErrorMessageProvider 错误详情提供者接口
// 模型实现此接口可以提供更详细的错误信息
type ErrorMessageProvider interface {
	// GetErrorMessage 获取字段验证错误的详细信息
	// namespace: 字段完整命名空间
	// tag: 验证标签
	// code: 错误代码
	// param: 验证参数
	// value: 字段实际值
	// 返回 FieldError 结构，包含完整的错误详情
	GetErrorMessage(namespace, tag, code string, params []string, value interface{}) *FieldError
}

// ValidationError 统一的验证错误结构，多错误聚合
type ValidationError struct {
	// Errors 所有验证错误的集合
	Errors []*FieldError `json:"errors"`
	// Scene 验证场景
	Scene ValidateScene `json:"scene,omitempty"`
	// Message 总体错误消息（可选）
	Message string `json:"message,omitempty"`
}

// FieldError 单个字段的验证错误
// 国际化时，可以通过 Namespace + Tag 和 Params 字段查找对应的翻译
// 如User.Profile.Email_regex_len + params=["3", "100"]
type FieldError struct {
	// Namespace 字段的完整命名空间（如 User.Profile.Email）
	Namespace string `json:"namespace"`
	// Tag 验证标签（如 required, email, min 等）
	Tag string `json:"tag"`
	// Params 验证参数（如 min=3 中的 3）
	Params []string `json:"param,omitempty"`
	// Value 字段的实际值（可选，用于调试）
	Value interface{} `json:"value,omitempty"`
}

// NewValidationError 创建一个新的验证错误
func NewValidationError(scene ValidateScene) *ValidationError {
	return &ValidationError{
		Errors: make([]*FieldError, 0),
		Scene:  scene,
	}
}

// NewFieldError 创建一个新的字段错误
func NewFieldError(namespace, tag string, params []string, value interface{}) *FieldError {
	return &FieldError{
		Namespace: namespace,
		Tag:       tag,
		Params:    params,
		Value:     value,
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

// Error 实现 error 接口
func (ve *ValidationError) Error() string {
	if len(ve.Errors) == 0 {
		return "validation failed: no errors"
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
	return fmt.Sprintf("field '%s' validation failed on tag '%s'", fe.Namespace, fe.Tag)
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

// GetErrorsByNamespace 按命名空间获取错误
func (ve *ValidationError) GetErrorsByNamespace(namespace string) []*FieldError {
	var errors []*FieldError
	for _, err := range ve.Errors {
		if err.Namespace == namespace {
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

// FieldErrorOption 字段错误选项函数
type FieldErrorOption func(*FieldError)

// BuildFieldError 构建字段错误（供模型使用的辅助方法）
// 参数：
//   - field: 字段名
//   - tag: 验证标签
//   - message: 错误消息
//   - opts: 可选参数
func BuildFieldError(namespace, tag string, opts ...FieldErrorOption) *FieldError {
	fe := &FieldError{
		Namespace: namespace,
		Tag:       tag,
	}

	for _, opt := range opts {
		opt(fe)
	}

	return fe
}

// WithParam 设置验证参数
func WithParam(params []string) FieldErrorOption {
	return func(fe *FieldError) {
		fe.Params = params
	}
}

// WithValue 设置字段值
func WithValue(value interface{}) FieldErrorOption {
	return func(fe *FieldError) {
		fe.Value = value
	}
}
