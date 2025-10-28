package v5_refactored

import "fmt"

// ============================================================================
// 默认错误格式化器
// ============================================================================

// DefaultErrorFormatter 默认错误格式化器
type DefaultErrorFormatter struct{}

// NewDefaultErrorFormatter 创建默认错误格式化器
func NewDefaultErrorFormatter() *DefaultErrorFormatter {
	return &DefaultErrorFormatter{}
}

// Format 格式化单个错误
func (f *DefaultErrorFormatter) Format(err *FieldError) string {
	if err == nil {
		return ""
	}

	// 优先使用自定义消息
	if err.Message != "" {
		return err.Message
	}

	// 生成默认消息
	msg := "validation failed"

	if err.Namespace != "" {
		msg = fmt.Sprintf("field '%s' validation failed", err.Namespace)
	}

	if err.Tag != "" {
		msg += fmt.Sprintf(" on tag '%s'", err.Tag)
	}

	if err.Param != "" {
		msg += fmt.Sprintf(" with param '%s'", err.Param)
	}

	return msg
}

// FormatAll 格式化所有错误
func (f *DefaultErrorFormatter) FormatAll(errs []*FieldError) string {
	if len(errs) == 0 {
		return "validation passed"
	}

	if len(errs) == 1 {
		return f.Format(errs[0])
	}

	msg := fmt.Sprintf("validation failed with %d errors:\n", len(errs))
	for i, err := range errs {
		msg += fmt.Sprintf("%d. %s\n", i+1, f.Format(err))
	}

	return msg
}

// ============================================================================
// 中文错误格式化器
// ============================================================================

// ChineseErrorFormatter 中文错误格式化器
type ChineseErrorFormatter struct{}

// NewChineseErrorFormatter 创建中文错误格式化器
func NewChineseErrorFormatter() *ChineseErrorFormatter {
	return &ChineseErrorFormatter{}
}

// Format 格式化单个错误
func (f *ChineseErrorFormatter) Format(err *FieldError) string {
	if err == nil {
		return ""
	}

	if err.Message != "" {
		return err.Message
	}

	msg := "验证失败"

	if err.Namespace != "" {
		msg = fmt.Sprintf("字段 '%s' 验证失败", err.Namespace)
	}

	if err.Tag != "" {
		msg += fmt.Sprintf("，规则: %s", err.Tag)
	}

	if err.Param != "" {
		msg += fmt.Sprintf("，参数: %s", err.Param)
	}

	return msg
}

// FormatAll 格式化所有错误
func (f *ChineseErrorFormatter) FormatAll(errs []*FieldError) string {
	if len(errs) == 0 {
		return "验证通过"
	}

	if len(errs) == 1 {
		return f.Format(errs[0])
	}

	msg := fmt.Sprintf("验证失败，共 %d 个错误:\n", len(errs))
	for i, err := range errs {
		msg += fmt.Sprintf("%d. %s\n", i+1, f.Format(err))
	}

	return msg
}

// ============================================================================
// JSON 错误格式化器
// ============================================================================

// JSONErrorFormatter JSON 错误格式化器
type JSONErrorFormatter struct{}

// NewJSONErrorFormatter 创建 JSON 错误格式化器
func NewJSONErrorFormatter() *JSONErrorFormatter {
	return &JSONErrorFormatter{}
}

// Format 格式化单个错误
func (f *JSONErrorFormatter) Format(err *FieldError) string {
	if err == nil {
		return "{}"
	}

	return fmt.Sprintf(`{"field":"%s","tag":"%s","param":"%s","message":"%s"}`,
		err.Namespace, err.Tag, err.Param, err.Message)
}

// FormatAll 格式化所有错误
func (f *JSONErrorFormatter) FormatAll(errs []*FieldError) string {
	if len(errs) == 0 {
		return `{"errors":[]}`
	}

	msg := `{"errors":[`
	for i, err := range errs {
		if i > 0 {
			msg += ","
		}
		msg += f.Format(err)
	}
	msg += `]}`

	return msg
}
