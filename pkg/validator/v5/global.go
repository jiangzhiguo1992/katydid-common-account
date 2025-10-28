package v5

import (
	"katydid-common-account/pkg/validator/v5/core"
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
