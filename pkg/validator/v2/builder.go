package v2

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// 验证器建造者 - 建造者模式
// ============================================================================

// DefaultValidatorBuilder 默认验证器建造者实现
// 设计原则：
//   - 建造者模式：提供灵活的构建方式
//   - 流式接口：支持链式调用
//   - 开放封闭：通过建造者扩展配置，而不修改验证器
type DefaultValidatorBuilder struct {
	strategies []ValidationStrategy
	typeCache  TypeCache
	registry   RegistryManager
	validate   *validator.Validate
	maxDepth   int
}

// NewValidatorBuilder 创建验证器建造者 - 工厂方法
func NewValidatorBuilder() *DefaultValidatorBuilder {
	v := validator.New()

	// 配置 validator：使用 json tag 作为字段名
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})

	return &DefaultValidatorBuilder{
		strategies: make([]ValidationStrategy, 0),
		typeCache:  NewTypeCache(),
		registry:   NewRegistryManager(),
		validate:   v,
		maxDepth:   100, // 默认最大深度
	}
}

// WithStrategy 添加验证策略 - 实现 ValidatorBuilder 接口
func (b *DefaultValidatorBuilder) WithStrategy(strategy ValidationStrategy) ValidatorBuilder {
	if strategy != nil {
		b.strategies = append(b.strategies, strategy)
	}
	return b
}

// WithTypeCache 设置类型缓存 - 实现 ValidatorBuilder 接口
func (b *DefaultValidatorBuilder) WithTypeCache(cache TypeCache) ValidatorBuilder {
	if cache != nil {
		b.typeCache = cache
	}
	return b
}

// WithRegistry 设置注册管理器 - 实现 ValidatorBuilder 接口
func (b *DefaultValidatorBuilder) WithRegistry(registry RegistryManager) ValidatorBuilder {
	if registry != nil {
		b.registry = registry
	}
	return b
}

// WithMaxDepth 设置最大嵌套深度
func (b *DefaultValidatorBuilder) WithMaxDepth(depth int) *DefaultValidatorBuilder {
	if depth > 0 {
		b.maxDepth = depth
	}
	return b
}

// WithDefaultStrategies 使用默认策略集
func (b *DefaultValidatorBuilder) WithDefaultStrategies() *DefaultValidatorBuilder {
	// 清空现有策略
	b.strategies = make([]ValidationStrategy, 0, 3)

	// 添加默认策略（顺序很重要）
	b.strategies = append(b.strategies,
		NewRuleValidationStrategy(b.typeCache, b.validate),
		NewCustomValidationStrategy(b.typeCache),
	)

	return b
}

// Build 构建验证器 - 实现 ValidatorBuilder 接口
func (b *DefaultValidatorBuilder) Build() Validator {
	v := &DefaultValidator{
		strategies: b.strategies,
		typeCache:  b.typeCache,
		registry:   b.registry,
		validate:   b.validate,
	}

	// 如果没有添加任何策略，使用默认策略
	if len(v.strategies) == 0 {
		v.strategies = []ValidationStrategy{
			NewRuleValidationStrategy(b.typeCache, b.validate),
			NewCustomValidationStrategy(b.typeCache),
		}
	}

	// 添加嵌套验证策略（必须在最后，因为需要引用构建好的 v）
	v.strategies = append(v.strategies,
		NewNestedValidationStrategy(v, b.maxDepth))

	return v
}
