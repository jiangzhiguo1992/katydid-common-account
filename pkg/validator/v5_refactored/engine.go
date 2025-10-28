package v5_refactored

// ============================================================================
// 验证引擎
// ============================================================================

// ValidatorEngine 验证引擎
// 职责：协调各个组件，提供统一的验证入口
// 设计原则：依赖倒置 - 依赖抽象接口，不依赖具体实现
type ValidatorEngine struct {
	// pipeline 管道执行器
	pipeline PipelineExecutor

	// eventBus 事件总线
	eventBus EventBus

	// hookManager 钩子管理器
	hookManager HookManager

	// registry 类型注册表
	registry TypeRegistry

	// collectorFactory 错误收集器工厂
	collectorFactory ErrorCollectorFactory

	// errorFormatter 错误格式化器
	errorFormatter ErrorFormatter

	// maxErrors 最大错误数
	maxErrors int

	// maxDepth 最大嵌套深度
	maxDepth int
}

// NewValidatorEngine 创建验证引擎
// 使用依赖注入，所有依赖都是接口
func NewValidatorEngine(
	pipeline PipelineExecutor,
	eventBus EventBus,
	hookManager HookManager,
	registry TypeRegistry,
	collectorFactory ErrorCollectorFactory,
	errorFormatter ErrorFormatter,
) *ValidatorEngine {
	// 使用默认值填充 nil 依赖
	if pipeline == nil {
		pipeline = NewDefaultPipelineExecutor()
	}
	if eventBus == nil {
		eventBus = NewSyncEventBus()
	}
	if hookManager == nil {
		hookManager = NewDefaultHookManager()
	}
	if registry == nil {
		registry = NewDefaultTypeRegistry()
	}
	if collectorFactory == nil {
		collectorFactory = NewDefaultErrorCollectorFactory(false)
	}
	if errorFormatter == nil {
		errorFormatter = NewDefaultErrorFormatter()
	}

	return &ValidatorEngine{
		pipeline:         pipeline,
		eventBus:         eventBus,
		hookManager:      hookManager,
		registry:         registry,
		collectorFactory: collectorFactory,
		errorFormatter:   errorFormatter,
		maxErrors:        100,
		maxDepth:         100,
	}
}

// Validate 执行完整验证
func (e *ValidatorEngine) Validate(target any, scene Scene) *ValidationError {
	if target == nil {
		return NewValidationError(e.errorFormatter).
			WithError(NewFieldError("target", "required"))
	}

	// 1. 注册类型信息
	e.registry.Register(target)

	// 2. 创建验证上下文
	ctx := AcquireContext(scene, target)
	ctx.WithMaxDepth(e.maxDepth)
	defer ReleaseContext(ctx)

	// 3. 创建错误收集器
	collector := e.collectorFactory.Create(e.maxErrors)

	// 4. 发布验证开始事件
	e.eventBus.Publish(NewBaseEvent(EventValidationStart, ctx))

	// 5. 执行前置钩子
	e.eventBus.Publish(NewBaseEvent(EventHookBefore, ctx))
	if err := e.hookManager.ExecuteBefore(target, ctx); err != nil {
		return NewValidationError(e.errorFormatter).
			WithError(NewFieldErrorWithMessage(err.Error()))
	}

	// 6. 执行验证管道
	if err := e.pipeline.Execute(target, ctx, collector); err != nil {
		// 管道执行失败
		return NewValidationError(e.errorFormatter).
			WithError(NewFieldErrorWithMessage(err.Error()))
	}

	// 7. 执行后置钩子
	e.eventBus.Publish(NewBaseEvent(EventHookAfter, ctx))
	if err := e.hookManager.ExecuteAfter(target, ctx); err != nil {
		return NewValidationError(e.errorFormatter).
			WithError(NewFieldErrorWithMessage(err.Error()))
	}

	// 8. 发布验证结束事件
	e.eventBus.Publish(NewBaseEvent(EventValidationEnd, ctx))

	// 9. 检查是否有错误
	if collector.HasErrors() {
		return NewValidationError(e.errorFormatter).
			WithErrors(collector.GetAll())
	}

	return nil
}

