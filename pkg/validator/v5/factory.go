package v5

import (
	"sync"
)

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

// CreateDefault 创建默认验证器
// 包含所有标准验证策略
func (f *ValidatorFactory) CreateDefault() *ValidatorEngine {
	sceneMatcher := NewDefaultSceneMatcher()

	engine := NewValidatorEngine(
		WithSceneMatcher(sceneMatcher),
		WithMaxDepth(100),
		WithMaxErrors(1000),
	)

	// 添加标准策略
	engine.AddStrategy(NewRuleStrategy(sceneMatcher))
	engine.AddStrategy(NewBusinessStrategy())

	// 注意：嵌套策略需要引擎引用，延迟添加
	nestedStrategy := NewNestedStrategy(engine, 100)
	engine.AddStrategy(nestedStrategy)

	return engine
}

// CreateMinimal 创建最小验证器
// 只包含规则验证策略
func (f *ValidatorFactory) CreateMinimal() *ValidatorEngine {
	sceneMatcher := NewDefaultSceneMatcher()

	return NewValidatorEngine(
		WithStrategies(NewRuleStrategy(sceneMatcher)),
		WithSceneMatcher(sceneMatcher),
	)
}

// CreateCustom 创建自定义验证器
func (f *ValidatorFactory) CreateCustom(strategies ...ValidationStrategy) *ValidatorEngine {
	return NewValidatorEngine(
		WithStrategies(strategies...),
		WithSceneMatcher(NewDefaultSceneMatcher()),
	)
}

// ============================================================================
// ValidatorBuilder - 验证器构建器
// ============================================================================

// ValidatorBuilder 验证器构建器
// 职责：提供流畅的 API 构建复杂配置的验证器
// 设计模式：建造者模式
type ValidatorBuilder struct {
	strategies     []ValidationStrategy
	typeRegistry   TypeRegistry
	sceneMatcher   SceneMatcher
	listeners      []ValidationListener
	errorFormatter ErrorFormatter
	maxDepth       int
	maxErrors      int
}

// NewValidatorBuilder 创建验证器构建器
func NewValidatorBuilder() *ValidatorBuilder {
	return &ValidatorBuilder{
		strategies:     make([]ValidationStrategy, 0),
		typeRegistry:   NewDefaultTypeRegistry(),
		sceneMatcher:   NewDefaultSceneMatcher(),
		listeners:      make([]ValidationListener, 0),
		errorFormatter: NewDefaultErrorFormatter(),
		maxDepth:       100,
		maxErrors:      1000,
	}
}

// WithStrategy 添加验证策略
func (b *ValidatorBuilder) WithStrategy(strategy ValidationStrategy) *ValidatorBuilder {
	b.strategies = append(b.strategies, strategy)
	return b
}

// WithRuleStrategy 添加规则验证策略
func (b *ValidatorBuilder) WithRuleStrategy() *ValidatorBuilder {
	b.strategies = append(b.strategies, NewRuleStrategy(b.sceneMatcher))
	return b
}

// WithBusinessStrategy 添加业务验证策略
func (b *ValidatorBuilder) WithBusinessStrategy() *ValidatorBuilder {
	b.strategies = append(b.strategies, NewBusinessStrategy())
	return b
}

// WithTypeRegistry 设置类型注册表
func (b *ValidatorBuilder) WithTypeRegistry(registry TypeRegistry) *ValidatorBuilder {
	b.typeRegistry = registry
	return b
}

// WithSceneMatcher 设置场景匹配器
func (b *ValidatorBuilder) WithSceneMatcher(matcher SceneMatcher) *ValidatorBuilder {
	b.sceneMatcher = matcher
	return b
}

// WithListener 添加监听器
func (b *ValidatorBuilder) WithListener(listener ValidationListener) *ValidatorBuilder {
	b.listeners = append(b.listeners, listener)
	return b
}

// WithErrorFormatter 设置错误格式化器
func (b *ValidatorBuilder) WithErrorFormatter(formatter ErrorFormatter) *ValidatorBuilder {
	b.errorFormatter = formatter
	return b
}

// WithMaxDepth 设置最大嵌套深度
func (b *ValidatorBuilder) WithMaxDepth(depth int) *ValidatorBuilder {
	b.maxDepth = depth
	return b
}

// WithMaxErrors 设置最大错误数
func (b *ValidatorBuilder) WithMaxErrors(maxErrors int) *ValidatorBuilder {
	b.maxErrors = maxErrors
	return b
}

// Build 构建验证器
func (b *ValidatorBuilder) Build() *ValidatorEngine {
	engine := NewValidatorEngine(
		WithStrategies(b.strategies...),
		WithTypeRegistry(b.typeRegistry),
		WithSceneMatcher(b.sceneMatcher),
		WithListeners(b.listeners...),
		WithErrorFormatter(b.errorFormatter),
		WithMaxDepth(b.maxDepth),
		WithMaxErrors(b.maxErrors),
	)

	return engine
}

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

// ============================================================================
// 便捷函数 - 使用默认验证器
// ============================================================================

// Validate 使用默认验证器验证对象
func Validate(target any, scene Scene) error {
	return Default().Validate(target, scene)
}

// ValidateFields 使用默认验证器验证指定字段
func ValidateFields(target any, scene Scene, fields ...string) error {
	return Default().ValidateFields(target, scene, fields...)
}

// ValidateExcept 使用默认验证器验证排除字段外的所有字段
func ValidateExcept(target any, scene Scene, excludeFields ...string) error {
	return Default().ValidateExcept(target, scene, excludeFields...)
}

// ClearCache 清除默认验证器的缓存
func ClearCache() {
	Default().ClearCache()
}

// Stats 获取默认验证器的统计信息
func Stats() map[string]any {
	return Default().Stats()
}
