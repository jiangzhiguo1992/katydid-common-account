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
