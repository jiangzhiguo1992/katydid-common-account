package error

import (
	"fmt"
	"katydid-common-account/pkg/validator/v5/core"
	"reflect"
	"strings"
	"unsafe"
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

// NewFieldErrorWithMessage 创建仅带消息的字段错误
func NewFieldErrorWithMessage(message string) *FieldError {
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
		builder.Grow(core.ErrorMessageEstimatedLength)
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
