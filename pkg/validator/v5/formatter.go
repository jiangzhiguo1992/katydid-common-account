package v5

import (
	"fmt"
	"strings"
)

// ErrorFormatter 错误格式化器接口
// 职责：格式化错误信息
type ErrorFormatter interface {
	// Format 格式化单个错误
	Format(err *FieldError) string
	// FormatAll 格式化所有错误
	FormatAll(errs []*FieldError) string
}

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
	if len(err.Message) > 0 {
		return err.Message
	}

	// 生成默认消息
	var builder strings.Builder
	builder.Grow(errorMessageEstimatedLength)

	if len(err.Namespace) > 0 {
		builder.WriteString("字段 '")
		builder.WriteString(err.Namespace)
		builder.WriteString("' ")
	}

	builder.WriteString("验证失败")

	if len(err.Tag) > 0 {
		builder.WriteString("，规则: ")
		builder.WriteString(err.Tag)
	}

	if len(err.Param) > 0 {
		builder.WriteString("，参数: ")
		builder.WriteString(err.Param)
	}

	if err.Value != nil {
		builder.WriteString("，值: ")
		builder.WriteString(fmt.Sprintf("%v", err.Value))
	}

	return builder.String()
}

// FormatAll 格式化所有错误
func (f *DefaultErrorFormatter) FormatAll(errs []*FieldError) string {
	if len(errs) == 0 {
		return "验证通过"
	}

	if len(errs) == 1 {
		return f.Format(errs[0])
	}

	var builder strings.Builder
	builder.Grow(len(errs) * errorMessageEstimatedLength)

	builder.WriteString(fmt.Sprintf("验证失败，共 %d 个错误:\n", len(errs)))

	for i, err := range errs {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, f.Format(err)))
	}

	return builder.String()
}

// JSONErrorFormatter JSON错误格式化器
// 职责：将错误格式化为JSON格式
type JSONErrorFormatter struct{}

// NewJSONErrorFormatter 创建JSON错误格式化器
func NewJSONErrorFormatter() *JSONErrorFormatter {
	return &JSONErrorFormatter{}
}

// Format 格式化单个错误为JSON字符串
func (f *JSONErrorFormatter) Format(err *FieldError) string {
	if err == nil {
		return "{}"
	}

	var builder strings.Builder
	builder.WriteString("{")
	builder.WriteString(fmt.Sprintf(`"namespace":"%s"`, err.Namespace))
	builder.WriteString(fmt.Sprintf(`,"tag":"%s"`, err.Tag))

	if len(err.Param) > 0 {
		builder.WriteString(fmt.Sprintf(`,"param":"%s"`, err.Param))
	}

	if len(err.Message) > 0 {
		builder.WriteString(fmt.Sprintf(`,"message":"%s"`, err.Message))
	}

	builder.WriteString("}")
	return builder.String()
}

// FormatAll 格式化所有错误为JSON数组
func (f *JSONErrorFormatter) FormatAll(errs []*FieldError) string {
	if len(errs) == 0 {
		return "[]"
	}

	var builder strings.Builder
	builder.WriteString("[")

	for i, err := range errs {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(f.Format(err))
	}

	builder.WriteString("]")
	return builder.String()
}

// TODO:GG 再来个国际化fomteer，外部注册?
