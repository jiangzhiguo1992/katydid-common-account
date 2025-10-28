package error

// ValidationError 验证错误集合
// 职责：包装多个字段错误
type ValidationError struct {
	formatter formatter2.ErrorFormatter
	message   string
	errors    []*FieldError
}

// NewValidationError 创建验证错误
func NewValidationError(formatter formatter2.ErrorFormatter) *ValidationError {
	return &ValidationError{formatter: formatter}
}

// WithMessage 设置消息
func (ve *ValidationError) WithMessage(message string) *ValidationError {
	ve.message = message
	return ve
}

// WithError 添加单个错误
func (ve *ValidationError) WithError(err *FieldError) *ValidationError {
	ve.errors = append(ve.errors, err)
	return ve
}

// WithErrors 添加多个错误
func (ve *ValidationError) WithErrors(errs []*FieldError) *ValidationError {
	ve.errors = errs
	return ve
}

// FormatterAll 格式化所有错误
func (ve *ValidationError) FormatterAll() []string {
	var formatters []string

	for _, err := range ve.errors {
		if err == nil {
			continue
		}
		format := ve.formatter.Format(err)
		formatters = append(formatters, format)
	}

	return formatters
}

// Error 实现 error 接口
func (ve *ValidationError) Error() string {
	if len(ve.errors) == 0 {
		if len(ve.message) > 0 {
			return ve.message
		}
		return "validation passed"
	}
	if len(ve.errors) == 1 {
		return ve.errors[0].Error()
	}
	return "validation failed with " + string(rune(len(ve.errors))) + " errors"
}

// HasErrors 是否有错误
func (ve *ValidationError) HasErrors() bool {
	return len(ve.errors) > 0
}
