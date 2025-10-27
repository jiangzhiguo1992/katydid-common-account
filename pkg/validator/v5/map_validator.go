package v5

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// ============================================================================
// Map验证器 - 用于验证动态扩展字段
// ============================================================================

// MapValidator Map字段验证器
// 职责：专门验证 map[string]any 类型的动态字段
// 设计原则：单一职责、高内聚低耦合
type MapValidator struct {
	// parentNamespace 父级命名空间，用于生成准确的错误路径
	parentNamespace string

	// requiredKeys 必填键列表
	requiredKeys []string

	// allowedKeys 允许的键白名单（空则不限制）
	allowedKeys []string

	// keyValidators 自定义键验证器
	keyValidators map[string]func(value any) error

	// allowedKeysMap 缓存的允许键映射（性能优化）
	allowedKeysMap map[string]bool

	// initOnce 确保缓存只初始化一次
	initOnce sync.Once
}

// MapValidatorOption Map验证器选项
type MapValidatorOption func(*MapValidator)

// NewMapValidator 创建Map验证器
func NewMapValidator(parentNamespace string, opts ...MapValidatorOption) *MapValidator {
	mv := &MapValidator{
		parentNamespace: parentNamespace,
		requiredKeys:    make([]string, 0),
		allowedKeys:     make([]string, 0),
		keyValidators:   make(map[string]func(value any) error),
	}

	for _, opt := range opts {
		opt(mv)
	}

	return mv
}

// WithRequiredKeys 设置必填键
func WithRequiredKeys(keys ...string) MapValidatorOption {
	return func(mv *MapValidator) {
		mv.requiredKeys = append(mv.requiredKeys, keys...)
	}
}

// WithAllowedKeys 设置允许的键白名单
func WithAllowedKeys(keys ...string) MapValidatorOption {
	return func(mv *MapValidator) {
		mv.allowedKeys = append(mv.allowedKeys, keys...)
	}
}

// WithKeyValidator 添加自定义键验证器
func WithKeyValidator(key string, validator func(value any) error) MapValidatorOption {
	return func(mv *MapValidator) {
		if mv.keyValidators == nil {
			mv.keyValidators = make(map[string]func(value any) error)
		}
		mv.keyValidators[key] = validator
	}
}

// Validate 验证Map字段
func (mv *MapValidator) Validate(data map[string]any, ctx *ValidationContext) {
	if data == nil {
		if len(mv.requiredKeys) > 0 {
			ctx.AddError(NewFieldError(mv.parentNamespace, "required").
				WithMessage("map field cannot be nil when required keys are specified"))
		}
		return
	}

	// 安全检查：防止DoS攻击
	if len(data) > maxMapSize {
		ctx.AddError(NewFieldError(mv.parentNamespace, "size").
			WithParam(strconv.Itoa(maxMapSize)).
			WithMessage(fmt.Sprintf("map size exceeds maximum limit %d", maxMapSize)))
		return
	}

	// 1. 验证必填键
	mv.validateRequiredKeys(data, ctx)

	// 2. 验证允许的键（白名单）
	mv.validateAllowedKeys(data, ctx)

	// 3. 执行自定义键验证
	mv.validateCustomKeys(data, ctx)
}

// validateRequiredKeys 验证必填键
func (mv *MapValidator) validateRequiredKeys(data map[string]any, ctx *ValidationContext) {
	for _, key := range mv.requiredKeys {
		if len(key) > maxMapKeyLength {
			ctx.AddError(NewFieldError(mv.parentNamespace, "key_length").
				WithParam(strconv.Itoa(maxMapKeyLength)).
				WithMessage(fmt.Sprintf("required key name exceeds maximum length %d", maxMapKeyLength)))
			continue
		}

		if err := validateKeyName(key); err != nil {
			ctx.AddError(NewFieldError(mv.parentNamespace, "invalid_key").
				WithMessage(fmt.Sprintf("invalid required key name '%s': %v", key, err)))
			continue
		}

		if _, exists := data[key]; !exists {
			namespace := mv.getNamespace(key)
			ctx.AddError(NewFieldError(namespace, "required").
				WithMessage(fmt.Sprintf("required key '%s' is missing", key)))
		}
	}
}

