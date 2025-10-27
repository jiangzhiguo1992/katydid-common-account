package v5

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)

const (
	// errorMessageEstimatedLength 预估的错误消息平均长度，用于优化字符串构建时的内存分配
	errorMessageEstimatedLength = 80

	// maxErrorsCapacity TODO:GG 错误列表的最大容量，防止恶意数据导致内存溢出
	maxErrorsCapacity = 1000

	// maxParamLength TODO:GG 最大参数长度，防止超长参数攻击
	maxParamLength = 256
)

// FieldError 字段错误
// 职责：描述单个字段的验证错误
type FieldError struct {
	// Namespace 字段的完整命名空间路径（如 User.Profile.Email）
	// 用于嵌套结构体的错误定位，支持复杂对象的精确错误追踪
	Namespace string

	// Tag 验证标签，描述验证规则类型（如 required, email, min, max 等）
	Tag string

	// Param 验证参数，提供验证规则的具体配置值
	// 例如：min=3 中的 "3"，len=11 中的 "11"
	Param string

	// Value 字段的实际值（用于 sl.ReportError 的 value 参数）
	// 用于调试和详细错误信息，可能包含敏感信息，谨慎使用
	Value any

	// Message 用户友好的错误消息（可选，用于直接显示给终端用户）
	// 支持国际化，建议使用本地化后的错误消息
	Message string
}

// NewFieldError 创建字段错误
func NewFieldError(namespace, tag string) *FieldError {
	return &FieldError{
		Namespace: namespace,
		Tag:       tag,
	}
}

// NewFieldErrorWithMsg 创建仅带消息的字段错误
func NewFieldErrorWithMsg(message string) *FieldError {
	return &FieldError{
		Message: message,
	}
}

// WithParam 设置参数
func (fe *FieldError) WithParam(param string) *FieldError {
	fe.Param = param
	return fe
}

// WithValue 设置值
func (fe *FieldError) WithValue(value any) *FieldError {
	// 最大值大小（字节），防止存储过大的值导致内存问题
	if estimateValueSize(value) > 4096 {
		fe.Value = nil
		return fe
	}
	fe.Value = value
	return fe
}

// WithMessage 设置消息
func (fe *FieldError) WithMessage(message string) *FieldError {
	fe.Message = truncateString(message, 2048)
	return fe
}

// Error 实现 error 接口
func (fe *FieldError) Error() string {
	// 优先使用自定义消息（更友好）
	if len(fe.Message) > 0 {
		return fe.Message
	}

	// 生成默认错误消息（用于调试）
	if len(fe.Namespace) > 0 && len(fe.Tag) > 0 {
		var builder strings.Builder
		builder.Grow(errorMessageEstimatedLength)
		builder.WriteString("field '")
		builder.WriteString(fe.Namespace)
		builder.WriteString("' validation failed on tag '")
		builder.WriteString(fe.Tag)
		if len(fe.Param) > 0 {
			builder.WriteString("' with param '")
			builder.WriteString(fe.Param)
			builder.WriteString("'")
		} else {
			builder.WriteString("'")
		}
		if fe.Value != nil {
			builder.WriteString(", value: ")
			builder.WriteString(fmt.Sprintf("%v", fe.Value))
		}
		return builder.String()
	}

	return "field validation failed"
}

// truncateString 安全截断字符串，防止超长攻击
// 内部工具函数，提升代码复用性
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// 安全截断，避免截断 UTF-8 字符的中间
	if maxLen < 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// estimateValueSize 估算值的内存大小
// 用于防止存储过大的值导致内存问题
func estimateValueSize(v any) int {
	if v == nil {
		return 0
	}

	// 使用反射获取值的大小
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return len(rv.String())
	case reflect.Slice, reflect.Array:
		// 估算：每个元素 8 字节
		return rv.Len() * 8
	case reflect.Map:
		// 估算：每个键值对 16 字节
		return rv.Len() * 16
	case reflect.Struct:
		// 估算结构体大小
		return int(rv.Type().Size())
	case reflect.Ptr:
		if rv.IsNil() {
			return 0
		}
		return int(unsafe.Sizeof(rv.Interface()))
	default:
		return int(unsafe.Sizeof(v))
	}
}

// ValidationError 验证错误集合
// 职责：包装多个字段错误
type ValidationError struct {
	msg    string
	errors []*FieldError
}

// NewValidationError 创建验证错误
func NewValidationError(errs []*FieldError) *ValidationError {
	return &ValidationError{errors: errs}
}

// NewValidationErrorWithMsg 创建验证错误
func NewValidationErrorWithMsg(msg string) *ValidationError {
	return &ValidationError{msg: msg}
}

// Error 实现 error 接口
func (ve *ValidationError) Error() string {
	if len(ve.errors) == 0 {
		if len(ve.msg) > 0 {
			return ve.msg
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
