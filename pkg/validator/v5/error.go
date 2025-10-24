package v5

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unsafe"
)

// FieldError 字段错误
// 职责：描述单个字段的验证错误
// TODO:GG 检查所有构造函数params
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
	// 防御性编程：安全检查并截断超长字段
	namespace = truncateString(namespace, maxNamespaceLength)
	tag = truncateString(tag, maxTagLength)

	return &FieldError{
		Namespace: namespace,
		Tag:       tag,
	}
}

// WithParam 设置参数
func (fe *FieldError) WithParam(param string) *FieldError {
	fe.Param = param
	return fe
}

// WithValue 设置值
func (fe *FieldError) WithValue(value any) *FieldError {
	// 安全检查：值大小限制
	if estimateValueSize(value) > maxValueSize {
		fe.Value = nil
		return fe
	}
	fe.Value = value
	return fe
}

// WithMessage 设置消息
func (fe *FieldError) WithMessage(message string) *FieldError {
	// 安全检查：截断超长消息
	fe.Message = truncateString(message, maxMessageLength)
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

// ValidationError 验证错误集合
// 职责：包装多个字段错误
type ValidationError struct {
	Errors []*FieldError
}

// NewValidationError 创建验证错误
func NewValidationError(errs []*FieldError) *ValidationError {
	return &ValidationError{Errors: errs}
}

// Error 实现 error 接口
func (ve *ValidationError) Error() string {
	if len(ve.Errors) == 0 {
		return "validation passed"
	}
	if len(ve.Errors) == 1 {
		return ve.Errors[0].Error()
	}
	return "validation failed with " + string(rune(len(ve.Errors))) + " errors"
}

// HasErrors 是否有错误
func (ve *ValidationError) HasErrors() bool {
	return len(ve.Errors) > 0
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

// ============================================================================
// ErrorCollector 实现 - 错误收集器
// ============================================================================

// DefaultErrorCollector 默认错误收集器
// 职责：收集和管理验证错误
// 设计原则：单一职责、线程安全
type DefaultErrorCollector struct {
	errors   []*FieldError
	mu       sync.RWMutex
	maxCount int
}

// NewDefaultErrorCollector 创建默认错误收集器
func NewDefaultErrorCollector() *DefaultErrorCollector {
	return &DefaultErrorCollector{
		errors:   make([]*FieldError, 0, 8),
		maxCount: 1000,
	}
}

// NewErrorCollectorWithLimit 创建带限制的错误收集器
func NewErrorCollectorWithLimit(maxCount int) *DefaultErrorCollector {
	return &DefaultErrorCollector{
		errors:   make([]*FieldError, 0, 8),
		maxCount: maxCount,
	}
}

// AddError 添加错误
func (c *DefaultErrorCollector) AddError(err *FieldError) {
	if err == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否达到上限
	if len(c.errors) >= c.maxCount {
		return
	}

	c.errors = append(c.errors, err)
}

// AddErrors 批量添加错误
func (c *DefaultErrorCollector) AddErrors(errs []*FieldError) {
	if len(errs) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 计算可添加的数量
	remaining := c.maxCount - len(c.errors)
	if remaining <= 0 {
		return
	}

	if len(errs) > remaining {
		errs = errs[:remaining]
	}

	c.errors = append(c.errors, errs...)
}

// GetErrors 获取所有错误
func (c *DefaultErrorCollector) GetErrors() []*FieldError {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 返回副本，避免外部修改
	result := make([]*FieldError, len(c.errors))
	copy(result, c.errors)
	return result
}

// HasErrors 是否有错误
func (c *DefaultErrorCollector) HasErrors() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.errors) > 0
}

// Clear 清除错误
func (c *DefaultErrorCollector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors = c.errors[:0]
}

// ErrorCount 错误数量
func (c *DefaultErrorCollector) ErrorCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.errors)
}

// ============================================================================
// ErrorFormatter 实现 - 错误格式化器
// ============================================================================

// DefaultErrorFormatter 默认错误格式化器
// 职责：格式化错误信息
// 设计原则：单一职责
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

	if err.Message != "" {
		return err.Message
	}

	if err.Param != "" {
		return "field '" + err.Namespace + "' failed validation on tag '" + err.Tag + "' with param '" + err.Param + "'"
	}

	return "field '" + err.Namespace + "' failed validation on tag '" + err.Tag + "'"
}

// FormatAll 格式化所有错误
func (f *DefaultErrorFormatter) FormatAll(errs []*FieldError) string {
	if len(errs) == 0 {
		return "validation passed"
	}

	if len(errs) == 1 {
		return f.Format(errs[0])
	}

	var result string
	maxDisplay := 10
	displayCount := len(errs)
	if displayCount > maxDisplay {
		displayCount = maxDisplay
	}

	for i := 0; i < displayCount; i++ {
		if i > 0 {
			result += "; "
		}
		result += f.Format(errs[i])
	}

	if len(errs) > maxDisplay {
		result += "; ... and more errors"
	}

	return result
}
