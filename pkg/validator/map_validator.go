package validator

import (
	"fmt"
	"strings"
	"sync"
)

// MapValidator Map 验证器，用于验证 map[string]any 类型的扩展字段
type MapValidator struct {
	// RequiredKeys 必填的键
	RequiredKeys []string
	// AllowedKeys 允许的键列表（如果为空则不限制）
	AllowedKeys []string
	// KeyValidators 特定键的验证函数
	KeyValidators map[string]func(value interface{}) error
	// allowedKeysMap 内部缓存的允许键 map（优化查找性能）
	allowedKeysMap map[string]bool
	// mu 保护 allowedKeysMap 的并发访问
	mu sync.RWMutex
}

// ValidateMap 验证 map[string]any 类型的扩展字段
func ValidateMap(kvs map[string]any, v *MapValidator) error {
	if v == nil {
		return nil
	}

	// 1. 验证必填键
	if len(v.RequiredKeys) > 0 {
		for _, key := range v.RequiredKeys {
			if _, exists := kvs[key]; !exists {
				return fmt.Errorf("map 缺少必填键: %s", key)
			}
		}
	}

	// 2. 验证允许的键（使用缓存的 map 提高查找性能）
	if len(v.AllowedKeys) > 0 {
		// 使用读锁检查缓存是否存在
		v.mu.RLock()
		allowedMap := v.allowedKeysMap
		v.mu.RUnlock()

		// 如果缓存不存在，创建缓存
		if allowedMap == nil {
			v.mu.Lock()
			// 双重检查
			if v.allowedKeysMap == nil {
				v.allowedKeysMap = make(map[string]bool, len(v.AllowedKeys))
				for _, key := range v.AllowedKeys {
					v.allowedKeysMap[key] = true
				}
			}
			allowedMap = v.allowedKeysMap
			v.mu.Unlock()
		}

		for key := range kvs {
			if !allowedMap[key] {
				return fmt.Errorf("map 包含不允许的键: %s", key)
			}
		}
	}

	// 3. 执行自定义键验证器
	if len(v.KeyValidators) > 0 {
		for key, validatorFunc := range v.KeyValidators {
			if value, exists := kvs[key]; exists {
				if err := validatorFunc(value); err != nil {
					return fmt.Errorf("map 键 '%s' 验证失败: %w", key, err)
				}
			}
		}
	}

	return nil
}

// ValidateMapKey 验证 map[string]any 中特定键的值
func ValidateMapKey(kvs map[string]any, key string, validatorFunc func(value interface{}) error) error {
	value, exists := kvs[key]
	if !exists {
		return nil // 键不存在时不验证
	}
	return validatorFunc(value)
}

// ValidateMapMustHaveKey 验证 map[string]any 必须包含指定的键
func ValidateMapMustHaveKey(kvs map[string]any, key string) error {
	if _, exists := kvs[key]; !exists {
		return fmt.Errorf("map 必须包含键: %s", key)
	}
	return nil
}

// ValidateMapMustHaveKeys 验证 map[string]any 必须包含指定的多个键
func ValidateMapMustHaveKeys(kvs map[string]any, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	// 优化：收集所有缺失的键，一次性报告
	var missingKeys []string
	for _, key := range keys {
		if _, exists := kvs[key]; !exists {
			missingKeys = append(missingKeys, key)
		}
	}

	if len(missingKeys) > 0 {
		var builder strings.Builder
		builder.WriteString("map 缺少必填键: ")
		builder.WriteString(strings.Join(missingKeys, ", "))
		return fmt.Errorf("%s", builder.String())
	}

	return nil
}

// ValidateMapStringKey 验证 map[string]any 中字符串类型的键
func ValidateMapStringKey(kvs map[string]any, key string, minLen, maxLen int) error {
	value, exists := kvs[key]
	if !exists {
		return nil // 键不存在时不验证
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("键 '%s' 必须是字符串类型", key)
	}

	strLen := len(str)
	if minLen > 0 && strLen < minLen {
		return fmt.Errorf("键 '%s' 的值长度不能小于 %d", key, minLen)
	}

	if maxLen > 0 && strLen > maxLen {
		return fmt.Errorf("键 '%s' 的值长度不能大于 %d", key, maxLen)
	}

	return nil
}

