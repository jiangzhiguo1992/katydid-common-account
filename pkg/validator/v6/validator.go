package v6

import (
	"katydid-common-account/pkg/validator/v6/core"
	"katydid-common-account/pkg/validator/v6/facade"
)

var (
	// defaultValidator 全局默认验证器实例
	defaultValidator core.Validator
)

// init 初始化全局验证器
func init() {
	defaultValidator = facade.NewBuilder().BuildDefault()
}

// Validate 使用全局验证器验证
// 便捷方法，适合简单场景
func Validate(target any, scene core.Scene) error {
	return defaultValidator.Validate(target, scene)
}

// ValidateWithRequest 使用请求对象验证
func ValidateWithRequest(req *core.ValidationRequest) (*core.ValidationResult, error) {
	return defaultValidator.ValidateWithRequest(req)
}

// SetDefaultValidator 设置全局验证器
// 允许用户自定义全局验证器
func SetDefaultValidator(validator core.Validator) {
	if validator != nil {
		defaultValidator = validator
	}
}

// GetDefaultValidator 获取全局验证器
func GetDefaultValidator() core.Validator {
	return defaultValidator
}

// NewValidator 创建新的验证器（推荐用法）
// 使用建造者模式构建验证器
func NewValidator() *facade.Builder {
	return facade.NewBuilder()
}