// validateAllowedKeys 验证允许的键（白名单）
func (mv *MapValidator) validateAllowedKeys(data map[string]any, ctx *ValidationContext) {
	if len(mv.allowedKeys) == 0 {
		return
	}

	// 懒加载缓存
	mv.initOnce.Do(func() {
		mv.allowedKeysMap = make(map[string]bool, len(mv.allowedKeys))
		for _, key := range mv.allowedKeys {
			mv.allowedKeysMap[key] = true
		}
	})

	for key := range data {
		if len(key) > maxMapKeyLength {
			ctx.AddError(NewFieldError(mv.parentNamespace, "key_length").
				WithParam(strconv.Itoa(maxMapKeyLength)).
				WithMessage(fmt.Sprintf("key name exceeds maximum length %d", maxMapKeyLength)))
			continue
		}

		if err := validateKeyName(key); err != nil {
			ctx.AddError(NewFieldError(mv.parentNamespace, "invalid_key").
				WithMessage(fmt.Sprintf("invalid key name '%s': %v", key, err)))
			continue
		}

		if !mv.allowedKeysMap[key] {
			namespace := mv.getNamespace(key)
			ctx.AddError(NewFieldError(namespace, "not_allowed").
				WithMessage(fmt.Sprintf("key '%s' is not in the allowed list", key)))
		}
	}
}

// validateCustomKeys 执行自定义键验证
func (mv *MapValidator) validateCustomKeys(data map[string]any, ctx *ValidationContext) {
	for key, validator := range mv.keyValidators {
		if validator == nil {
			continue
		}

		value, exists := data[key]
		if !exists {
			continue
		}

		// 错误恢复：防止验证函数panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					namespace := mv.getNamespace(key)
					ctx.AddError(NewFieldError(namespace, "validator_panic").
						WithMessage(fmt.Sprintf("validator function panicked: %v", r)))
				}
			}()

			if err := validator(value); err != nil {
				namespace := mv.getNamespace(key)
				ctx.AddError(NewFieldError(namespace, "custom").
					WithMessage(err.Error()))
			}
		}()
	}
}

// getNamespace 获取完整的命名空间
func (mv *MapValidator) getNamespace(key string) string {
	if len(mv.parentNamespace) == 0 {
		return key
	}
	return mv.parentNamespace + "." + key
}

// ============================================================================
// 便捷验证函数
// ============================================================================

const (
	maxMapSize      = 10000       // 最大map大小
	maxMapKeyLength = 256         // 最大键名长度
	maxMapValueSize = 1024 * 1024 // 最大值大小（1MB）
)

// validateKeyName 验证键名的有效性
func validateKeyName(key string) error {
	if len(key) == 0 {
		return fmt.Errorf("key name cannot be empty")
	}

	// 检查是否包含危险字符
	if strings.ContainsAny(key, "\x00\n\r\t") {
		return fmt.Errorf("key name contains invalid characters")
	}

	return nil
}

// ValidateMapKey 验证map中特定键的值（使用自定义验证函数）
func ValidateMapKey(data map[string]any, key string, validator func(value any) error) error {
	if data == nil {
		return fmt.Errorf("map is nil")
	}

	if err := validateKeyName(key); err != nil {
		return fmt.Errorf("invalid key name: %w", err)
	}

	value, exists := data[key]
	if !exists {
		return fmt.Errorf("key '%s' does not exist", key)
	}

	return validator(value)
}

// ValidateMapMustHaveKey 验证map必须包含指定的键
func ValidateMapMustHaveKey(data map[string]any, key string) error {
	if data == nil {
		return fmt.Errorf("map is nil")
	}

	if err := validateKeyName(key); err != nil {
		return fmt.Errorf("invalid key name: %w", err)
	}

	if _, exists := data[key]; !exists {
		return fmt.Errorf("required key '%s' is missing", key)
	}

	return nil
}

// ValidateMapMustHaveKeys 验证map必须包含指定的多个键
func ValidateMapMustHaveKeys(data map[string]any, keys ...string) error {
	if data == nil {
		return fmt.Errorf("map is nil")
	}

	var missingKeys []string
	for _, key := range keys {
		if err := validateKeyName(key); err != nil {
			return fmt.Errorf("invalid key name '%s': %w", key, err)
		}

		if _, exists := data[key]; !exists {
			missingKeys = append(missingKeys, key)
		}
	}

	if len(missingKeys) > 0 {
		return fmt.Errorf("missing required keys: %s", strings.Join(missingKeys, ", "))
	}

	return nil
}

