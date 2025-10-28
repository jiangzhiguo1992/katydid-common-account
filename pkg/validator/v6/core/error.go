package core

import "fmt"

// FieldError 字段错误
// 职责：描述单个字段的验证错误
// 设计原则：不可变对象，线程安全
type FieldError struct {
	// Namespace 字段的完整命名空间路径（如 User.Profile.Email）
	Namespace string

	// Field 字段名（简短名称）
	Field string

	// Tag 验证标签（如 required, email, min, max）
	Tag string

	// Param 验证参数（如 min=3 中的 "3"）
	Param string

	// Value 字段的实际值（谨慎使用，可能包含敏感信息）
	Value any

	// Message 用户友好的错误消息
	Message string
}

// NewFieldError 创建字段错误
func NewFieldError(namespace, tag string) *FieldError {
	return &FieldError{
		Namespace: namespace,
		Tag:       tag,
	}
}

// WithField 设置字段名
func (e *FieldError) WithField(field string) *FieldError {
	e.Field = field
	return e
}

// WithParam 设置参数
func (e *FieldError) WithParam(param string) *FieldError {
	e.Param = param
	return e
}

// WithValue 设置值（安全检查：限制大小）
func (e *FieldError) WithValue(value any) *FieldError {
	// 防止存储过大的值
	if estimateSize(value) <= 4096 {
		e.Value = value
	}
	return e
}

// WithMessage 设置消息
func (e *FieldError) WithMessage(message string) *FieldError {
	if len(message) > 2048 {
		e.Message = message[:2048]
	} else {
		e.Message = message
	}
	return e
}

// Error 实现 error 接口
func (e *FieldError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("field '%s' validation failed on tag '%s'", e.Namespace, e.Tag)
}

// estimateSize 估算值的大小
func estimateSize(value any) int {
	if value == nil {
		return 0
	}
	// 简单估算，实际应该更精确
	return len(fmt.Sprintf("%v", value))
}

// ValidationError 验证错误集合
// 职责：封装多个字段错误
type ValidationError struct {
	errors  []*FieldError
	message string
}

// NewValidationError 创建验证错误
func NewValidationError(errors []*FieldError) *ValidationError {
	return &ValidationError{
		errors: errors,
	}
}

// Errors 获取所有错误
func (e *ValidationError) Errors() []*FieldError {
	return e.errors
}

// Error 实现 error 接口
func (e *ValidationError) Error() string {
	if e.message != "" {
		return e.message
	}
	if len(e.errors) == 0 {
		return "validation failed"
	}
	if len(e.errors) == 1 {
		return e.errors[0].Error()
	}
	return fmt.Sprintf("validation failed with %d errors", len(e.errors))
}

// WithMessage 设置整体消息
func (e *ValidationError) WithMessage(message string) *ValidationError {
	e.message = message
	return e
}

// HasErrors 是否有错误
func (e *ValidationError) HasErrors() bool {
	return len(e.errors) > 0
}

// Count 错误数量
func (e *ValidationError) Count() int {
	return len(e.errors)
}
