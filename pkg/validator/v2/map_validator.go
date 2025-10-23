package v2

import (
	"fmt"
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

// GetNamespace 获取命名空间 - 实现 MapValidatorConfig 接口
func (m *MapValidator) GetNamespace() string {
	return m.namespace
}

// GetRequiredKeys 获取必填键列表 - 实现 MapValidatorConfig 接口
func (m *MapValidator) GetRequiredKeys() []string {
	return m.requiredKeys
}

// GetAllowedKeys 获取允许的键列表 - 实现 MapValidatorConfig 接口
func (m *MapValidator) GetAllowedKeys() []string {
	return m.allowedKeys
}

// GetKeyValidator 获取指定键的验证器 - 实现 MapValidatorConfig 接口
func (m *MapValidator) GetKeyValidator(key string) func(value any) error {
	return m.keyValidators[key]
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
