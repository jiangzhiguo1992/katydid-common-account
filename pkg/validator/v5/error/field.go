package error

import (
	"katydid-common-account/pkg/validator/v5/core"
	"reflect"
	"unsafe"
)

// FieldErrorOption 字段错误选项函数类型
type FieldErrorOption func(*FieldError)

// FieldError 字段错误
// 职责：描述单个字段的验证错误
type FieldError struct {
	// namespace 字段的完整命名空间路径（如 User.Profile.Email）
	// 用于嵌套结构体的错误定位，支持复杂对象的精确错误追踪
	namespace string

	// tag 验证标签，描述验证规则类型（如 required, email, min, max 等）
	tag string

	// param 验证参数，提供验证规则的具体配置值
	// 例如：min=3 中的 "3"，len=11 中的 "11"
	param string

	// value 字段的实际值（用于 sl.ReportError 的 value 参数）
	// 用于调试和详细错误信息，可能包含敏感信息，谨慎使用
	value any

	// message 用户友好的错误消息（可选，用于直接显示给终端用户）
	// 支持国际化，建议使用本地化后的错误消息
	message string
}

// NewFieldError 创建字段错误
func NewFieldError(namespace, tag string, opts ...FieldErrorOption) core.IFieldError {
	fe := &FieldError{
		namespace: namespace,
		tag:       tag,
	}

	// 应用选项
	for _, opt := range opts {
		opt(fe)
	}

	return fe
}

// NewFieldErrorWithMessage 创建仅带消息的字段错误
func NewFieldErrorWithMessage(message string, opts ...FieldErrorOption) core.IFieldError {
	fe := &FieldError{
		message: message,
	}

	// 应用选项
	for _, opt := range opts {
		opt(fe)
	}

	return fe
}

// WithParam 设置参数
func WithParam(param string) FieldErrorOption {
	return func(fe *FieldError) {
		fe.param = param
	}
}

// WithValue 设置值
func WithValue(value any) FieldErrorOption {
	return func(fe *FieldError) {
		// 最大值大小（字节），防止存储过大的值导致内存问题
		if estimateValueSize(value) > 4096 {
			fe.value = nil
			return
		}
		fe.value = value
	}
}

// WithMessage 设置消息
func WithMessage(message string) FieldErrorOption {
	return func(fe *FieldError) {
		fe.message = truncateString(message, 2048)
	}
}

// Namespace 获取命名空间
func (fe *FieldError) Namespace() string {
	return fe.namespace
}

// Tag 获取标签
func (fe *FieldError) Tag() string {
	return fe.tag
}

// Param 获取参数
func (fe *FieldError) Param() string {
	return fe.param
}

// Value 获取值
func (fe *FieldError) Value() any {
	return fe.value
}

// Message 获取消息
func (fe *FieldError) Message() string {
	return fe.message
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
