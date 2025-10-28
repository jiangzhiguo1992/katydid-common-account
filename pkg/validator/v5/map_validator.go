package v5

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// 常量定义：用于错误消息和性能优化
const (
	maxMapSize      = 1000 // 最大键名长度，防止恶意超长键名
	maxMapKeyLength = 256  // 最大 map 大小，防止 DoS 攻击
)

// MapValidator Map字段验证器
// 职责：专门验证 map[string]any 类型的动态字段
type MapValidator struct {
	// parentNamespace 父级命名空间，用于生成准确的错误路径
	// 例如：User.Extras, Product.Metadata
	parentNamespace string

	// requiredKeys 必填键列表
	requiredKeys []string

	// allowedKeys 允许的键白名单（空则不限制）
	// 用于防止非法字段注入，提升数据安全性
	allowedKeys []string

	// keyValidators 自定义键验证器 map[tag][func]
	// key: Tag，value: 验证函数（返回 error 表示Param/Message）
	keyValidators map[string]func(value any) error

	// allowedKeysMap 缓存的允许键映射（性能优化）
	//	// 使用 map 查找的时间复杂度为 O(1)，优于切片遍历的 O(n)
	allowedKeysMap map[string]bool

	// initOnce 确保缓存只初始化一次（线程安全）
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
	if ctx == nil {
		return
	}

	if data == nil || len(data) == 0 {
		if len(mv.requiredKeys) > 0 {
			ctx.AddError(NewFieldError("Map", "required"))
		}
		return
	}

	// 安全检查：防止DoS攻击
	if len(data) > maxMapSize {
		ctx.AddError(NewFieldError("Map", "size").
			WithParam(strconv.Itoa(maxMapSize)).
			WithValue(len(data)))
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
	if mv.requiredKeys == nil || len(mv.requiredKeys) == 0 {
		return
	}

	for _, key := range mv.requiredKeys {
		if len(key) > maxMapKeyLength {
			if !ctx.AddError(NewFieldError("Map", "key_len").
				WithParam(strconv.Itoa(maxMapKeyLength)).
				WithValue(key)) {
				break
			}
			continue
		}

		if err := validateKeyName(key); err != nil {
			if !ctx.AddError(NewFieldError("Map", "invalid_key").
				WithValue(key).
				WithMessage(fmt.Sprintf("invalid required key name '%s': %v", key, err))) {
				break
			}
			continue
		}

		if _, exists := data[key]; !exists {
			namespace := mv.getNamespace(key)
			if !ctx.AddError(NewFieldError(namespace, "required")) {
				break
			}
		}
	}
}

// validateAllowedKeys 验证允许的键（白名单）
func (mv *MapValidator) validateAllowedKeys(data map[string]any, ctx *ValidationContext) {
	if mv.allowedKeys == nil || len(mv.allowedKeys) == 0 {
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
			if !ctx.AddError(NewFieldError("Map", "key_len").
				WithParam(strconv.Itoa(maxMapKeyLength)).
				WithValue(key)) {
				break
			}
			continue
		}

		if err := validateKeyName(key); err != nil {
			if !ctx.AddError(NewFieldError("Map", "invalid_key").
				WithValue(key).
				WithMessage(fmt.Sprintf("invalid key name '%s': %v", key, err))) {
				break
			}
			continue
		}

		if !mv.allowedKeysMap[key] {
			namespace := mv.getNamespace(key)
			if !ctx.AddError(NewFieldError(namespace, "not_allowed")) {
				break
			}
		}
	}
}

// validateCustomKeys 执行自定义键验证
func (mv *MapValidator) validateCustomKeys(data map[string]any, ctx *ValidationContext) {
	if mv.keyValidators == nil || len(mv.keyValidators) == 0 {
		return
	}

	for key, validator := range mv.keyValidators {
		if validator == nil {
			continue
		}

		value, exists := data[key]
		if !exists {
			continue // 键不存在时不执行验证
		}

		// 错误恢复：防止验证函数panic
		canNext := func() bool {
			defer func() {
				if r := recover(); r != nil {
					ctx.AddError(NewFieldError("Map", "validate_panic").
						WithValue(key).
						WithMessage(fmt.Sprintf("validator function panicked: %v", r)))
				}
			}()

			if err := validator(value); err != nil {
				if !ctx.AddError(NewFieldError(mv.parentNamespace, key).
					WithParam(err.Error()).
					WithMessage(err.Error())) {
					return false
				}
			}
			return true
		}()
		if !canNext {
			break
		}
	}
}

