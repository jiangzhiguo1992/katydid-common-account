package engine

import (
	"fmt"
	"katydid-common-account/pkg/validator/v5/core"
	"katydid-common-account/pkg/validator/v5/err"
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
	// 例如：User.Extras, Product.metadata
	parentNamespace string

	// requiredKeys 必填键列表
	requiredKeys []string

	// allowedKeys 允许的键白名单（空则不限制）
	// 用于防止非法字段注入，提升数据安全性
	allowedKeys []string

	// keyValidators 自定义键验证器 map[tag][func]
	// key: tag，value: 验证函数（返回 error 表示Param/message）
	keyValidators map[string]func(value any) error

	// allowedKeysMap 缓存的允许键映射（性能优化）
	//	// 使用 map 查找的时间复杂度为 O(1)，优于切片遍历的 O(n)
	allowedKeysMap map[string]bool

	// initOnce 确保缓存只初始化一次（线程安全）
	initOnce sync.Once
}

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

// MapValidatorOption Map验证器选项
type MapValidatorOption func(*MapValidator)

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
func (mv *MapValidator) Validate(data map[string]any, ctx core.IValidationContext) {
	if ctx == nil {
		return
	}

	if data == nil || len(data) == 0 {
		if len(mv.requiredKeys) > 0 {
			ctx.AddError(err.NewFieldError("Map", "required"))
		}
		return
	}

	// 安全检查：防止DoS攻击
	if len(data) > maxMapSize {
		ctx.AddError(err.NewFieldError("Map", "size",
			err.WithParam(strconv.Itoa(maxMapSize)),
			err.WithValue(len(data))))
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
func (mv *MapValidator) validateRequiredKeys(data map[string]any, ctx core.IValidationContext) {
	if mv.requiredKeys == nil || len(mv.requiredKeys) == 0 {
		return
	}

	for _, key := range mv.requiredKeys {
		if len(key) > maxMapKeyLength {
			if !ctx.AddError(err.NewFieldError("Map", "key_len",
				err.WithParam(strconv.Itoa(maxMapKeyLength)),
				err.WithValue(key))) {
				break
			}
			continue
		}

		if e := ValidateKeyName(key); e != nil {
			if !ctx.AddError(err.NewFieldError("Map", "invalid_key",
				err.WithValue(key),
				err.WithMessage(fmt.Sprintf("invalid required key name '%s': %v", key, e)))) {
				break
			}
			continue
		}

		if _, exists := data[key]; !exists {
			namespace := mv.getNamespace(key)
			if !ctx.AddError(err.NewFieldError(namespace, "required")) {
				break
			}
		}
	}
}

// validateAllowedKeys 验证允许的键（白名单）
func (mv *MapValidator) validateAllowedKeys(data map[string]any, ctx core.IValidationContext) {
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
			if !ctx.AddError(err.NewFieldError("Map", "key_len",
				err.WithParam(strconv.Itoa(maxMapKeyLength)),
				err.WithValue(key))) {
				break
			}
			continue
		}

		if e := ValidateKeyName(key); e != nil {
			if !ctx.AddError(err.NewFieldError("Map", "invalid_key",
				err.WithValue(key),
				err.WithMessage(fmt.Sprintf("invalid key name '%s': %v", key, e)))) {
				break
			}
			continue
		}

		if !mv.allowedKeysMap[key] {
			namespace := mv.getNamespace(key)
			if !ctx.AddError(err.NewFieldError(namespace, "not_allowed")) {
				break
			}
		}
	}
}

// validateCustomKeys 执行自定义键验证
func (mv *MapValidator) validateCustomKeys(data map[string]any, ctx core.IValidationContext) {
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
					ctx.AddError(err.NewFieldError("Map", "validate_panic",
						err.WithValue(key),
						err.WithMessage(fmt.Sprintf("validator function panicked: %v", r))))
				}
			}()

			if e := validator(value); e != nil {
				if !ctx.AddError(err.NewFieldError(mv.parentNamespace, key,
					err.WithParam(e.Error()),
					err.WithMessage(e.Error()))) {
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
	builder := core.AcquireStringBuilder()
	defer core.ReleaseStringBuilder(builder)

	builder.Grow(len(mv.parentNamespace) + len(key) + 1)
	builder.WriteString(mv.parentNamespace)
	builder.WriteString(".")
	builder.WriteString(key)
	return builder.String()
}

// ValidateKeyName 验证键名的有效性
func ValidateKeyName(key string) error {
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
