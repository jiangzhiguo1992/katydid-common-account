package v5

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
func (f *ValidatorFactory) CreateDefault() *ValidatorEngine {
	engine := NewValidatorEngine()

	// 添加标准策略
	engine.AddStrategy(NewRuleStrategy(engine.sceneMatcher, engine.registry))
	engine.AddStrategy(NewBusinessStrategy())
	engine.AddStrategy(NewNestedStrategy(engine, engine.maxDepth))

	// 没有listener
	return engine
}

// CreateMinimal 创建最小验证器
func (f *ValidatorFactory) CreateMinimal() *ValidatorEngine {
	engine := NewValidatorEngine()

	// 只包含规则验证策略
	engine.AddStrategy(NewRuleStrategy(engine.sceneMatcher, engine.registry))

	return engine
}
