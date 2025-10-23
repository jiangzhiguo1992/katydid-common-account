package v2

import (
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// 核心验证器实现
// ============================================================================

// DefaultValidator 默认验证器实现
// 设计原则：
//   - 依赖倒置：依赖抽象接口而非具体实现
//   - 策略模式：通过可插拔的策略执行不同的验证逻辑
//   - 单一职责：只负责协调各个策略的执行
type DefaultValidator struct {
	strategies []ValidationStrategy
	typeCache  TypeCache
	registry   RegistryManager
	validate   *validator.Validate
}

// NewValidator 创建默认验证器 - 工厂方法
// 返回一个配置好的验证器实例
func NewValidator() *DefaultValidator {
	v := validator.New()

	// 配置 validator：使用 json tag 作为字段名
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})

	typeCache := NewTypeCache()
	registry := NewRegistryManager()

	valid := &DefaultValidator{
		strategies: make([]ValidationStrategy, 0, 3),
		typeCache:  typeCache,
		registry:   registry,
		validate:   v,
	}

	// 注册默认策略（执行顺序很重要）
	valid.addStrategy(NewRuleValidationStrategy(typeCache, v))
	valid.addStrategy(NewCustomValidationStrategy(typeCache))
	// 嵌套验证策略需要在实例化后添加，避免循环依赖

	return valid
}

// Validate 验证对象 - 实现 Validator 接口
func (v *DefaultValidator) Validate(obj any, scene Scene) Result {
	// 参数校验
	if obj == nil {
		return NewValidationResultWithErrors([]*FieldError{
			NewFieldError("", "", "required", "").
				WithMessage("validation target cannot be nil"),
		})
	}

	// 内存优化：从对象池获取错误收集器
	collector := AcquireErrorCollector()
	defer ReleaseErrorCollector(collector)

	// 按顺序执行所有策略
	for _, strategy := range v.strategies {
		if !strategy.Execute(obj, scene, collector) {
			// 策略返回 false 表示停止后续策略
			break
		}
	}

	// 构建并返回结果
	return NewValidationResultWithErrors(collector.GetErrors())
}

// ValidateFields 只验证结构体的指定字段 - 部分验证
// 用途：当只需要验证部分字段时使用，避免不必要的验证开销
//
// 示例：
//
//	// 只验证 Username 和 Email 字段
//	result := validator.ValidateFields(user, "create", "Username", "Email")
//
// 参数：
//   - obj: 待验证的对象
//   - scene: 验证场景
//   - fields: 需要验证的字段名列表（支持 JSON tag 名称）
//
// 返回：验证结果
func (v *DefaultValidator) ValidateFields(obj any, scene Scene, fields ...string) Result {
	if obj == nil {
		return NewValidationResultWithErrors([]*FieldError{
			NewFieldError("", "", "required", "").
				WithMessage("validation target cannot be nil"),
		})
	}

	if len(fields) == 0 {
		return NewValidationResult() // 没有指定字段，返回成功
	}

	// 创建字段集合，用于快速查找
	fieldSet := make(map[string]bool, len(fields))
	for _, f := range fields {
		if f != "" {
			fieldSet[f] = true
		}
	}

	// 内存优化：从对象池获取错误收集器
	collector := AcquireErrorCollector()
	defer ReleaseErrorCollector(collector)

	// 创建部分验证策略
	partialStrategy := NewPartialValidationStrategy(v.typeCache, v.validate, fieldSet, false)
	partialStrategy.Execute(obj, scene, collector)

	return NewValidationResultWithErrors(collector.GetErrors())
}

