package v2

import (
	"fmt"
	"math"
	"strings"
)

// ============================================================================
// Map 验证器 - 用于验证 map[string]any 类型的动态字段
// ============================================================================

// MapValidator Map 验证器
// 设计原则：
//   - 单一职责：只负责 map 类型的验证
//   - 开放封闭：通过配置扩展验证规则
//   - 流式接口：支持链式调用配置
type MapValidator struct {
	namespace     string
	requiredKeys  []string
	allowedKeys   []string
	keyValidators map[string]func(value any) error
}

// NewMapValidator 创建 Map 验证器 - 工厂方法
func NewMapValidator() *MapValidator {
	return &MapValidator{
		requiredKeys:  make([]string, 0),
		allowedKeys:   make([]string, 0),
		keyValidators: make(map[string]func(value any) error),
	}
}

// WithNamespace 设置命名空间 - 链式调用
func (m *MapValidator) WithNamespace(namespace string) *MapValidator {
	m.namespace = namespace
	return m
}

// WithRequiredKeys 设置必填键 - 链式调用
func (m *MapValidator) WithRequiredKeys(keys ...string) *MapValidator {
	m.requiredKeys = keys
	return m
}

// WithAllowedKeys 设置允许的键（白名单）- 链式调用
func (m *MapValidator) WithAllowedKeys(keys ...string) *MapValidator {
	m.allowedKeys = keys
	return m
}

// WithKeyValidator 添加键验证器 - 链式调用
func (m *MapValidator) WithKeyValidator(key string, validator func(value any) error) *MapValidator {
	if key != "" && validator != nil {
		m.keyValidators[key] = validator
	}
	return m
}

// Validate 验证 map 数据
func (m *MapValidator) Validate(data map[string]any) []*FieldError {
	if data == nil {
		if len(m.requiredKeys) > 0 {
			return []*FieldError{
				NewFieldError(m.namespace, "map", "required", "").
					WithMessage("map cannot be nil when required keys are specified"),
			}
		}
		return nil
	}

	var errors []*FieldError

	// 1. 验证必填键
	errors = append(errors, m.validateRequiredKeys(data)...)

	// 2. 验证允许的键（白名单）
	if len(m.allowedKeys) > 0 {
		errors = append(errors, m.validateAllowedKeys(data)...)
	}

	// 3. 执行自定义键验证
	errors = append(errors, m.validateCustomKeys(data)...)

	return errors
}

// validateRequiredKeys 验证必填键
func (m *MapValidator) validateRequiredKeys(data map[string]any) []*FieldError {
	var errors []*FieldError

	for _, key := range m.requiredKeys {
		if _, exists := data[key]; !exists {
			errors = append(errors, NewFieldError(
				m.getFieldPath(key),
				key,
				"required",
				"",
			).WithMessage(fmt.Sprintf("required key '%s' is missing", key)))
		}
	}

	return errors
}

// validateAllowedKeys 验证允许的键（白名单）
func (m *MapValidator) validateAllowedKeys(data map[string]any) []*FieldError {
	var errors []*FieldError

	// 构建允许键的 map（快速查找）
	allowedMap := make(map[string]bool, len(m.allowedKeys))
	for _, key := range m.allowedKeys {
		allowedMap[key] = true
	}

	// 检查每个键是否在白名单中
	for key := range data {
		if !allowedMap[key] {
			errors = append(errors, NewFieldError(
				m.getFieldPath(key),
				key,
				"not_allowed",
				"",
			).WithMessage(fmt.Sprintf("key '%s' is not in the allowed list", key)))
		}
	}

	return errors
}

// validateCustomKeys 执行自定义键验证
func (m *MapValidator) validateCustomKeys(data map[string]any) []*FieldError {
	var errors []*FieldError

	for key, validator := range m.keyValidators {
		value, exists := data[key]
		if !exists {
			continue // 键不存在时不验证
		}

		// 执行自定义验证（带 panic 恢复）
		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("validator panicked: %v", r)
				}
			}()
			return validator(value)
		}()

		if err != nil {
			errors = append(errors, NewFieldError(
				m.getFieldPath(key),
				key,
				"custom",
				"",
			).WithMessage(err.Error()))
		}
	}

	return errors
}