// ValidateMapIntKey 验证 map[string]any 中整数类型的键
func ValidateMapIntKey(kvs map[string]any, key string, min, max int) error {
	value, exists := kvs[key]
	if !exists {
		return nil // 键不存在时不验证
	}

	// 尝试转换为整数
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
	case float32:
		intValue = int(v)
	default:
		return fmt.Errorf("键 '%s' 必须是整数类型", key)
	}

	if intValue < min {
		return fmt.Errorf("键 '%s' 的值不能小于 %d", key, min)
	}

	if intValue > max {
		return fmt.Errorf("键 '%s' 的值不能大于 %d", key, max)
	}

	return nil
}

// ValidateMapFloatKey 验证 map[string]any 中浮点数类型的键
func ValidateMapFloatKey(kvs map[string]any, key string, min, max float64) error {
	value, exists := kvs[key]
	if !exists {
		return nil
	}

	var floatValue float64
	switch v := value.(type) {
	case float64:
		floatValue = v
	case float32:
		floatValue = float64(v)
	case int:
		floatValue = float64(v)
	case int64:
		floatValue = float64(v)
	case int32:
		floatValue = float64(v)
	default:
		return fmt.Errorf("键 '%s' 必须是数字类型", key)
	}

	if floatValue < min {
		return fmt.Errorf("键 '%s' 的值不能小于 %f", key, min)
	}

	if floatValue > max {
		return fmt.Errorf("键 '%s' 的值不能大于 %f", key, max)
	}

	return nil
}

// ValidateMapBoolKey 验证 map[string]any 中布尔类型的键
func ValidateMapBoolKey(kvs map[string]any, key string) error {
	value, exists := kvs[key]
	if !exists {
		return nil
	}

	if _, ok := value.(bool); !ok {
		return fmt.Errorf("键 '%s' 必须是布尔类型", key)
	}

	return nil
}

// NewMapValidator 创建一个新的 MapValidator
func NewMapValidator() *MapValidator {
	return &MapValidator{
		RequiredKeys:  make([]string, 0),
		AllowedKeys:   make([]string, 0),
		KeyValidators: make(map[string]func(value interface{}) error),
	}
}

// WithRequiredKeys 设置必填键（链式调用）
func (mv *MapValidator) WithRequiredKeys(keys ...string) *MapValidator {
	mv.RequiredKeys = keys
	return mv
}

// WithAllowedKeys 设置允许的键（链式调用）
func (mv *MapValidator) WithAllowedKeys(keys ...string) *MapValidator {
	mv.mu.Lock()
	defer mv.mu.Unlock()
	mv.AllowedKeys = keys
	// 清除缓存，下次验证时重新构建
	mv.allowedKeysMap = nil
	return mv
}

// WithKeyValidator 添加键验证器（链式调用）
func (mv *MapValidator) WithKeyValidator(key string, validatorFunc func(value interface{}) error) *MapValidator {
	if mv.KeyValidators == nil {
		mv.KeyValidators = make(map[string]func(value interface{}) error)
	}
	mv.KeyValidators[key] = validatorFunc
	return mv
}

// AddRequiredKey 添加单个必填键
func (mv *MapValidator) AddRequiredKey(key string) *MapValidator {
	mv.RequiredKeys = append(mv.RequiredKeys, key)
	return mv
}

// AddAllowedKey 添加单个允许的键
func (mv *MapValidator) AddAllowedKey(key string) *MapValidator {
	mv.mu.Lock()
	defer mv.mu.Unlock()
	mv.AllowedKeys = append(mv.AllowedKeys, key)
	// 清除缓存，下次验证时重新构建
	mv.allowedKeysMap = nil
	return mv
}

// Validate 验证 map（方法形式，支持链式调用后直接验证）
func (mv *MapValidator) Validate(kvs map[string]any) error {
	return ValidateMap(kvs, mv)
}
