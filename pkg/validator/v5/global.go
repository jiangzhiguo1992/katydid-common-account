package v5

import (
	"fmt"
	"katydid-common-account/pkg/validator/v5/core"
	"katydid-common-account/pkg/validator/v5/engine"
	"strings"
	"sync"
)

// 全局默认验证器实例（单例）
var (
	defaultValidator core.IValidator
	once             sync.Once
)

// Default 获取默认验证器实例（单例）
// 线程安全，延迟初始化
func Default() core.IValidator {
	once.Do(func() {
		if defaultValidator == nil {
			factory := NewValidatorFactory()
			defaultValidator = factory.CreateDefault()
		}
	})
	return defaultValidator
}

// RegisterAlias 注册别名（alias:tags）
func RegisterAlias(alias, tags string) {
	Default().RegisterAlias(alias, tags)
}

// RegisterValidation 注册自定义验证函数（tag:func）
func RegisterValidation(tag string, fn func()) error {
	return Default().RegisterValidation(tag, fn)
}

// Validate 使用默认验证器验证对象
func Validate(target any, scene core.Scene) core.IValidationError {
	return Default().Validate(target, scene)
}

// ValidateFields 使用默认验证器验证指定字段
func ValidateFields(target any, scene core.Scene, fields ...string) core.IValidationError {
	return Default().ValidateFields(target, scene, fields...)
}

// ValidateFieldsExcept 使用默认验证器验证排除字段外的所有字段
func ValidateFieldsExcept(target any, scene core.Scene, fields ...string) core.IValidationError {
	return Default().ValidateFieldsExcept(target, scene, fields...)
}

// ============================================================================
// Map验证
// ============================================================================

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

	if err := engine.ValidateKeyName(key); err != nil {
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

	if err := engine.ValidateKeyName(key); err != nil {
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

		if err := engine.ValidateKeyName(key); err != nil {
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
		errMsg := core.AcquireStringBuilder()
		defer core.ReleaseStringBuilder(errMsg)

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
