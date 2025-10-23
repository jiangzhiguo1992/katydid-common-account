package v2

import (
	"fmt"
)

// ============================================================================
// 场景化 Map 验证器 - 支持不同验证场景使用不同的验证规则
// 设计模式：策略模式（Strategy Pattern）+ 工厂模式（Factory Pattern）
// 设计原则：
//   - 单一职责原则（SRP）：只负责场景化的 Map 验证
//   - 开放封闭原则（OCP）：通过配置扩展场景，而不修改核心代码
// ============================================================================

// SceneMapValidators 场景化的 Map 验证器集合
// 线程安全性：只读操作，多个 goroutine 可以并发使用
type SceneMapValidators struct {
	validators map[Scene]MapValidatorConfig
}

// NewSceneMapValidators 创建场景化 Map 验证器 - 工厂方法
func NewSceneMapValidators() *SceneMapValidators {
	return &SceneMapValidators{
		validators: make(map[Scene]MapValidatorConfig),
	}
}

// WithScene 为指定场景添加验证器 - 链式调用
// 参数：
//   - scene: 验证场景
//   - validator: Map 验证器配置
func (s *SceneMapValidators) WithScene(scene Scene, validator MapValidatorConfig) *SceneMapValidators {
	if validator != nil {
		s.validators[scene] = validator
	}
	return s
}

// Validate 执行场景化验证
// 参数：
//   - scene: 验证场景
//   - data: 待验证的 map 数据
//
// 返回：验证错误列表
func (s *SceneMapValidators) Validate(scene Scene, data map[string]any) []*FieldError {
	if s.validators == nil || len(s.validators) == 0 {
		return nil
	}

	// 查找匹配的验证器
	validator, exists := s.validators[scene]
	if !exists || validator == nil {
		return nil
	}

	// 执行验证
	return validator.Validate(data)
}

// HasScene 检查是否存在指定场景的验证器
func (s *SceneMapValidators) HasScene(scene Scene) bool {
	_, exists := s.validators[scene]
	return exists
}

// GetValidator 获取指定场景的验证器
func (s *SceneMapValidators) GetValidator(scene Scene) MapValidatorConfig {
	return s.validators[scene]
}

// ============================================================================
// 便捷函数 - 简化常见的 Map 验证场景
// ============================================================================

// ValidateMapWithScene 场景化验证 map 字段
// 根据验证场景匹配相应的验证器并执行验证
//
// 参数：
//   - scene: 验证场景
//   - data: 待验证的 map 数据
//   - validators: 场景化的验证器集合
//
// 返回：验证错误列表，nil 表示验证通过
func ValidateMapWithScene(scene Scene, data map[string]any, validators *SceneMapValidators) []*FieldError {
	if validators == nil {
		return nil
	}
	return validators.Validate(scene, data)
}

// ============================================================================
// Map 验证辅助函数 - 提供常用的验证逻辑
// ============================================================================

// ValidateMapRequired 验证 map 必填键
// 检查指定的键是否都存在于 map 中
func ValidateMapRequired(data map[string]any, keys ...string) []*FieldError {
	if data == nil {
		if len(keys) > 0 {
			return []*FieldError{
				NewFieldError("map", "map", "required", "").
					WithMessage("map cannot be nil when required keys are specified"),
			}
		}
		return nil
	}

	var errors []*FieldError
	for _, key := range keys {
		if _, exists := data[key]; !exists {
			errors = append(errors, NewFieldError(
				"map."+key,
				key,
				"required",
				"",
			).WithMessage(fmt.Sprintf("required key '%s' is missing", key)))
		}
	}

	return errors
}

// ValidateMapString 验证 map 中的字符串键
// 检查字符串值的长度是否在指定范围内
func ValidateMapString(data map[string]any, key string, minLen, maxLen int) error {
	if data == nil {
		return nil
	}

	value, exists := data[key]
	if !exists {
		return nil // 键不存在时不验证
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("key '%s' is not a string", key)
	}

	if minLen > 0 && len(str) < minLen {
		return fmt.Errorf("key '%s' length must be at least %d", key, minLen)
	}

	if maxLen > 0 && len(str) > maxLen {
		return fmt.Errorf("key '%s' length must not exceed %d", key, maxLen)
	}

	return nil
}

// ValidateMapInt 验证 map 中的整数键
// 检查整数值是否在指定范围内
func ValidateMapInt(data map[string]any, key string, min, max int) error {
	if data == nil {
		return nil
	}

	value, exists := data[key]
	if !exists {
		return nil // 键不存在时不验证
	}

	// 支持多种整数类型
	var intValue int
	switch v := value.(type) {
	case int:
		intValue = v
	case int64:
		intValue = int(v)
	case int32:
		intValue = int(v)
	case float64:
		intValue = int(v)
	default:
		return fmt.Errorf("key '%s' is not an integer", key)
	}

	if min != 0 && intValue < min {
		return fmt.Errorf("key '%s' must be at least %d", key, min)
	}

	if max != 0 && intValue > max {
		return fmt.Errorf("key '%s' must not exceed %d", key, max)
	}

	return nil
}

// ValidateMapBool 验证 map 中的布尔键
func ValidateMapBool(data map[string]any, key string) error {
	if data == nil {
		return nil
	}

	value, exists := data[key]
	if !exists {
		return nil // 键不存在时不验证
	}

	if _, ok := value.(bool); !ok {
		return fmt.Errorf("key '%s' is not a boolean", key)
	}

	return nil
}

// ValidateMapEnum 验证 map 中的枚举键
// 检查值是否在允许的枚举值列表中
func ValidateMapEnum(data map[string]any, key string, allowedValues ...string) error {
	if data == nil {
		return nil
	}

	value, exists := data[key]
	if !exists {
		return nil // 键不存在时不验证
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("key '%s' is not a string", key)
	}

	// 检查是否在允许的值列表中
	for _, allowed := range allowedValues {
		if str == allowed {
			return nil
		}
	}

	return fmt.Errorf("key '%s' must be one of: %v", key, allowedValues)
}

// ============================================================================
// 常用验证器工厂方法
// ============================================================================

// StringValidator 创建字符串验证器
func StringValidator(minLen, maxLen int) func(value any) error {
	return func(value any) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("value is not a string")
		}

		if minLen > 0 && len(str) < minLen {
			return fmt.Errorf("length must be at least %d", minLen)
		}

		if maxLen > 0 && len(str) > maxLen {
			return fmt.Errorf("length must not exceed %d", maxLen)
		}

		return nil
	}
}

// IntValidator 创建整数验证器
func IntValidator(min, max int) func(value any) error {
	return func(value any) error {
		var intValue int
		switch v := value.(type) {
		case int:
			intValue = v
		case int64:
			intValue = int(v)
		case int32:
			intValue = int(v)
		case float64:
			intValue = int(v)
		default:
			return fmt.Errorf("value is not an integer")
		}

		if min != 0 && intValue < min {
			return fmt.Errorf("must be at least %d", min)
		}

		if max != 0 && intValue > max {
			return fmt.Errorf("must not exceed %d", max)
		}

		return nil
	}
}

// EnumValidator 创建枚举验证器
func EnumValidator(allowedValues ...string) func(value any) error {
	return func(value any) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("value is not a string")
		}

		for _, allowed := range allowedValues {
			if str == allowed {
				return nil
			}
		}

		return fmt.Errorf("must be one of: %v", allowedValues)
	}
}
