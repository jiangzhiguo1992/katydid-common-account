package v5

import (
	"fmt"
	"strings"
	"sync"
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
	builder := acquireStringBuilder()
	releaseStringBuilder(builder)

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

	builder := acquireStringBuilder()
	releaseStringBuilder(builder)

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

	builder := acquireStringBuilder()
	releaseStringBuilder(builder)

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

	builder := acquireStringBuilder()
	releaseStringBuilder(builder)

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

// LocalizesErrorFormatter 国际化错误格式化器
// 职责：将错误格式化为国际化配置模板
type LocalizesErrorFormatter struct{}

// NewLocalizesErrorFormatter 创建国际化错误格式化器
func NewLocalizesErrorFormatter() *LocalizesErrorFormatter {
	return &LocalizesErrorFormatter{}
}

// Format 格式化单个错误为国际化模板字符串
func (f *LocalizesErrorFormatter) Format(err *FieldError) string {
	if err == nil {
		return ""
	}

	// 生成国际化模板消息
	builder := acquireStringBuilder()
	releaseStringBuilder(builder)

	builder.Grow(errorMessageEstimatedLength / 2)

	if len(err.Namespace) > 0 && len(err.Tag) > 0 {
		builder.WriteString(err.Namespace)
		builder.WriteString(".")
		builder.WriteString(err.Tag)

		if len(err.Param) > 0 {
			builder.WriteString(".")
			builder.WriteString(err.Param)
		}
		return builder.String()
	}

	return err.Message
}

// FormatAll 格式化所有错误为国际化模板字符串
func (f *LocalizesErrorFormatter) FormatAll(errs []*FieldError) string {
	if len(errs) == 0 {
		return ""
	}

	builder := acquireStringBuilder()
	releaseStringBuilder(builder)

	builder.Grow(len(errs) * (errorMessageEstimatedLength / 2))

	for i, err := range errs {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(f.Format(err))
	}

	return builder.String()
}

var stringBuilderPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

// acquireStringBuilder 从对象池获取字符串构建器
func acquireStringBuilder() *strings.Builder {
	sb := stringBuilderPool.Get().(*strings.Builder)
	sb.Reset()
	return sb
}

// releaseStringBuilder 归还字符串构建器到对象池
func releaseStringBuilder(sb *strings.Builder) {
	if sb == nil {
		return
	}

	// 防止内存泄漏：不归还过大的Builder
	if sb.Cap() > 10*1024 { // 超过10KB
		return
	}

	sb.Reset()
	stringBuilderPool.Put(sb)
}
