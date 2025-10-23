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

	validator := &DefaultValidator{
		strategies: make([]ValidationStrategy, 0, 3),
		typeCache:  typeCache,
		registry:   registry,
		validate:   v,
	}

	// 注册默认策略（执行顺序很重要）
	validator.addStrategy(NewRuleValidationStrategy(typeCache, v))
	validator.addStrategy(NewCustomValidationStrategy(typeCache))
	// 嵌套验证策略需要在实例化后添加，避免循环依赖

	return validator
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

	// 创建错误收集器
	collector := NewErrorCollector()

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

// ClearCache 清除默认验证器的缓存 - 便捷函数
// 用于测试场景
func ClearCache() {
	if defaultValidator != nil {
		defaultValidator.ClearCache()
	}
}
