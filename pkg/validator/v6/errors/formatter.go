package errors

import (
	"fmt"
	"katydid-common-account/pkg/validator/v6/core"
)

// ============================================================================
// 默认格式化器 - 简单格式
// ============================================================================

// defaultFormatter 默认错误格式化器
type defaultFormatter struct{}

// NewDefaultFormatter 创建默认格式化器
func NewDefaultFormatter() core.IErrorFormatter {
	return &defaultFormatter{}
}

// Format 格式化单个错误
func (f *defaultFormatter) Format(err core.IFieldError) string {
	return err.Message()
}

// FormatAll 格式化所有错误
func (f *defaultFormatter) FormatAll(errs []core.IFieldError) []string {
	messages := make([]string, len(errs))
	for i, err := range errs {
		messages[i] = f.Format(err)
	}
	return messages
}

// ============================================================================
// JSON 格式化器 - 适合 API 返回
// ============================================================================

// jsonFormatter JSON 格式化器
type jsonFormatter struct{}

// NewJSONFormatter 创建 JSON 格式化器
func NewJSONFormatter() core.IErrorFormatter {
	return &jsonFormatter{}
}

// Format 格式化单个错误为 JSON 风格
func (f *jsonFormatter) Format(err core.IFieldError) string {
	return fmt.Sprintf(`{"field":"%s","tag":"%s","message":"%s"}`,
		err.Field(), err.Tag(), err.Message())
}

// FormatAll 格式化所有错误为 JSON 数组风格
func (f *jsonFormatter) FormatAll(errs []core.IFieldError) []string {
	messages := make([]string, len(errs))
	for i, err := range errs {
		messages[i] = f.Format(err)
	}
	return messages
}

// ============================================================================
// 详细格式化器 - 包含所有信息
// ============================================================================

// detailedFormatter 详细格式化器
type detailedFormatter struct{}

// NewDetailedFormatter 创建详细格式化器
func NewDetailedFormatter() core.IErrorFormatter {
	return &detailedFormatter{}
}

// Format 格式化单个错误（包含详细信息）
func (f *detailedFormatter) Format(err core.IFieldError) string {
	if err.Param() != "" {
		return fmt.Sprintf("[%s] %s (tag=%s, param=%s, value=%v)",
			err.Namespace(), err.Message(), err.Tag(), err.Param(), err.Value())
	}
	return fmt.Sprintf("[%s] %s (tag=%s, value=%v)",
		err.Namespace(), err.Message(), err.Tag(), err.Value())
}

// FormatAll 格式化所有错误
func (f *detailedFormatter) FormatAll(errs []core.IFieldError) []string {
	messages := make([]string, len(errs))
	for i, err := range errs {
		messages[i] = f.Format(err)
	}
	return messages
}
