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

// 常用场景定义
const (
	SceneCreate Scene = 1 << iota // 创建场景 (1)
	SceneUpdate                   // 更新场景 (2)
	SceneDelete                   // 删除场景 (4)
	SceneQuery                    // 查询场景 (8)
)

// Has 检查是否包含指定场景
func (s Scene) Has(scene Scene) bool {
	if s == SceneAll || scene == SceneAll {
		return true
	}
	return s&scene != 0
}

// String 场景名称
func (s Scene) String() string {
	if s == SceneNone {
		return "None"
	}
	if s == SceneAll {
		return "All"
	}

	var scenes []string
	if s.Has(SceneCreate) {
		scenes = append(scenes, "Create")
	}
	if s.Has(SceneUpdate) {
		scenes = append(scenes, "Update")
	}
	if s.Has(SceneDelete) {
		scenes = append(scenes, "Delete")
	}
	if s.Has(SceneQuery) {
		scenes = append(scenes, "Query")
	}

	if len(scenes) == 0 {
		return "Unknown"
	}
	return strings.Join(scenes, "|")
}

// ============================================================================
// 错误类型定义
// ============================================================================

// FieldError 字段验证错误
type FieldError struct {
	Field   string      // 字段名称
	Tag     string      // 验证标签
	Param   string      // 验证参数
	Value   interface{} // 字段值
	Message string      // 错误消息
}

// Error 实现 error 接口
func (e *FieldError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Param != "" {
		return fmt.Sprintf("字段 '%s' 验证失败: %s=%s", e.Field, e.Tag, e.Param)
	}
	return fmt.Sprintf("字段 '%s' 验证失败: %s", e.Field, e.Tag)
}

// WithMessage 设置自定义错误消息
func (e *FieldError) WithMessage(msg string) *FieldError {
	e.Message = msg
	return e
}

// NewFieldError 创建字段错误
func NewFieldError(field, tag, param string) *FieldError {
	return &FieldError{
		Field: field,
		Tag:   tag,
		Param: param,
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