// ValidateFields 验证指定字段
func (e *ValidatorEngine) ValidateFields(target any, scene Scene, fields ...string) *ValidationError {
	if target == nil || len(fields) == 0 {
		return nil
	}

	// 1. 注册类型信息
	e.registry.Register(target)

	// 2. 创建验证上下文
	ctx := AcquireContext(scene, target)
	ctx.WithMaxDepth(e.maxDepth)
	ctx.WithMetadata(MetadataKeyValidateFields, fields)
	defer ReleaseContext(ctx)

	// 3. 创建错误收集器
	collector := e.collectorFactory.Create(e.maxErrors)

	// 4. 执行验证管道
	if err := e.pipeline.Execute(target, ctx, collector); err != nil {
		return NewValidationError(e.errorFormatter).
			WithError(NewFieldErrorWithMessage(err.Error()))
	}

	// 5. 检查是否有错误
	if collector.HasErrors() {
		return NewValidationError(e.errorFormatter).
			WithErrors(collector.GetAll())
	}

	return nil
}

// ValidateFieldsExcept 验证除指定字段外的所有字段
func (e *ValidatorEngine) ValidateFieldsExcept(target any, scene Scene, fields ...string) *ValidationError {
	if target == nil || len(fields) == 0 {
		return e.Validate(target, scene)
	}

	// 1. 注册类型信息
	e.registry.Register(target)

	// 2. 创建验证上下文
	ctx := AcquireContext(scene, target)
	ctx.WithMaxDepth(e.maxDepth)
	ctx.WithMetadata(MetadataKeyExcludeFields, fields)
	defer ReleaseContext(ctx)

	// 3. 创建错误收集器
	collector := e.collectorFactory.Create(e.maxErrors)

	// 4. 发布验证开始事件
	e.eventBus.Publish(NewBaseEvent(EventValidationStart, ctx))

	// 5. 执行前置钩子
	if err := e.hookManager.ExecuteBefore(target, ctx); err != nil {
		return NewValidationError(e.errorFormatter).
			WithError(NewFieldErrorWithMessage(err.Error()))
	}

	// 6. 执行验证管道
	if err := e.pipeline.Execute(target, ctx, collector); err != nil {
		return NewValidationError(e.errorFormatter).
			WithError(NewFieldErrorWithMessage(err.Error()))
	}

	// 7. 执行后置钩子
	if err := e.hookManager.ExecuteAfter(target, ctx); err != nil {
		return NewValidationError(e.errorFormatter).
			WithError(NewFieldErrorWithMessage(err.Error()))
	}

	// 8. 发布验证结束事件
	e.eventBus.Publish(NewBaseEvent(EventValidationEnd, ctx))

	// 9. 检查是否有错误
	if collector.HasErrors() {
		return NewValidationError(e.errorFormatter).
			WithErrors(collector.GetAll())
	}

	return nil
}

// ============================================================================
// 引擎选项
// ============================================================================

// EngineOption 引擎选项
// 设计模式：函数选项模式
type EngineOption func(*ValidatorEngine)

// WithMaxErrors 设置最大错误数
func WithMaxErrors(maxErrors int) EngineOption {
	return func(e *ValidatorEngine) {
		e.maxErrors = maxErrors
	}
}

// WithMaxDepth 设置最大嵌套深度
func WithMaxDepth(maxDepth int) EngineOption {
	return func(e *ValidatorEngine) {
		e.maxDepth = maxDepth
	}
}

// ============================================================================
// 默认实例
// ============================================================================

var (
	defaultEngine *ValidatorEngine
)

// Default 获取默认验证引擎实例（单例）
func Default() *ValidatorEngine {
	if defaultEngine == nil {
		defaultEngine = NewValidatorEngine(
			nil, // 使用默认管道执行器
			nil, // 使用默认事件总线
			nil, // 使用默认钩子管理器
			nil, // 使用默认类型注册表
			nil, // 使用默认错误收集器工厂
			nil, // 使用默认错误格式化器
		)
	}
	return defaultEngine
}

// SetDefault 设置默认验证引擎实例
func SetDefault(engine *ValidatorEngine) {
	defaultEngine = engine
}

// ============================================================================
// 便捷函数
// ============================================================================

// Validate 使用默认引擎进行验证
func Validate(target any, scene Scene) *ValidationError {
	return Default().Validate(target, scene)
}

// ValidateFields 使用默认引擎验证指定字段
func ValidateFields(target any, scene Scene, fields ...string) *ValidationError {
	return Default().ValidateFields(target, scene, fields...)
}

// ValidateFieldsExcept 使用默认引擎验证除指定字段外的所有字段
func ValidateFieldsExcept(target any, scene Scene, fields ...string) *ValidationError {
	return Default().ValidateFieldsExcept(target, scene, fields...)
}
