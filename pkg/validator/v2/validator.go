package v2

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// 验证器核心实现 - 协调者模式
// ============================================================================

// Validator 验证器
// 设计原则：
//   - 依赖倒置：依赖抽象接口而非具体实现
//   - 单一职责：只负责协调各个组件
type Validator struct {
	validate  *validator.Validate
	typeCache TypeInfoCache
	strategy  ValidationStrategy
}

// Config 验证器配置
type Config struct {
	// TypeCache 类型缓存（可选，默认使用内置实现）
	TypeCache TypeInfoCache

	// Strategy 验证策略（可选，默认使用组合策略）
	Strategy ValidationStrategy
}

// NewValidator 创建验证器（工厂方法）
func NewValidator(configs ...Config) *Validator {
	v := validator.New()

	// 配置 JSON tag 作为字段名
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})

	// 默认配置
	var config Config
	if len(configs) > 0 {
		config = configs[0]
	}

	// 创建类型缓存
	typeCache := config.TypeCache
	if typeCache == nil {
		typeCache = NewTypeCache()
	}

	// 创建验证策略
	strategy := config.Strategy
	if strategy == nil {
		// 默认使用组合策略
		strategy = NewCompositeStrategy(
			NewRuleStrategy(v),
			NewBusinessStrategy(),
		)
	}

	return &Validator{
		validate:  v,
		typeCache: typeCache,
		strategy:  strategy,
	}
}

// Validate 验证对象
// 参数：
//   - obj: 待验证对象
//   - scene: 验证场景
//
// 返回：
//   - 验证错误列表，nil 表示验证通过
func (v *Validator) Validate(obj any, scene ValidateScene) []ValidationError {
	// 参数校验
	if obj == nil {
		return []ValidationError{
			NewFieldError("object", "required", "validation target cannot be nil"),
		}
	}

	// 创建错误收集器
	collector := NewErrorCollector()

	// 执行验证策略
	v.strategy.Execute(obj, scene, collector)

	// 返回错误
	return collector.GetAll()
}

// ClearCache 清除类型缓存
func (v *Validator) ClearCache() {
	v.typeCache.Clear()
}

// GetUnderlyingValidator 获取底层验证器
// 用于高级场景，直接访问 go-playground/validator
func (v *Validator) GetUnderlyingValidator() *validator.Validate {
	return v.validate
}

// RegisterAlias 注册验证规则别名
func (v *Validator) RegisterAlias(alias, tags string) {
	if alias != "" && tags != "" {
		v.validate.RegisterAlias(alias, tags)
	}
}

// Package v2 提供了重构后的验证器实现
//
// 设计原则：
//   - 单一职责原则（SRP）：每个组件只负责一个功能
//   - 开放封闭原则（OCP）：对扩展开放，对修改封闭
//   - 里氏替换原则（LSP）：所有实现可以互相替换
//   - 接口隔离原则（ISP）：细化的专用接口
//   - 依赖倒置原则（DIP）：依赖抽象而非具体实现
//
// 架构特点：
//   - 高内聚低耦合
//   - 策略模式实现可扩展验证
//   - 工厂模式创建对象
//   - 依赖注入支持测试
//
// 使用示例：
//
//	// 创建验证器
//	validator := v2.NewValidator()
//
//	// 验证对象
//	errors := validator.Validate(user, v2.SceneCreate)
//
//	// 处理错误
//	if len(errors) > 0 {
//	    for _, err := range errors {
//	        fmt.Println(err.Message)
//	    }
//	}
