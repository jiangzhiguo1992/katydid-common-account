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
	Scene ValidateScene
	// Message 总体错误消息（可选）
	Message string `json:"message,omitempty"`
	// Errors 所有验证错误的集合
	Errors []*FieldError `json:"errors"`
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
	Value any `json:"value,omitempty"`
}

// NewValidationContext 创建验证上下文
func NewValidationContext(scene ValidateScene) *ValidationContext {
	return &ValidationContext{
		Scene:  scene,
		Errors: make([]*FieldError, 0),
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
	return fmt.Sprintf("field '%s' validation failed on tag '%s'", fe.Namespace, fe.Tag)
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
		Namespace: error.Namespace(),
		Tag:       error.Tag(),
		Params:    []string{error.Param()},
		Value:     error.Value(),
	})
}

// AddErrorByDetail 通过详细信息添加字段错误
func (vc *ValidationContext) AddErrorByDetail(namespace, tag string, params []string, value any) {
	vc.Errors = append(vc.Errors, &FieldError{
		Namespace: namespace,
		Tag:       tag,
		Params:    params,
		Value:     value,
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
