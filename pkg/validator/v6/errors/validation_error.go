package errors

import (
	"katydid-common-account/pkg/validator/v6/core"
	"strings"
)

// validationError 验证错误实现
type validationError struct {
	fieldErrors []core.IFieldError
	formatter   core.IErrorFormatter
	messages    []string // 缓存格式化后的消息
}

// NewValidationError 创建验证错误
func NewValidationError(fieldErrors []core.IFieldError, formatter core.IErrorFormatter) core.IValidationError {
	if formatter == nil {
		formatter = NewDefaultFormatter()
	}

	ve := &validationError{
		fieldErrors: fieldErrors,
		formatter:   formatter,
	}

	// 预先格式化所有错误
	if len(fieldErrors) > 0 {
		ve.messages = formatter.FormatAll(fieldErrors)
	}

	return ve
}

// Error 实现 error 接口
func (e *validationError) Error() string {
	if len(e.messages) == 0 {
		return "validation failed"
	}
	return strings.Join(e.messages, "; ")
}

// HasErrors 是否有错误
func (e *validationError) HasErrors() bool {
	return len(e.fieldErrors) > 0
}

// Errors 获取所有格式化的错误消息
func (e *validationError) Errors() []string {
	return e.messages
}

// FieldErrors 获取原始字段错误
func (e *validationError) FieldErrors() []core.IFieldError {
	return e.fieldErrors
}

// First 获取第一个错误
func (e *validationError) First() string {
	if len(e.messages) == 0 {
		return ""
	}
	return e.messages[0]
}
