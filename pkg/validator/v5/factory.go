package v5

// ============================================================================
// ValidatorFactory - 验证器工厂
// ============================================================================

// ValidatorFactory 验证器工厂
// 职责：创建预配置的验证器实例
// 设计模式：工厂模式
type ValidatorFactory struct{}

// NewValidatorFactory 创建验证器工厂
func NewValidatorFactory() *ValidatorFactory {
	return &ValidatorFactory{}
}

// Create 创建自定义验证器
func (f *ValidatorFactory) Create(opts ...EngineOption) *ValidatorEngine {
	return NewValidatorEngine(opts...)
}

// CreateDefault 创建默认验证器
// 包含所有标准验证策略
func (f *ValidatorFactory) CreateDefault() *ValidatorEngine {

	engine := NewValidatorEngine(
		WithMaxDepth(100),
		WithMaxErrors(100),
	)

	// 添加标准策略
	engine.AddStrategy(NewRuleStrategy(engine.sceneMatcher, engine.typeRegistry))
	engine.AddStrategy(NewBusinessStrategy())
	engine.AddStrategy(NewNestedStrategy(engine, engine.maxDepth))

	return engine
}

// CreateMinimal 创建最小验证器
// 只包含规则验证策略
func (f *ValidatorFactory) CreateMinimal() *ValidatorEngine {
	engine := NewValidatorEngine()
	engine.AddStrategy(NewRuleStrategy(engine.sceneMatcher, engine.typeRegistry))
	return engine
}

//
//// ============================================================================
//// ValidatorBuilder - 验证器构建器
//// ============================================================================
//
//// ValidatorBuilder 验证器构建器
//// 职责：提供流畅的 API 构建复杂配置的验证器
//// 设计模式：建造者模式
//type ValidatorBuilder struct {
//	strategies     []ValidationStrategy
//	typeRegistry   TypeRegistry
//	sceneMatcher   SceneMatcher
//	listeners      []ValidationListener
//	errorFormatter ErrorFormatter
//	maxDepth       int
//	maxErrors      int
//}
//
//// NewValidatorBuilder 创建验证器构建器
//func NewValidatorBuilder() *ValidatorBuilder {
//	return &ValidatorBuilder{
//		strategies:     make([]ValidationStrategy, 0),
//		typeRegistry:   NewTypeCacheRegistry(),
//		sceneMatcher:   NewSceneBitMatcher(),
//		listeners:      make([]ValidationListener, 0),
//		errorFormatter: NewDefaultErrorFormatter(),
//		maxDepth:       100,
//		maxErrors:      1000,
//	}
//}
//
//// WithDefault 默认建造器
//func (b *ValidatorBuilder) WithDefault() *ValidatorBuilder {
//	b.WithSceneMatcher(NewSceneBitMatcher()).
//		WithTypeRegistry(NewTypeCacheRegistry()).
//		WithRuleStrategy().
//		WithBusinessStrategy().
//		WithMaxDepth(100).
//		WithMaxErrors(1000)
//	return b
//}
//
//// WithSceneMatcher 设置场景匹配器
//func (b *ValidatorBuilder) WithSceneMatcher(matcher SceneMatcher) *ValidatorBuilder {
//	b.sceneMatcher = matcher
//	return b
//}
//
//// WithStrategy 添加验证策略
//func (b *ValidatorBuilder) WithStrategy(strategy ValidationStrategy) *ValidatorBuilder {
//	b.strategies = append(b.strategies, strategy)
//	return b
//}
//
//// WithRuleStrategy 添加规则验证策略
//func (b *ValidatorBuilder) WithRuleStrategy() *ValidatorBuilder {
//	b.strategies = append(b.strategies, NewRuleStrategy(b.sceneMatcher))
//	return b
//}
//
//// WithBusinessStrategy 添加业务验证策略
//func (b *ValidatorBuilder) WithBusinessStrategy() *ValidatorBuilder {
//	b.strategies = append(b.strategies, NewBusinessStrategy())
//	return b
//}
//
//// WithTypeRegistry 设置类型注册表
//func (b *ValidatorBuilder) WithTypeRegistry(registry TypeRegistry) *ValidatorBuilder {
//	b.typeRegistry = registry
//	return b
//}
//
//// WithListener 添加监听器
//func (b *ValidatorBuilder) WithListener(listener ValidationListener) *ValidatorBuilder {
//	b.listeners = append(b.listeners, listener)
//	return b
//}
//
//// WithErrorFormatter 设置错误格式化器
//func (b *ValidatorBuilder) WithErrorFormatter(formatter ErrorFormatter) *ValidatorBuilder {
//	b.errorFormatter = formatter
//	return b
//}
//
//// WithMaxDepth 设置最大嵌套深度
//func (b *ValidatorBuilder) WithMaxDepth(depth int) *ValidatorBuilder {
//	b.maxDepth = depth
//	return b
//}
//
//// WithMaxErrors 设置最大错误数
//func (b *ValidatorBuilder) WithMaxErrors(maxErrors int) *ValidatorBuilder {
//	b.maxErrors = maxErrors
//	return b
//}
//
//// Build 构建验证器
//func (b *ValidatorBuilder) Build() *ValidatorEngine {
//	engine := NewValidatorEngine(
//		WithStrategies(b.strategies...),
//		WithTypeRegistry(b.typeRegistry),
//		WithSceneMatcher(b.sceneMatcher),
//		WithListeners(b.listeners...),
//		WithErrorFormatter(b.errorFormatter),
//		WithMaxDepth(b.maxDepth),
//		WithMaxErrors(b.maxErrors),
//	)
//
//	return engine
//}
