package err

import "katydid-common-account/pkg/validator/v5/core"

// ValidationError 验证错误集合
// 职责：包装多个字段错误
type ValidationError struct {
	formatter core.IErrorFormatter
	message   string
	errors    []core.IFieldError
}

// NewValidationError 创建验证错误
func NewValidationError(formatter core.IErrorFormatter, opts ...ValidationErrorOption) core.IValidationError {
	ve := &ValidationError{formatter: formatter}

	// 应用选项
	for _, opt := range opts {
		opt(ve)
	}

	return ve
}

// ValidationErrorOption 字段错误选项函数类型
type ValidationErrorOption func(*ValidationError)

// WithTotalMessage 设置消息
func WithTotalMessage(message string) ValidationErrorOption {
	return func(ve *ValidationError) {
		ve.message = message
	}
}

// WithError 追加单个错误
func WithError(err core.IFieldError) ValidationErrorOption {
	return func(ve *ValidationError) {
		ve.errors = append(ve.errors, err)
	}
}

// WithErrors 设置多个错误
func WithErrors(errs []core.IFieldError) ValidationErrorOption {
	return func(ve *ValidationError) {
		ve.errors = append(ve.errors, errs...)
	}
}

// HasErrors 是否有错误
func (ve *ValidationError) HasErrors() bool {
	return len(ve.errors) > 0
}

// Formatter 格式化所有错误
func (ve *ValidationError) Formatter() []string {
	var formatters []string

	if len(ve.errors) == 0 {
		if len(ve.message) > 0 {
			formatters = append(formatters, ve.message)
		}
		return formatters
	}

	for _, err := range ve.errors {
		if err == nil {
			continue
		}
		format := ve.formatter.Format(err)
		formatters = append(formatters, format)
	}

	return formatters
}
