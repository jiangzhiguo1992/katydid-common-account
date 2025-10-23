package v2

import (
	"fmt"
	"strings"
)

// ============================================================================
// 基础类型定义
// ============================================================================

// Scene 验证场景类型 - 使用位掩码支持组合场景
type Scene uint32

// 预定义的通用验证场景
const (
	SceneCreate Scene = 1 << iota // 创建场景 (1)
	SceneUpdate                   // 更新场景 (2)
	SceneDelete                   // 删除场景 (4)
	SceneQuery                    // 查询场景 (8)
	SceneList                     // 列表场景 (16)
	SceneImport                   // 导入场景 (32)
	SceneExport                   // 导出场景 (64)
	SceneBatch                    // 批量操作场景 (128)
)

// Has 检查是否包含指定场景
func (s Scene) Has(scene Scene) bool {
	return s&scene != 0
}

// Add 添加场景
func (s Scene) Add(scene Scene) Scene {
	return s | scene
}

// Remove 移除场景
func (s Scene) Remove(scene Scene) Scene {
	return s &^ scene
}

// String 场景名称
func (s Scene) String() string {
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
	if s.Has(SceneList) {
		scenes = append(scenes, "List")
	}
	if s.Has(SceneImport) {
		scenes = append(scenes, "Import")
	}
	if s.Has(SceneExport) {
		scenes = append(scenes, "Export")
	}
	if s.Has(SceneBatch) {
		scenes = append(scenes, "Batch")
	}
	if len(scenes) == 0 {
		return "Unknown"
	}
	return strings.Join(scenes, "|")
}

// ============================================================================
// 错误类型定义
// ============================================================================

// ValidationError 单个验证错误
type ValidationError struct {
	Field   string      `json:"field"`           // 字段名
	Tag     string      `json:"tag"`             // 验证标签
	Param   string      `json:"param"`           // 参数
	Message string      `json:"message"`         // 错误消息
	Value   interface{} `json:"value,omitempty"` // 字段值（可选）
}

// Error 实现 error 接口
func (e ValidationError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("字段 '%s' 验证失败: %s", e.Field, e.Tag)
}

// ValidationErrors 验证错误集合
type ValidationErrors []ValidationError

// Error 实现 error 接口
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// HasErrors 是否有错误
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// GetFieldErrors 获取指定字段的错误
func (e ValidationErrors) GetFieldErrors(field string) []ValidationError {
	var errors []ValidationError
	for _, err := range e {
		if err.Field == field {
			errors = append(errors, err)
		}
	}
	return errors
}

// ToMap 转换为 map 格式（字段 -> 错误消息列表）
func (e ValidationErrors) ToMap() map[string][]string {
	result := make(map[string][]string)
	for _, err := range e {
		result[err.Field] = append(result[err.Field], err.Message)
	}
	return result
}

// First 获取第一个错误
func (e ValidationErrors) First() *ValidationError {
	if len(e) == 0 {
		return nil
	}
	return &e[0]
}

// ============================================================================
// 验证选项
// ============================================================================

// ValidateOptions 验证选项
type ValidateOptions struct {
	Scene          Scene             // 验证场景
	PartialFields  []string          // 部分字段验证
	SkipCustom     bool              // 跳过自定义验证
	FailFast       bool              // 快速失败（遇到第一个错误就停止）
	UseCache       bool              // 使用缓存
	CustomMessages map[string]string // 自定义消息覆盖
}

// DefaultValidateOptions 默认验证选项
func DefaultValidateOptions() *ValidateOptions {
	return &ValidateOptions{
		Scene:      SceneCreate,
		SkipCustom: false,
		FailFast:   false,
		UseCache:   true,
	}
}

// ============================================================================
// 规则定义
// ============================================================================

// FieldRule 字段规则
type FieldRule struct {
	Field string // 字段名
	Rule  string // 验证规则
}

// SceneRules 场景规则集合
type SceneRules map[Scene]map[string]string

// Get 获取指定场景的规则
func (sr SceneRules) Get(scene Scene) map[string]string {
	// 精确匹配
	if rules, ok := sr[scene]; ok {
		return rules
	}

	// 组合场景匹配：查找包含该场景的组合
	for s, rules := range sr {
		if s.Has(scene) {
			return rules
		}
	}

	return nil
}

// Set 设置场景规则
func (sr SceneRules) Set(scene Scene, rules map[string]string) {
	sr[scene] = rules
}

// Merge 合并场景规则
func (sr SceneRules) Merge(other SceneRules) {
	for scene, rules := range other {
		if existing, ok := sr[scene]; ok {
			// 合并规则
			for field, rule := range rules {
				existing[field] = rule
			}
		} else {
			sr[scene] = rules
		}
	}
}
