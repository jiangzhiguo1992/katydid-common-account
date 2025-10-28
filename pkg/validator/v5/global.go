package v5

import (
	"katydid-common-account/pkg/validator/v5/formatter"
	"sync"
)

// 全局默认验证器实例（单例）
var (
	defaultValidator *ValidatorEngine
	once             sync.Once
)

// Default 获取默认验证器实例（单例）
// 线程安全，延迟初始化
func Default() *ValidatorEngine {
	once.Do(func() {
		if defaultValidator == nil {
			factory := NewValidatorFactory()
			defaultValidator = factory.CreateDefault()
		}
	})
	return defaultValidator
}

// SetDefault 设置默认验证器
// 用于自定义全局验证器
func SetDefault(validator *ValidatorEngine) {
	defaultValidator = validator
}

// SetDefaultFormater 设置默认错误格式化器
func SetDefaultFormater(errorFormatter formatter.ErrorFormatter) {
	Default().errorFormatter = errorFormatter
}

// Validate 使用默认验证器验证对象
func Validate(target any, scene Scene) *ValidationError {
	return Default().Validate(target, scene)
}

// ValidateFields 使用默认验证器验证指定字段
func ValidateFields(target any, scene Scene, fields ...string) *ValidationError {
	return Default().ValidateFields(target, scene, fields...)
}

// ValidateFieldsExcept 使用默认验证器验证排除字段外的所有字段
func ValidateFieldsExcept(target any, scene Scene, fields ...string) *ValidationError {
	return Default().ValidateFieldsExcept(target, scene, fields...)
}

// ClearCache 清除默认验证器的缓存
func ClearCache() {
	Default().ClearCache()
}

// Stats 获取默认验证器的统计信息
func Stats() map[string]any {
	return Default().Stats()
}

// RegisterAlias 注册验证标签别名
// 用途：创建自定义标签别名，简化常用的复杂验证规则
//
// 示例：
//
//	validator.Default().RegisterAlias("password", "required,min=8,max=50,containsany=!@#$%^&*()")
//
//	// 在 RuleValidator 中使用别名
//	func (u *User) RuleValidation() map[ValidateScene]map[string]string {
//	    return map[ValidateScene]map[string]string{
//	        SceneCreate: {"Password": "password"},  // 使用别名
//	    }
//	}
//
// 参数：
//   - alias: 别名标签名
//   - tags: 实际的验证规则字符串
func RegisterAlias(alias, tags string) {
	if len(alias) == 0 || len(tags) == 0 {
		return
	}
	Default().GetValidator().RegisterAlias(alias, tags)
}
