package v5

import (
	"fmt"
	"sort"
	"sync"
)

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

// ============================================================================
// ValidatorEngine - 验证引擎
// ============================================================================

// ValidatorEngine 验证引擎
// 职责：协调验证流程，编排各个组件
// 设计原则：
//   - 单一职责：只负责流程编排
//   - 依赖倒置：依赖抽象接口，不依赖具体实现
//   - 开放封闭：通过策略扩展功能
type ValidatorEngine struct {
	// strategies 验证策略列表
	strategies []ValidationStrategy
	// typeRegistry 类型注册表
	typeRegistry TypeRegistry
	// sceneMatcher 场景匹配器
	sceneMatcher SceneMatcher
	// listeners 验证监听器
	listeners []ValidationListener
	// errorFormatter 错误格式化器
	errorFormatter ErrorFormatter
	// maxDepth 最大嵌套深度
	maxDepth int
	// maxErrors 最大错误数
	maxErrors int
}

// NewValidatorEngine 创建验证引擎
// 工厂方法，确保对象正确初始化
func NewValidatorEngine(opts ...EngineOption) *ValidatorEngine {
	engine := &ValidatorEngine{
		strategies:     make([]ValidationStrategy, 0),
		typeRegistry:   NewTypeCacheRegistry(),
		sceneMatcher:   NewSceneBitMatcher(),
		listeners:      make([]ValidationListener, 0),
		errorFormatter: NewDefaultErrorFormatter(),
		maxDepth:       100,
		maxErrors:      100,
	}

	// 应用选项
	for _, opt := range opts {
		opt(engine)
	}

	// 按优先级排序策略
	sort.Slice(engine.strategies, func(i, j int) bool {
		return engine.strategies[i].Priority() < engine.strategies[j].Priority()
	})

	return engine
}

// EngineOption 引擎选项
// 设计模式：函数选项模式，支持灵活配置
type EngineOption func(*ValidatorEngine)

// WithStrategies 设置验证策略
func WithStrategies(strategies ...ValidationStrategy) EngineOption {
	return func(e *ValidatorEngine) {
		e.strategies = append(e.strategies, strategies...)
	}
}

// WithTypeRegistry 设置类型注册表
func WithTypeRegistry(registry TypeRegistry) EngineOption {
	return func(e *ValidatorEngine) {
		e.typeRegistry = registry
	}
}

// WithSceneMatcher 设置场景匹配器
func WithSceneMatcher(matcher SceneMatcher) EngineOption {
	return func(e *ValidatorEngine) {
		e.sceneMatcher = matcher
	}
}

// WithListeners 设置监听器
func WithListeners(listeners ...ValidationListener) EngineOption {
	return func(e *ValidatorEngine) {
		e.listeners = append(e.listeners, listeners...)
	}
}

// WithErrorFormatter 设置错误格式化器
func WithErrorFormatter(formatter ErrorFormatter) EngineOption {
	return func(e *ValidatorEngine) {
		e.errorFormatter = formatter
	}
}

// WithMaxDepth 设置最大嵌套深度
func WithMaxDepth(depth int) EngineOption {
	return func(e *ValidatorEngine) {
		e.maxDepth = depth
	}
}

// WithMaxErrors 设置最大错误数
func WithMaxErrors(maxErrors int) EngineOption {
	return func(e *ValidatorEngine) {
		e.maxErrors = maxErrors
	}
}

// Validate 执行验证
// 职责：编排整个验证流程
// 流程：
//  1. 创建验证上下文
//  2. 触发验证前钩子
//  3. 注册类型信息
//  4. 按优先级执行验证策略
//  5. 触发验证后钩子
//  6. 返回验证结果
func (e *ValidatorEngine) Validate(target any, scene Scene) error {
	// 防御性编程：参数校验
	if target == nil {
		return NewValidationError([]*FieldError{
			NewFieldError("struct", "required").
				WithMessage("validation target cannot be nil"),
		})
	}

	// 1. 创建验证上下文
	ctx := NewValidationContext(scene, target)

	// 2. 触发验证开始事件 TODO:GG 干嘛的
	e.notifyValidationStart(ctx)

	// 3. 注册类型信息（首次使用时）
	e.typeRegistry.Register(target)

	// 4. 执行生命周期前钩子 TODO:GG 干嘛的
	if err := e.executeBeforeHooks(target, ctx); err != nil {
		return err
	}

	// 5. 按优先级执行所有验证策略
	for _, strategy := range e.strategies {
		// 检查是否超过最大错误数 TODO:GG 应该在errAdd的时候加
		if ctx.ErrorCount() >= e.maxErrors {
			break
		}

		// 执行策略，捕获 panic
		if err := e.executeStrategyWithRecovery(strategy, target, ctx); err != nil {
			// 策略执行失败，记录错误但继续执行其他策略
			ctx.AddError(NewFieldError("", strategy.Name()).
				WithMessage(fmt.Sprintf("strategy %s failed: %v", strategy.Name(), err)))
		}
	}

	// 6. 执行生命周期后钩子
	if err := e.executeAfterHooks(target, ctx); err != nil {
		return err
	}

	// 7. 触发验证结束事件
	e.notifyValidationEnd(ctx)

	// 8. 返回验证结果
	if ctx.HasErrors() {
		return NewValidationError(ctx.GetErrors())
	}

	return nil
}

