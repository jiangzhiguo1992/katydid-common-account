package v2

import (
	"fmt"
	"strings"
)

// ============================================================================
// 场景类型定义
// ============================================================================

// Scene 验证场景类型 - 使用位掩码支持组合场景
type Scene int64

// 预定义的通用验证场景
const (
	SceneNone Scene = 0  // 无场景
	SceneAll  Scene = -1 // 所有场景
)

// Has 检查是否包含指定场景
func (s Scene) Has(scene Scene) bool {
	if s == SceneAll || scene == SceneAll {
		return true
	}
	return s&scene != 0
}

// ============================================================================
// 错误类型定义
// ============================================================================

// FieldError 字段验证错误
type FieldError struct {
	Namespace string      `json:"namespace"`         // 字段的完整命名空间路径（如 User.Profile.Email）
	Field     string      `json:"field"`             // 字段名称
	Tag       string      `json:"tag"`               // 验证标签
	Param     string      `json:"param"`             // 验证参数
	Value     interface{} `json:"value,omitempty"`   // 字段值
	Message   string      `json:"message,omitempty"` // 错误消息
}

// Error 实现 error 接口
func (e *FieldError) Error() string {
	if e.Message != "" {
		return e.Message
	}

	// 优先使用 Namespace，如果没有则使用 Field
	fieldName := e.Namespace
	if fieldName == "" {
		fieldName = e.Field
	}

	if e.Param != "" {
		return fmt.Sprintf("字段 '%s' 验证失败: %s=%s", fieldName, e.Tag, e.Param)
	}
	return fmt.Sprintf("字段 '%s' 验证失败: %s", fieldName, e.Tag)
}

// WithMessage 设置自定义错误消息
func (e *FieldError) WithMessage(msg string) *FieldError {
	e.Message = msg
	return e
}

// NewFieldError 创建字段错误
func NewFieldError(namespace, tag, param string) *FieldError {
	// 从 namespace 中提取 field 名称（取最后一个点后面的部分）
	field := namespace
	if idx := strings.LastIndex(namespace, "."); idx >= 0 {
		field = namespace[idx+1:]
	}

	return &FieldError{
		Namespace: namespace,
		Field:     field,
		Tag:       tag,
		Param:     param,
	}
}

// ValidationErrors 验证错误集合
type ValidationErrors []*FieldError

// Error 实现 error 接口
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "validation failed"
	}

	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// HasErrors 检查是否有错误
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// ============================================================================
// 常量定义
// ============================================================================

const (
	// MaxNestedDepth 最大嵌套验证深度
	MaxNestedDepth = 100

	// MaxValidationErrors 单次验证最大错误数
	MaxValidationErrors = 1000
)