// getNamespace 获取完整的命名空间
func (mv *MapValidator) getNamespace(key string) string {
	if len(mv.parentNamespace) == 0 {
		return key
	}

	// 内存优化：从对象池获取 strings.Builder
	builder := acquireStringBuilder()
	defer releaseStringBuilder(builder)

	builder.Grow(len(mv.parentNamespace) + len(key) + 1)
	builder.WriteString(mv.parentNamespace)
	builder.WriteString(".")
	builder.WriteString(key)
	return builder.String()
}

// ValidateMapKey 验证map中特定键的值（使用自定义验证函数）
func ValidateMapKey(data map[string]any, key string, validatorFunc func(value any) error) error {
	if data == nil || len(data) == 0 {
		return fmt.Errorf("map is nil")
	}

	if len(key) == 0 {
		return fmt.Errorf("map key validation failed: key name cannot be empty")
	}

	if validatorFunc == nil {
		return fmt.Errorf("map key validation failed: validator function cannot be nil")
	}

	if err := validateKeyName(key); err != nil {
		return fmt.Errorf("invalid key name: %w", err)
	}

	value, exists := data[key]
	if !exists {
		return fmt.Errorf("key '%s' does not exist", key)
	}

	var validationErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				validationErr = fmt.Errorf("map key '%s' validation failed: validator panicked: %v", key, r)
			}
		}()
		validationErr = validatorFunc(value)
	}()

	return validationErr
}

// ValidateMapMustHaveKey 验证map必须包含指定的键
func ValidateMapMustHaveKey(data map[string]any, key string) error {
	if data == nil || len(data) == 0 {
		return fmt.Errorf("map is nil")
	}

	if len(key) == 0 {
		return fmt.Errorf("map key validation failed: key name cannot be empty")
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
	if data == nil || len(data) == 0 {
		return fmt.Errorf("map is nil")
	}

	if len(keys) == 0 {
		return nil
	}

	var missingKeys []string
	var invalidKeys []string

	for _, key := range keys {
		if len(key) == 0 {
			continue // 忽略空键名
		}

		if err := validateKeyName(key); err != nil {
			invalidKeys = append(invalidKeys, key)
			continue
		}

		if _, exists := data[key]; !exists {
			missingKeys = append(missingKeys, key)
		}
	}

	// 构建错误消息
	if len(invalidKeys) > 0 || len(missingKeys) > 0 {
		// 内存优化：从对象池获取 strings.Builder
		errMsg := acquireStringBuilder()
		defer releaseStringBuilder(errMsg)

		errMsg.WriteString("map validation failed: ")

		if len(invalidKeys) > 0 {
			errMsg.WriteString(fmt.Sprintf("invalid key names: %s", strings.Join(invalidKeys, ", ")))
		}

		if len(missingKeys) > 0 {
			if len(invalidKeys) > 0 {
				errMsg.WriteString("; ")
			}
			errMsg.WriteString(fmt.Sprintf("missing required keys: %s", strings.Join(missingKeys, ", ")))
		}

		return fmt.Errorf("%s", errMsg.String())
	}

	return nil
}

// validateKeyName 验证键名的有效性
func validateKeyName(key string) error {
	if len(key) == 0 {
		return fmt.Errorf("key name cannot be empty")
	}

	// 检查是否包含危险字符
	if strings.ContainsAny(key, "\x00\n\r\t") {
		return fmt.Errorf("key name contains invalid characters")
	}

	// 检查是否包含控制字符（ASCII 0-31）
	for _, r := range key {
		if r < 32 {
			return fmt.Errorf("key name contains control character (code %d)", r)
		}
		// 检查是否包含危险字符（防止注入攻击）
		if r == 0x7F { // DEL 字符
			return fmt.Errorf("key name contains delete character")
		}
	}

	return nil
}

// ============================================================================
// MapStrategy - Map验证策略 TODO:GG 有必要吗
// ============================================================================

// MapStrategy Map验证策略
// 职责：在验证流程中支持Map字段验证
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