// validateWithContext 使用已有上下文执行验证（内部方法）
// 用于嵌套验证场景，保持上下文连续性（如深度信息）
func (e *ValidatorEngine) validateWithContext(target any, ctx *ValidationContext) error {
	if target == nil || ctx == nil {
		return nil
	}

	// 注册类型信息（首次使用时）
	e.typeRegistry.Register(target)

	// 执行生命周期前钩子
	if err := e.executeBeforeHooks(target, ctx); err != nil {
		return err
	}

	// 按优先级执行所有验证策略
	for _, strategy := range e.strategies {
		// 检查是否超过最大错误数 TODO:GG 应该在errAdd的时候加
		if ctx.ErrorCount() >= e.maxErrors {
			break
		}

		// 执行策略，捕获 panic
		// TODO:GG 嵌套的字段里面，rule+custom还要触发吗，或者是会正确触发吗？
		if err := e.executeStrategyWithRecovery(strategy, target, ctx); err != nil {
			// 策略执行失败，记录错误但继续执行其他策略
			ctx.AddError(NewFieldError("", strategy.Name()).
				WithMessage(fmt.Sprintf("strategy %s failed: %v", strategy.Name(), err)))
		}
	}

	// 执行生命周期后钩子
	if err := e.executeAfterHooks(target, ctx); err != nil {
		return err
	}

	return nil
}

// ValidateFields 只验证指定字段 TODO:GG 多余?
func (e *ValidatorEngine) ValidateFields(target any, scene Scene, fields ...string) error {
	if len(fields) == 0 {
		return nil
	}

	ctx := NewValidationContext(scene, target)
	ctx.WithMetadata("validate_fields", fields)

	// 只执行规则验证策略
	for _, strategy := range e.strategies {
		if strategy.Name() == "rule" {
			if err := e.executeStrategyWithRecovery(strategy, target, ctx); err != nil {
				ctx.AddError(NewFieldError("", strategy.Name()).
					WithMessage(err.Error()))
			}
			break
		}
	}

	if ctx.HasErrors() {
		return NewValidationError(ctx.GetErrors())
	}
	return nil
}

// ValidateExcept 验证除指定字段外的所有字段
func (e *ValidatorEngine) ValidateExcept(target any, scene Scene, excludeFields ...string) error {
	ctx := NewValidationContext(scene, target)
	if len(excludeFields) > 0 {
		ctx.WithMetadata("exclude_fields", excludeFields)
	}

	return e.Validate(target, scene)
}

// AddStrategy 添加验证策略
// 支持运行时动态添加策略
func (e *ValidatorEngine) AddStrategy(strategy ValidationStrategy) {
	e.strategies = append(e.strategies, strategy)
	// 重新排序
	sort.Slice(e.strategies, func(i, j int) bool {
		return e.strategies[i].Priority() < e.strategies[j].Priority()
	})
}

// AddListener 添加监听器
func (e *ValidatorEngine) AddListener(listener ValidationListener) {
	e.listeners = append(e.listeners, listener)
}

// ClearCache 清除缓存
func (e *ValidatorEngine) ClearCache() {
	if e.typeRegistry != nil {
		e.typeRegistry.Clear()
	}
}

// Stats 获取统计信息
func (e *ValidatorEngine) Stats() map[string]any {
	stats := make(map[string]any)
	stats["strategy_count"] = len(e.strategies)
	stats["listener_count"] = len(e.listeners)
	if e.typeRegistry != nil {
		stats["type_cache_count"] = e.typeRegistry.Stats()
	}
	return stats
}

// executeStrategyWithRecovery 执行策略并捕获 panic
func (e *ValidatorEngine) executeStrategyWithRecovery(
	strategy ValidationStrategy,
	target any,
	ctx *ValidationContext,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("strategy panic: %v", r)
		}
	}()

	return strategy.Validate(target, ctx)
}

// executeBeforeHooks 执行前置钩子
func (e *ValidatorEngine) executeBeforeHooks(target any, ctx *ValidationContext) error {
	if hooks, ok := target.(LifecycleHooks); ok {
		return hooks.BeforeValidation(ctx)
	}
	return nil
}

// executeAfterHooks 执行后置钩子
func (e *ValidatorEngine) executeAfterHooks(target any, ctx *ValidationContext) error {
	if hooks, ok := target.(LifecycleHooks); ok {
		return hooks.AfterValidation(ctx)
	}
	return nil
}

// notifyValidationStart 通知验证开始
func (e *ValidatorEngine) notifyValidationStart(ctx *ValidationContext) {
	for _, listener := range e.listeners {
		listener.OnValidationStart(ctx)
	}
}

// notifyValidationEnd 通知验证结束
func (e *ValidatorEngine) notifyValidationEnd(ctx *ValidationContext) {
	for _, listener := range e.listeners {
		listener.OnValidationEnd(ctx)
	}
}
