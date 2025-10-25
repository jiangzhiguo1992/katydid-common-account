package v5

import "sync"

// ============================================================================
// 全局默认验证器 - 单例模式
// ============================================================================

var (
	defaultValidator *ValidatorEngine
	once             sync.Once
)

// Default 获取默认验证器实例（单例）
// 线程安全，延迟初始化
func Default() *ValidatorEngine {
	once.Do(func() {
		factory := NewValidatorFactory()
		defaultValidator = factory.CreateDefault()
	})
	return defaultValidator
}

// SetDefault 设置默认验证器
// 用于自定义全局验证器
func SetDefault(validator *ValidatorEngine) {
	defaultValidator = validator
}

// Validate 使用默认验证器验证对象
func Validate(target any, scene Scene) error {
	return Default().Validate(target, scene)
}

// ValidateFields 使用默认验证器验证指定字段
func ValidateFields(target any, scene Scene, fields ...string) error {
	return Default().ValidateFields(target, scene, fields...)
}

// ValidateExcept 使用默认验证器验证排除字段外的所有字段
func ValidateExcept(target any, scene Scene, fields ...string) error {
	return Default().ValidateExcept(target, scene, fields...)
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
	Default().typeRegistry.GetValidator().RegisterAlias(alias, tags)
}
