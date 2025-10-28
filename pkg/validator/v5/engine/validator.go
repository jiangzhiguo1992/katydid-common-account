package engine

import (
	"fmt"
	"katydid-common-account/pkg/validator/v5/context"
	"katydid-common-account/pkg/validator/v5/core"
	"katydid-common-account/pkg/validator/v5/err"
	"katydid-common-account/pkg/validator/v5/formatter"
	"sort"
)

// ValidatorEngine 验证引擎
// 职责：协调验证流程，编排各个组件
type ValidatorEngine struct {
	// typeRegistry 类型注册表
	typeRegistry core.ITypeRegistry
	// strategies 验证策略列表
	strategies []core.IValidationStrategy
	// listeners 验证监听器
	listeners []core.IValidationListener
	// errorFormatter 错误格式化器
	errorFormatter core.IErrorFormatter
	// maxDepth 最大嵌套深度
	maxDepth int8
	// maxErrors 最大错误数
	maxErrors int
}

// NewValidatorEngine 创建验证引擎
// 工厂方法，确保对象正确初始化
func NewValidatorEngine(typeRegistry core.ITypeRegistry, opts ...ValidatorEngineOption) core.IValidator {
	engine := &ValidatorEngine{
		typeRegistry:   typeRegistry,
		strategies:     make([]core.IValidationStrategy, 0),
		listeners:      make([]core.IValidationListener, 0),
		errorFormatter: formatter.NewNormalErrorFormatter(),
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

// ValidatorEngineOption 引擎选项
// 设计模式：函数选项模式，支持灵活配置
type ValidatorEngineOption func(*ValidatorEngine)

// WithStrategies 设置验证策略
func WithStrategies(strategies ...core.IValidationStrategy) ValidatorEngineOption {
	return func(e *ValidatorEngine) {
		e.strategies = append(e.strategies, strategies...)
	}
}

// WithRegistry 设置类型注册表
func WithRegistry(registry core.ITypeRegistry) ValidatorEngineOption {
	return func(e *ValidatorEngine) {
		e.typeRegistry = registry
	}
}

// WithListeners 设置监听器
func WithListeners(listeners ...core.IValidationListener) ValidatorEngineOption {
	return func(e *ValidatorEngine) {
		e.listeners = append(e.listeners, listeners...)
	}
}

// WithErrorFormatter 设置错误格式化器
func WithErrorFormatter(formatter core.IErrorFormatter) ValidatorEngineOption {
	return func(e *ValidatorEngine) {
		e.errorFormatter = formatter
	}
}

// WithMaxDepth 设置最大嵌套深度
func WithMaxDepth(depth int8) ValidatorEngineOption {
	return func(e *ValidatorEngine) {
		e.maxDepth = depth
	}
}

// WithMaxErrors 设置最大错误数
func WithMaxErrors(maxErrors int) ValidatorEngineOption {
	return func(e *ValidatorEngine) {
		e.maxErrors = maxErrors
	}
}

// AddStrategy 添加验证策略
// 支持运行时动态添加策略
func (ve *ValidatorEngine) AddStrategy(strategy core.IValidationStrategy) {
	ve.strategies = append(ve.strategies, strategy)
	// 重新排序
	sort.Slice(ve.strategies, func(i, j int) bool {
		return ve.strategies[i].Priority() < ve.strategies[j].Priority()
	})
}

// Validate 执行验证
// 职责：编排整个验证流程
func (ve *ValidatorEngine) Validate(target any, scene core.Scene) core.IValidationError {
	if target == nil {
		return err.NewValidationError(ve.errorFormatter,
			err.WithError(err.NewFieldError("Struct", "required")))
	}

	// 创建验证上下文
	ctx := context.NewValidationContext(scene, ve.maxErrors)
	defer ctx.Release()

	// 触发验证开始事件
	ve.notifyValidationStart(ctx)

	// 执行验证
	e := ve.ValidateWithContext(target, ctx)

	// 触发验证结束事件
	ve.notifyValidationEnd(ctx)

	// 返回验证结果
	if e != nil {
		return err.NewValidationError(ve.errorFormatter, err.WithTotalMessage(e.Error()))
	} else if ctx.HasErrors() {
		return err.NewValidationError(ve.errorFormatter, err.WithErrors(ctx.Errors()))
	}

	return nil
}

// ValidateWithContext 使用已有上下文执行验证（内部方法）
// 还可用于嵌套验证场景，保持上下文连续性（如深度信息）
func (ve *ValidatorEngine) ValidateWithContext(target any, ctx core.IValidationContext) error {
	// 注册类型信息（首次使用时）
	ve.typeRegistry.Register(target)

	// 执行生命周期前钩子
	if e := ve.executeBeforeHooks(target, ctx); e != nil {
		return e
	}

	// 按优先级执行所有验证策略
	for _, strategy := range ve.strategies {
		// 执行策略，捕获 panic
		if e := ve.executeStrategyWithRecovery(strategy, target, ctx); e != nil {
			// 检查是否超过最大错误数
			if !ctx.AddError(err.NewFieldErrorWithMessage(e.Error())) {
				break
			}
		}
	}

	// 执行生命周期后钩子
	if e := ve.executeAfterHooks(target, ctx); e != nil {
		return e
	}

	return nil
}

// ValidateFields 只验证指定字段
func (ve *ValidatorEngine) ValidateFields(target any, scene core.Scene, fields ...string) core.IValidationError {
	if target == nil || len(fields) == 0 {
		return nil
	}

	// 创建验证上下文
	ctx := context.NewValidationContext(scene, ve.maxErrors,
		context.WithAddMetadata(context.MetadataKeyValidateFields, fields))
	defer ctx.Release()

	// 只执行规则验证策略
	for _, strategy := range ve.strategies {
		if strategy.Type() == core.StrategyTypeRule {
			if e := ve.executeStrategyWithRecovery(strategy, target, ctx); e != nil {
				// 检查是否超过最大错误数
				if !ctx.AddError(err.NewFieldErrorWithMessage(e.Error())) {
					break
				}
			}
			break
		}
	}

	// 返回验证结果
	if ctx.HasErrors() {
		return err.NewValidationError(ve.errorFormatter, err.WithErrors(ctx.Errors()))
	}

	return nil
}

// ValidateFieldsExcept 验证除指定字段外的所有字段
func (ve *ValidatorEngine) ValidateFieldsExcept(target any, scene core.Scene, fields ...string) core.IValidationError {
	if target == nil || len(fields) == 0 {
		return nil
	}

	// 创建验证上下文
	ctx := context.NewValidationContext(scene, ve.maxErrors,
		context.WithAddMetadata(context.MetadataKeyExcludeFields, fields))
	defer ctx.Release()

	// 只执行规则验证策略
	for _, strategy := range ve.strategies {
		if strategy.Type() == core.StrategyTypeRule {
			if e := ve.executeStrategyWithRecovery(strategy, target, ctx); e != nil {
				// 检查是否超过最大错误数
				if !ctx.AddError(err.NewFieldErrorWithMessage(e.Error())) {
					break
				}
			}
			break
		}
	}

	// 返回验证结果
	if ctx.HasErrors() {
		return err.NewValidationError(ve.errorFormatter, err.WithErrors(ctx.Errors()))
	}

	return nil
}

// executeStrategyWithRecovery 执行策略并捕获 panic
func (ve *ValidatorEngine) executeStrategyWithRecovery(
	strategy core.IValidationStrategy,
	target any,
	ctx core.IValidationContext,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("strategy panic: %v", r)
		}
	}()

	strategy.Validate(target, ctx)

	return nil
}

