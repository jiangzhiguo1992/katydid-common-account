package v5

import (
	"fmt"
	"sort"

	"github.com/go-playground/validator/v10"
)

// ValidatorEngine 验证引擎
// 职责：协调验证流程，编排各个组件
type ValidatorEngine struct {
	// validate 第三方验证器实例
	validator *validator.Validate
	// sceneMatcher 场景匹配器
	sceneMatcher SceneMatcher
	// registry 类型注册表
	registry Registry
	// strategies 验证策略列表
	strategies []ValidationStrategy
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
	v := validator.New()
	engine := &ValidatorEngine{
		validator:      v,
		sceneMatcher:   NewSceneBitMatcher(),
		registry:       NewTypeRegistry(v),
		strategies:     make([]ValidationStrategy, 0),
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

// WithRegistry 设置类型注册表
func WithRegistry(registry Registry) EngineOption {
	return func(e *ValidatorEngine) {
		e.registry = registry
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

// GetValidator 获取底层 validator 实例
func (e *ValidatorEngine) GetValidator() *validator.Validate {
	return e.validator
}

// Validate 执行验证
// 职责：编排整个验证流程
func (e *ValidatorEngine) Validate(target any, scene Scene) error {
	if target == nil {
		return NewValidationError([]*FieldError{
			NewFieldError("Struct", "required").
				WithMessage("validation target cannot be nil"),
		})
	}

	// 创建验证上下文
	ctx := NewValidationContext(scene, target)
	defer ctx.Release()

	// 触发验证开始事件
	e.notifyValidationStart(ctx)

	// 执行验证
	if err := e.validateWithContext(target, ctx); err != nil {
		return err
	}

	// 触发验证结束事件
	e.notifyValidationEnd(ctx)

	// 返回验证结果
	if ctx.HasErrors() {
		return NewValidationError(ctx.GetErrors())
	}

	return nil
}

// validateWithContext 使用已有上下文执行验证（内部方法）
// 还可用于嵌套验证场景，保持上下文连续性（如深度信息）
func (e *ValidatorEngine) validateWithContext(target any, ctx *ValidationContext) error {
	// 注册类型信息（首次使用时）
	e.registry.Register(target)

	// 执行生命周期前钩子
	if err := e.executeBeforeHooks(target, ctx); err != nil {
		return err
	}

	// 按优先级执行所有验证策略
	for _, strategy := range e.strategies {
		// 检查是否超过最大错误数
		if !ctx.CanAddError() {
			break
		}

		// 执行策略，捕获 panic
		if err := e.executeStrategyWithRecovery(strategy, target, ctx); err != nil {
			// 策略执行失败，记录错误但继续执行其他策略
			ctx.AddError(NewFieldErrorWithMsg(err.Error()))
		}
	}

	// 执行生命周期后钩子
	if err := e.executeAfterHooks(target, ctx); err != nil {
		return err
	}

	return nil
}

// ValidateFields 只验证指定字段
func (e *ValidatorEngine) ValidateFields(target any, scene Scene, fields ...string) error {
	if target == nil || len(fields) == 0 {
		return nil
	}

	// 创建验证上下文
	ctx := NewValidationContext(scene, target)
	defer ctx.Release()

	// 设置需要验证的字段
	ctx.WithMetadata("validate_fields", fields)

	// 只执行规则验证策略
	for _, strategy := range e.strategies {
		if strategy.Type() == StrategyTypeRule {
			if err := e.executeStrategyWithRecovery(strategy, target, ctx); err != nil {
				ctx.AddError(NewFieldErrorWithMsg(err.Error()))
			}
			break
		}
	}

	// 返回验证结果
	if ctx.HasErrors() {
		return NewValidationError(ctx.GetErrors())
	}

	return nil
}

// ValidateFieldsExcept 验证除指定字段外的所有字段
func (e *ValidatorEngine) ValidateFieldsExcept(target any, scene Scene, fields ...string) error {
	if target == nil || len(fields) == 0 {
		return nil
	}

	// 创建验证上下文
	ctx := NewValidationContext(scene, target)
	defer ctx.Release()

	// 设置排除验证的字段
	ctx.WithMetadata("exclude_fields", fields)

	// 只执行规则验证策略
	for _, strategy := range e.strategies {
		if strategy.Type() == StrategyTypeRule {
			if err := e.executeStrategyWithRecovery(strategy, target, ctx); err != nil {
				ctx.AddError(NewFieldErrorWithMsg(err.Error()))
			}
			break
		}
	}

	// 返回验证结果
	if ctx.HasErrors() {
		return NewValidationError(ctx.GetErrors())
	}

	return nil
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
	if e.registry != nil {
		e.registry.Clear()
	}
}

// Stats 获取统计信息
func (e *ValidatorEngine) Stats() map[string]any {
	stats := make(map[string]any)
	stats["strategy_count"] = len(e.strategies)
	stats["listener_count"] = len(e.listeners)
	if e.registry != nil {
		stats["register_count"] = e.registry.Stats()
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