// ValidateMapStringKey 验证map中字符串类型的键
func ValidateMapStringKey(data map[string]any, key string, minLen, maxLen int) error {
	return ValidateMapKey(data, key, func(value any) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("key '%s' must be a string, got %T", key, value)
		}

		strLen := len(str)
		if minLen > 0 && strLen < minLen {
			return fmt.Errorf("key '%s' length must be at least %d, got %d", key, minLen, strLen)
		}

		if maxLen > 0 && strLen > maxLen {
			return fmt.Errorf("key '%s' length must be at most %d, got %d", key, maxLen, strLen)
		}

		return nil
	})
}

// ValidateMapIntKey 验证map中整数类型的键
func ValidateMapIntKey(data map[string]any, key string, min, max int) error {
	return ValidateMapKey(data, key, func(value any) error {
		// 支持多种整数类型
		var intValue int
		switch v := value.(type) {
		case int:
			intValue = v
		case int8:
			intValue = int(v)
		case int16:
			intValue = int(v)
		case int32:
			intValue = int(v)
		case int64:
			intValue = int(v)
		case uint:
			intValue = int(v)
		case uint8:
			intValue = int(v)
		case uint16:
			intValue = int(v)
		case uint32:
			intValue = int(v)
		case uint64:
			intValue = int(v)
		case float32:
			intValue = int(v)
		case float64:
			intValue = int(v)
		default:
			return fmt.Errorf("key '%s' must be an integer, got %T", key, value)
		}

		if intValue < min {
			return fmt.Errorf("key '%s' must be at least %d, got %d", key, min, intValue)
		}

		if intValue > max {
			return fmt.Errorf("key '%s' must be at most %d, got %d", key, max, intValue)
		}

		return nil
	})
}

// ValidateMapFloatKey 验证map中浮点数类型的键
func ValidateMapFloatKey(data map[string]any, key string, min, max float64) error {
	return ValidateMapKey(data, key, func(value any) error {
		var floatValue float64
		switch v := value.(type) {
		case float32:
			floatValue = float64(v)
		case float64:
			floatValue = v
		case int:
			floatValue = float64(v)
		case int32:
			floatValue = float64(v)
		case int64:
			floatValue = float64(v)
		default:
			return fmt.Errorf("key '%s' must be a float, got %T", key, value)
		}

		if floatValue < min {
			return fmt.Errorf("key '%s' must be at least %f, got %f", key, min, floatValue)
		}

		if floatValue > max {
			return fmt.Errorf("key '%s' must be at most %f, got %f", key, max, floatValue)
		}

		return nil
	})
}

// ValidateMapBoolKey 验证map中布尔类型的键
func ValidateMapBoolKey(data map[string]any, key string) error {
	return ValidateMapKey(data, key, func(value any) error {
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("key '%s' must be a boolean, got %T", key, value)
		}
		return nil
	})
}

// ============================================================================
// MapStrategy - Map验证策略
// ============================================================================

// MapStrategy Map验证策略
// 职责：在验证流程中支持Map字段验证
// 设计原则：单一职责、策略模式
type MapStrategy struct {
	validators map[string]*MapValidator
}

// NewMapStrategy 创建Map验证策略
func NewMapStrategy() *MapStrategy {
	return &MapStrategy{
		validators: make(map[string]*MapValidator),
	}
}

// RegisterValidator 注册Map验证器
func (s *MapStrategy) RegisterValidator(typeName string, validator *MapValidator) {
	s.validators[typeName] = validator
}

// Type 策略类型
func (s *MapStrategy) Type() StrategyType {
	return 11
}

// Priority 优先级
func (s *MapStrategy) Priority() int8 {
	return 40
}

// Validate 执行验证
func (s *MapStrategy) Validate(target any, ctx *ValidationContext) error {
	// Map验证通常在BusinessValidation中手动触发
	// 这里提供一个框架支持，实际使用中可以在模型中调用MapValidator
	return nil
}
