package v5_refactored

// ============================================================================
// 验证器建造者
// ============================================================================

// DefaultValidatorBuilder 默认验证器建造者
// 职责：提供流畅的 API 构建验证器
// 设计模式：建造者模式
type DefaultValidatorBuilder struct {
	pipeline         PipelineExecutor
	eventBus         EventBus
	hookManager      HookManager
	registry         TypeRegistry
	collectorFactory ErrorCollectorFactory
	errorFormatter   ErrorFormatter
	maxErrors        int
	maxDepth         int
}

// NewValidatorBuilder 创建验证器建造者
func NewValidatorBuilder() *DefaultValidatorBuilder {
	return &DefaultValidatorBuilder{
		maxErrors: 100,
		maxDepth:  100,
	}
}

// WithPipeline 设置管道执行器
func (b *DefaultValidatorBuilder) WithPipeline(pipeline PipelineExecutor) ValidatorBuilder {
	b.pipeline = pipeline
	return b
}

// WithEventBus 设置事件总线
func (b *DefaultValidatorBuilder) WithEventBus(bus EventBus) ValidatorBuilder {
	b.eventBus = bus
	return b
}

// WithHookManager 设置钩子管理器
func (b *DefaultValidatorBuilder) WithHookManager(manager HookManager) ValidatorBuilder {
	b.hookManager = manager
	return b
}

// WithRegistry 设置类型注册表
func (b *DefaultValidatorBuilder) WithRegistry(registry TypeRegistry) ValidatorBuilder {
	b.registry = registry
	return b
}

// WithErrorCollectorFactory 设置错误收集器工厂
func (b *DefaultValidatorBuilder) WithErrorCollectorFactory(factory ErrorCollectorFactory) ValidatorBuilder {
	b.collectorFactory = factory
	return b
}

// WithErrorFormatter 设置错误格式化器
func (b *DefaultValidatorBuilder) WithErrorFormatter(formatter ErrorFormatter) ValidatorBuilder {
	b.errorFormatter = formatter
	return b
}

// WithMaxErrors 设置最大错误数
func (b *DefaultValidatorBuilder) WithMaxErrors(max int) ValidatorBuilder {
	b.maxErrors = max
	return b
}

// WithMaxDepth 设置最大嵌套深度
func (b *DefaultValidatorBuilder) WithMaxDepth(depth int) ValidatorBuilder {
	b.maxDepth = depth
	return b
}

// Build 构建验证器
func (b *DefaultValidatorBuilder) Build() Validator {
	engine := NewValidatorEngine(
		b.pipeline,
		b.eventBus,
		b.hookManager,
		b.registry,
		b.collectorFactory,
		b.errorFormatter,
	)

	engine.maxErrors = b.maxErrors
	engine.maxDepth = b.maxDepth

	return engine
}

// ============================================================================
// 验证器工厂
// ============================================================================

// DefaultValidatorFactory 默认验证器工厂
// 职责：创建验证器实例
// 设计模式：工厂模式
type DefaultValidatorFactory struct{}

// NewDefaultValidatorFactory 创建默认验证器工厂
func NewDefaultValidatorFactory() *DefaultValidatorFactory {
	return &DefaultValidatorFactory{}
}

// Create 创建验证器
func (f *DefaultValidatorFactory) Create(opts ...EngineOption) Validator {
	engine := NewValidatorEngine(nil, nil, nil, nil, nil, nil)

	// 应用选项
	for _, opt := range opts {
		opt(engine)
	}

	return engine
}

// CreateDefault 创建默认验证器
func (f *DefaultValidatorFactory) CreateDefault() Validator {
	return NewValidatorEngine(nil, nil, nil, nil, nil, nil)
}

// ============================================================================
// 便捷构建函数
// ============================================================================

// NewBuilder 创建建造者
func NewBuilder() ValidatorBuilder {
	return NewValidatorBuilder()
}

// NewFactory 创建工厂
func NewFactory() ValidatorFactory {
	return NewDefaultValidatorFactory()
}