// Stats 获取统计信息
func (ve *ValidatorEngine) Stats() map[string]any {
	stats := make(map[string]any)
	stats["strategy_count"] = len(ve.strategies)
	stats["listener_count"] = len(ve.listeners)
	if ve.typeRegistry != nil {
		stats["register_count"] = ve.typeRegistry.Stats()
	}
	return stats
}

// executeBeforeHooks 执行前置钩子
func (ve *ValidatorEngine) executeBeforeHooks(target any, ctx core.IValidationContext) error {
	if hooks, ok := target.(core.ILifecycleHooks); ok {
		return hooks.BeforeValidation(ctx)
	}
	return nil
}

// executeAfterHooks 执行后置钩子
func (ve *ValidatorEngine) executeAfterHooks(target any, ctx core.IValidationContext) error {
	if hooks, ok := target.(core.ILifecycleHooks); ok {
		return hooks.AfterValidation(ctx)
	}
	return nil
}

// notifyValidationStart 通知验证开始
func (ve *ValidatorEngine) notifyValidationStart(ctx core.IValidationContext) {
	for _, listener := range ve.listeners {
		listener.OnValidationStart(ctx)
	}
}

// notifyValidationEnd 通知验证结束
func (ve *ValidatorEngine) notifyValidationEnd(ctx core.IValidationContext) {
	for _, listener := range ve.listeners {
		listener.OnValidationEnd(ctx)
	}
}
