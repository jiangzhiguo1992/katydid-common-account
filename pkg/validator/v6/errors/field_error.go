package errors

import (
	"fmt"
	"katydid-common-account/pkg/validator/v6/core"
)

// fieldError 字段错误实现
// 设计原则：值对象模式，不可变
type fieldError struct {
	namespace string // 完整命名空间
	field     string // 字段名
	tag       string // 验证标签
	param     string // 验证参数
	value     any    // 字段值
	message   string // 错误消息
}

// NewFieldError 创建字段错误
func NewFieldError(namespace, field, tag string, opts ...FieldErrorOption) core.FieldError {
	err := &fieldError{
		namespace: namespace,
		field:     field,
		tag:       tag,
	}

	// 应用选项
	for _, opt := range opts {
		opt(err)
	}

	// 如果没有自定义消息，生成默认消息
	if err.message == "" {
		err.message = err.defaultMessage()
	}

	return err
}

// FieldErrorOption 字段错误选项
type FieldErrorOption func(*fieldError)

// WithParam 设置验证参数
func WithParam(param string) FieldErrorOption {
	return func(e *fieldError) {
		e.param = param
	}
}

// WithValue 设置字段值
func WithValue(value any) FieldErrorOption {
	return func(e *fieldError) {
		e.value = value
	}
}

// WithMessage 设置自定义消息
func WithMessage(message string) FieldErrorOption {
	return func(e *fieldError) {
		e.message = message
	}
}

// Namespace 实现 FieldError 接口
func (e *fieldError) Namespace() string {
	return e.namespace
}

// Field 实现 FieldError 接口
func (e *fieldError) Field() string {
	return e.field
}

// Tag 实现 FieldError 接口
func (e *fieldError) Tag() string {
	return e.tag
}

// Param 实现 FieldError 接口
func (e *fieldError) Param() string {
	return e.param
}

// Value 实现 FieldError 接口
func (e *fieldError) Value() any {
	return e.value
}

// Message 实现 FieldError 接口
func (e *fieldError) Message() string {
	return e.message
}

// Error 实现 error 接口
func (e *fieldError) Error() string {
	return e.message
}

// defaultMessage 生成默认错误消息
func (e *fieldError) defaultMessage() string {
	if e.param != "" {
		return fmt.Sprintf("Field '%s' failed validation on tag '%s' with param '%s'",
			e.field, e.tag, e.param)
	}
	return fmt.Sprintf("Field '%s' failed validation on tag '%s'",
		e.field, e.tag)
}