// ValidateExcept 验证结构体除了指定字段外的所有字段
// 用途：当需要跳过某些字段的验证时使用
//
// 示例：
//
//	// 验证除了 Password 外的所有字段
//	result := validator.ValidateExcept(user, "update", "Password")
//
// 参数：
//   - obj: 待验证的对象
//   - scene: 验证场景
//   - excludeFields: 需要排除的字段名列表
//
// 返回：验证结果
func (v *DefaultValidator) ValidateExcept(obj any, scene Scene, excludeFields ...string) Result {
	if obj == nil {
		return NewValidationResultWithErrors([]*FieldError{
			NewFieldError("", "", "required", "").
				WithMessage("validation target cannot be nil"),
		})
	}

	if len(excludeFields) == 0 {
		// 没有排除字段，执行完整验证
		return v.Validate(obj, scene)
	}

	// 创建排除字段集合
	excludeSet := make(map[string]bool, len(excludeFields))
	for _, f := range excludeFields {
		if f != "" {
			excludeSet[f] = true
		}
	}

	// 内存优化：从对象池获取错误收集器
	collector := AcquireErrorCollector()
	defer ReleaseErrorCollector(collector)

	// 创建排除验证策略
	excludeStrategy := NewPartialValidationStrategy(v.typeCache, v.validate, excludeSet, true)
	excludeStrategy.Execute(obj, scene, collector)

	// 仍然执行自定义验证和嵌套验证
	for _, strategy := range v.strategies {
		// 只执行自定义和嵌套验证策略
		switch strategy.(type) {
		case *CustomValidationStrategy, *NestedValidationStrategy:
			if !strategy.Execute(obj, scene, collector) {
				break
			}
		}
	}

	return NewValidationResultWithErrors(collector.GetErrors())
}

// RegisterAlias 注册验证标签别名
// 用途：创建自定义标签别名，简化常用的复杂验证规则
//
// 示例：
//
//	validator.RegisterAlias("password", "required,min=8,max=50,containsany=!@#$%^&*()")
//
// 参数：
//   - alias: 别名标签名
//   - tags: 实际的验证规则字符串
func (v *DefaultValidator) RegisterAlias(alias, tags string) {
	if alias == "" || tags == "" {
		return
	}
	v.validate.RegisterAlias(alias, tags)
}

// addStrategy 添加验证策略 - 私有方法
func (v *DefaultValidator) addStrategy(strategy ValidationStrategy) {
	if strategy != nil {
		v.strategies = append(v.strategies, strategy)
	}
}

// GetUnderlyingValidator 获取底层的 go-playground/validator 实例
// 用于高级场景下直接访问底层验证器
func (v *DefaultValidator) GetUnderlyingValidator() *validator.Validate {
	return v.validate
}

// ClearCache 清除所有缓存
// 用于测试或需要重新加载类型信息的场景
func (v *DefaultValidator) ClearCache() {
	v.typeCache.Clear()
	v.registry.Clear()
}

// ============================================================================
// 全局默认验证器 - 单例模式
// ============================================================================

var (
	defaultValidator *DefaultValidator
	defaultOnce      sync.Once
)

// Default 获取全局默认验证器实例 - 单例模式
// 线程安全，可在多个 goroutine 中并发调用
func Default() Validator {
	defaultOnce.Do(func() {
		defaultValidator = NewValidator()
	})
	return defaultValidator
}

// Validate 使用默认验证器验证对象 - 便捷函数
// 简化常见的验证调用场景
func Validate(obj any, scene Scene) Result {
	return Default().Validate(obj, scene)
}

// ValidateFields 使用默认验证器验证指定字段 - 便捷函数
func ValidateFields(obj any, scene Scene, fields ...string) Result {
	return Default().ValidateFields(obj, scene, fields...)
}

// ValidateExcept 使用默认验证器验证排除字段外的所有字段 - 便捷函数
func ValidateExcept(obj any, scene Scene, excludeFields ...string) Result {
	return Default().ValidateExcept(obj, scene, excludeFields...)
}

// RegisterAlias 在默认验证器上注册别名 - 便捷函数
func RegisterAlias(alias, tags string) {
	if dv, ok := Default().(*DefaultValidator); ok {
		dv.RegisterAlias(alias, tags)
	}
}

// ClearCache 清除默认验证器的缓存 - 便捷函数
// 用于测试场景
func ClearCache() {
	if defaultValidator != nil {
		defaultValidator.ClearCache()
	}
}
