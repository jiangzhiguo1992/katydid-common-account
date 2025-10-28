package v5

import (
	"katydid-common-account/pkg/validator/v5/core"
	"katydid-common-account/pkg/validator/v5/engine"
	"katydid-common-account/pkg/validator/v5/formatter"
	"katydid-common-account/pkg/validator/v5/registry"
	"katydid-common-account/pkg/validator/v5/strategy"

	"github.com/go-playground/validator/v10"
)

// ValidatorFactory 验证器工厂
// 职责：创建预配置的验证器实例
type ValidatorFactory struct{}

// NewValidatorFactory 创建验证器工厂
func NewValidatorFactory() *ValidatorFactory {
	return &ValidatorFactory{}
}

// Create 创建自定义验证器
func (f *ValidatorFactory) Create(opts ...engine.ValidatorEngineOption) core.IValidator {
	v := validator.New()
	typeRegistry := registry.NewTypeRegistry(v)
	return engine.NewValidatorEngine(typeRegistry, opts...)
}

// CreateDefault 创建默认验证器
func (f *ValidatorFactory) CreateDefault() core.IValidator {
	v := validator.New()
	typeRegistry := registry.NewTypeRegistry(v)
	sceneMatcher := core.NewSceneBitMatcher()
	maxDepth := int8(100)

	ve := engine.NewValidatorEngine(typeRegistry,
		engine.WithStrategies(strategy.NewRuleStrategy(v, typeRegistry, sceneMatcher)),
		engine.WithStrategies(strategy.NewBusinessStrategy()),
		engine.WithErrorFormatter(formatter.NewLocalizesErrorFormatter()),
		engine.WithMaxDepth(maxDepth))

	// 添加标准策略
	ve.AddStrategy(strategy.NewNestedStrategy(ve, maxDepth))

	// 没有listener
	return ve
}
