package v2

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// ============================================================================
// 核心数据类型 - 值对象（Value Objects）
// ============================================================================

// FieldError 字段验证错误 - 不可变的值对象
// 设计原则：单一职责 - 只负责表示一个字段的验证错误信息
type FieldError struct {
	// Namespace 字段的完整路径（如 User.Profile.Email）
	Namespace string `json:"namespace"`

	// Field 字段名（简化版，不含路径）
	Field string `json:"field"`

	// Tag 验证标签（如 required, email, min）
	Tag string `json:"tag"`

	// Param 验证参数（如 min=3 中的 "3"）
	Param string `json:"param"`

	// Value 字段的实际值（可选）
	Value any `json:"value,omitempty"`

	// Message 用户友好的错误消息
	Message string `json:"message,omitempty"`
}

// TypeInfo 类型信息 - 缓存的类型元数据
// 设计原则：单一职责 - 只负责存储类型的元数据信息
type TypeInfo struct {
	// Type 反射类型
	Type reflect.Type

	// IsRuleProvider 是否实现了 RuleProvider 接口
	IsRuleProvider bool

	// IsCustomValidator 是否实现了 CustomValidator 接口
	IsCustomValidator bool

	// Rules 缓存的验证规则
	Rules map[Scene]FieldRules
}

// ValidationResult 验证结果实现 - 实现 Result 接口
// 设计原则：单一职责 - 只负责存储和查询验证结果
type ValidationResult struct {
	errors []*FieldError
}

// ============================================================================
// FieldError 方法实现
// ============================================================================

// NewFieldError 创建字段错误 - 工厂方法
// 参数验证和边界检查，确保数据安全
func NewFieldError(namespace, field, tag, param string) *FieldError {
	return &FieldError{
		Namespace: truncate(namespace, 512),
		Field:     truncate(field, 256),
		Tag:       truncate(tag, 128),
		Param:     truncate(param, 256),
	}
}

// WithValue 设置字段值 - 链式调用（流式接口）
func (e *FieldError) WithValue(value any) *FieldError {
	e.Value = value
	return e
}

// WithMessage 设置错误消息 - 链式调用（流式接口）
func (e *FieldError) WithMessage(message string) *FieldError {
	e.Message = truncate(message, 2048)
	return e
}

// Error 实现 error 接口
func (e *FieldError) Error() string {
	if e.Message != "" {
		return e.Message
	}

	if e.Param != "" {
		return fmt.Sprintf("field '%s' validation failed on tag '%s' with param '%s'",
			e.Field, e.Tag, e.Param)
	}

	return fmt.Sprintf("field '%s' validation failed on tag '%s'", e.Field, e.Tag)
}

// String 返回友好的字符串表示
func (e *FieldError) String() string {
	return e.Error()
}

// ToJSON 转换为 JSON 格式
func (e *FieldError) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// ============================================================================
// TypeInfo 方法实现
// ============================================================================

// NewTypeInfo 创建类型信息 - 工厂方法
func NewTypeInfo(typ reflect.Type) *TypeInfo {
	return &TypeInfo{
		Type:              typ,
		IsRuleProvider:    false,
		IsCustomValidator: false,
		Rules:             make(map[Scene]FieldRules),
	}
}

// ============================================================================
// ValidationResult 方法实现 - 实现 Result 接口
// ============================================================================

// NewValidationResult 创建验证结果 - 工厂方法
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		errors: make([]*FieldError, 0),
	}
}

// NewValidationResultWithErrors 从错误列表创建验证结果
func NewValidationResultWithErrors(errors []*FieldError) *ValidationResult {
	if errors == nil {
		errors = make([]*FieldError, 0)
	}
	return &ValidationResult{
		errors: errors,
	}
}

// IsValid 验证是否通过
func (r *ValidationResult) IsValid() bool {
	return len(r.errors) == 0
}

// Errors 获取所有错误
func (r *ValidationResult) Errors() []*FieldError {
	// 返回副本，防止外部修改
	result := make([]*FieldError, len(r.errors))
	copy(result, r.errors)
	return result
}

// FirstError 获取第一个错误
func (r *ValidationResult) FirstError() *FieldError {
	if len(r.errors) == 0 {
		return nil
	}
	return r.errors[0]
}

// ErrorsByField 获取指定字段的错误
func (r *ValidationResult) ErrorsByField(field string) []*FieldError {
	var result []*FieldError
	for _, err := range r.errors {
		if err.Field == field {
			result = append(result, err)
		}
	}
	return result
}

// ErrorsByTag 获取指定标签的错误
func (r *ValidationResult) ErrorsByTag(tag string) []*FieldError {
	var result []*FieldError
	for _, err := range r.errors {
		if err.Tag == tag {
			result = append(result, err)
		}
	}
	return result
}

// Error 实现 error 接口
func (r *ValidationResult) Error() string {
	if len(r.errors) == 0 {
		return "validation passed"
	}

	if len(r.errors) == 1 {
		return r.errors[0].Error()
	}

	return fmt.Sprintf("validation failed with %d errors: %s",
		len(r.errors), r.errors[0].Error())
}

// ToJSON 转换为 JSON 格式
func (r *ValidationResult) ToJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"valid":  r.IsValid(),
		"errors": r.errors,
	})
}

// ============================================================================
// 辅助函数
// ============================================================================

// truncate 截断字符串到指定长度 - 防止超长攻击
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