// getFieldPath 获取字段路径
func (m *MapValidator) getFieldPath(key string) string {
	if m.namespace == "" {
		return key
	}
	return m.namespace + "." + key
}

// ============================================================================
// 便捷验证函数
// ============================================================================

// ValidateMapRequired 验证 map 必填键
func ValidateMapRequired(data map[string]any, keys ...string) error {
	if data == nil {
		return fmt.Errorf("map cannot be nil")
	}

	var missingKeys []string
	for _, key := range keys {
		if _, exists := data[key]; !exists {
			missingKeys = append(missingKeys, key)
		}
	}

	if len(missingKeys) > 0 {
		return fmt.Errorf("missing required keys: %s", strings.Join(missingKeys, ", "))
	}

	return nil
}

// ValidateMapString 验证 map 中的字符串键
func ValidateMapString(data map[string]any, key string, minLen, maxLen int) error {
	if data == nil {
		return nil
	}

	value, exists := data[key]
	if !exists {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("key '%s' must be string type, got %T", key, value)
	}

	length := len(str)
	if minLen > 0 && length < minLen {
		return fmt.Errorf("key '%s' length must be at least %d, got %d", key, minLen, length)
	}

	if maxLen > 0 && length > maxLen {
		return fmt.Errorf("key '%s' length must be at most %d, got %d", key, maxLen, length)
	}

	return nil
}

// ValidateMapInt 验证 map 中的整数键
func ValidateMapInt(data map[string]any, key string, min, max int) error {
	if data == nil {
		return nil
	}

	value, exists := data[key]
	if !exists {
		return nil
	}

	// 尝试转换为整数
	var intValue int
	switch v := value.(type) {
	case int:
		intValue = v
	case int64:
		if v > int64(math.MaxInt) || v < int64(math.MinInt) {
			return fmt.Errorf("key '%s' value %d overflows int type", key, v)
		}
		intValue = int(v)
	case int32:
		intValue = int(v)
	case float64:
		if v != float64(int(v)) {
			return fmt.Errorf("key '%s' value %f is not an integer", key, v)
		}
		intValue = int(v)
	default:
		return fmt.Errorf("key '%s' must be integer type, got %T", key, value)
	}

	if intValue < min {
		return fmt.Errorf("key '%s' value must be at least %d, got %d", key, min, intValue)
	}

	if intValue > max {
		return fmt.Errorf("key '%s' value must be at most %d, got %d", key, max, intValue)
	}

	return nil
}

// ValidateMapFloat 验证 map 中的浮点数键
func ValidateMapFloat(data map[string]any, key string, min, max float64) error {
	if data == nil {
		return nil
	}

	value, exists := data[key]
	if !exists {
		return nil
	}

	// 尝试转换为浮点数
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
		return fmt.Errorf("key '%s' must be numeric type, got %T", key, value)
	}

	if math.IsNaN(floatValue) {
		return fmt.Errorf("key '%s' value cannot be NaN", key)
	}

	if math.IsInf(floatValue, 0) {
		return fmt.Errorf("key '%s' value cannot be Inf", key)
	}

	if floatValue < min {
		return fmt.Errorf("key '%s' value must be at least %f, got %f", key, min, floatValue)
	}

	if floatValue > max {
		return fmt.Errorf("key '%s' value must be at most %f, got %f", key, max, floatValue)
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
		return nil
	}

	if _, ok := value.(bool); !ok {
		return fmt.Errorf("key '%s' must be bool type, got %T", key, value)
	}

	return nil
}

// ValidateMapKey 自定义键验证
func ValidateMapKey(data map[string]any, key string, validator func(value any) error) error {
	if data == nil {
		return nil
	}

	if validator == nil {
		return fmt.Errorf("validator cannot be nil")
	}

	value, exists := data[key]
	if !exists {
		return nil
	}

	return validator(value)
}
