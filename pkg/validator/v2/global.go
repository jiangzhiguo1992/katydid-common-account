package v2

// ============================================================================
// 全局单例验证器 - 便捷使用
// ============================================================================

var (
	// defaultGlobalValidator 全局默认验证器
	defaultGlobalValidator Validator
)

// init 初始化全局验证器
func init() {
	var err error
	defaultGlobalValidator, err = NewDefaultValidator()
	if err != nil {
		panic(err)
	}
}

// Validate 使用全局验证器进行验证
func Validate(data interface{}, scene Scene) error {
	return defaultGlobalValidator.Validate(data, scene)
}

// ValidatePartial 使用全局验证器进行部分字段验证
func ValidatePartial(data interface{}, fields ...string) error {
	return defaultGlobalValidator.ValidatePartial(data, fields...)
}

// ValidateExcept 使用全局验证器进行排除字段验证
func ValidateExcept(data interface{}, scene Scene, excludeFields ...string) error {
	return defaultGlobalValidator.ValidateExcept(data, scene, excludeFields...)
}

// ValidateFields 使用全局验证器进行场景化的部分字段验证
func ValidateFields(data interface{}, scene Scene, fields ...string) error {
	return defaultGlobalValidator.ValidateFields(data, scene, fields...)
}

// SetGlobalValidator 设置全局验证器
func SetGlobalValidator(validator Validator) {
	defaultGlobalValidator = validator
}

// GetGlobalValidator 获取全局验证器
func GetGlobalValidator() Validator {
	return defaultGlobalValidator
}

// ============================================================================
// 便捷函数 - 快速创建和使用验证器
// ============================================================================

// Quick 快速验证（创建一次性验证器）
func Quick(data interface{}, scene Scene) error {
	v, err := NewSimpleValidator()
	if err != nil {
		return err
	}
	return v.Validate(data, scene)
}

// QuickPartial 快速部分验证
func QuickPartial(data interface{}, fields ...string) error {
	v, err := NewSimpleValidator()
	if err != nil {
		return err
	}
	return v.ValidatePartial(data, fields...)
}

// Must 必须验证（panic on error）
func Must(data interface{}, scene Scene) {
	if err := Validate(data, scene); err != nil {
		panic(err)
	}
}

// MustPartial 必须部分验证（panic on error）
func MustPartial(data interface{}, fields ...string) {
	if err := ValidatePartial(data, fields...); err != nil {
		panic(err)
	}
}

// MustExcept 必须排除字段验证（panic on error）
func MustExcept(data interface{}, scene Scene, excludeFields ...string) {
	if err := ValidateExcept(data, scene, excludeFields...); err != nil {
		panic(err)
	}
}

// ============================================================================
// 全局配置函数
// ============================================================================

// RegisterAlias 在全局验证器上注册别名（需要重新构建验证器）
func RegisterAlias(alias, tags string) error {
	v, err := NewValidatorBuilder().
		WithCache(NewCacheManager()).
		WithPool(NewValidatorPool()).
		WithStrategy(NewDefaultStrategy()).
		RegisterAlias(alias, tags).
		Build()
	if err != nil {
		return err
	}
	SetGlobalValidator(v)
	return nil
}

// ClearCache 清除全局验证器的缓存
func ClearCache() {
	// 尝试从全局验证器中获取缓存并清理
	if v, ok := defaultGlobalValidator.(*defaultValidator); ok {
		if v.cache != nil {
			v.cache.Clear()
		}
	}
}
