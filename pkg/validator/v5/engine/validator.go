package engine

import (
	"fmt"
	"katydid-common-account/pkg/validator/v5/context"
	"katydid-common-account/pkg/validator/v5/core"
	"katydid-common-account/pkg/validator/v5/err"
	"katydid-common-account/pkg/validator/v5/formatter"
	"katydid-common-account/pkg/validator/v5/registry"
	"sort"

	"github.com/go-playground/validator/v10"
)

// ValidatorEngine 验证引擎
// 职责：协调验证流程，编排各个组件
type ValidatorEngine struct {
	// validate 第三方验证器实例
	validator *validator.Validate
	// sceneMatcher 场景匹配器
	sceneMatcher core.ISceneMatcher
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
func NewValidatorEngine(opts ...ValidatorEngineOption) core.IValidator {
	v := validator.New()
	engine := &ValidatorEngine{
		validator:      v,
		sceneMatcher:   core.NewSceneBitMatcher(),
		typeRegistry:   registry.NewTypeRegistry(v),
		strategies:     make([]core.IValidationStrategy, 0),
		listeners:      make([]core.IValidationListener, 0),
		errorFormatter: formatter.NewLocalizesErrorFormatter(),
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

// WithSceneMatcher 设置场景匹配器
func WithSceneMatcher(matcher core.ISceneMatcher) ValidatorEngineOption {
	return func(e *ValidatorEngine) {
		e.sceneMatcher = matcher
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

// GetValidator 获取底层 validator 实例
func (e *ValidatorEngine) GetValidator() *validator.Validate {
	return e.validator
}

// Validate 执行验证
// 职责：编排整个验证流程
func (e *ValidatorEngine) Validate(target any, scene core.Scene) core.IValidationError {
	if target == nil {
		return err.NewValidationError(e.errorFormatter).
			WithError(err.NewFieldError("Struct", "required"))
	}

	// 创建验证上下文
	ctx := context.NewValidationContext(scene, e.maxErrors)
	defer ctx.Release()

	// 触发验证开始事件
	e.notifyValidationStart(ctx)

	// 执行验证
	err := e.ValidateWithContext(target, ctx)

	// 触发验证结束事件
	e.notifyValidationEnd(ctx)

	// 返回验证结果
	if err != nil {
		return err.NewValidationError(e.errorFormatter).WithTotalMessage(err.Error())
	} else if ctx.HasErrors() {
		return err.NewValidationError(e.errorFormatter).WithErrors(ctx.Errors())
	}

	return nil
}

// ValidateWithContext 使用已有上下文执行验证（内部方法）
// 还可用于嵌套验证场景，保持上下文连续性（如深度信息）
func (e *ValidatorEngine) ValidateWithContext(target any, ctx core.IValidationContext) error {
	// 注册类型信息（首次使用时）
	e.typeRegistry.Register(target)

	// 执行生命周期前钩子
	if err := e.executeBeforeHooks(target, ctx); err != nil {
		return err
	}

	// 按优先级执行所有验证策略
	for _, strategy := range e.strategies {
		// 执行策略，捕获 panic
		if err := e.executeStrategyWithRecovery(strategy, target, ctx); err != nil {
			// 检查是否超过最大错误数
			if !ctx.AddError(err.NewFieldErrorWithMessage(err.Error())) {
				break
			}
		}
	}

	// 执行生命周期后钩子
	if err := e.executeAfterHooks(target, ctx); err != nil {
		return err
	}

	return nil
}

// ValidateFields 只验证指定字段
func (e *ValidatorEngine) ValidateFields(target any, scene core.Scene, fields ...string) core.IValidationError {
	if target == nil || len(fields) == 0 {
		return nil
	}

	// 创建验证上下文
	ctx := context.NewValidationContext(scene, e.maxErrors)
	defer ctx.Release()

	// 设置需要验证的字段
	ctx.WithMetadata(core.metadataKeyValidateFields, fields)

	// 只执行规则验证策略
	for _, strategy := range e.strategies {
		if strategy.Type() == core.StrategyTypeRule {
			if err := e.executeStrategyWithRecovery(strategy, target, ctx); err != nil {
				// 检查是否超过最大错误数
				if !ctx.AddError(err.NewFieldErrorWithMessage(err.Error())) {
					break
				}
			}
			break
		}
	}

	// 返回验证结果
	if ctx.HasErrors() {
		return err.NewValidationError(e.errorFormatter).WithErrors(ctx.Errors())
	}

	return nil
}

// ValidateFieldsExcept 验证除指定字段外的所有字段
func (e *ValidatorEngine) ValidateFieldsExcept(target any, scene core.Scene, fields ...string) core.IValidationError {
	if target == nil || len(fields) == 0 {
		return nil
	}

	// 创建验证上下文
	ctx := context.NewValidationContext(scene, e.maxErrors)
	defer ctx.Release()

	// 设置排除验证的字段
	ctx.WithMetadata(core.metadataKeyExcludeFields, fields)

	// 只执行规则验证策略
	for _, strategy := range e.strategies {
		if strategy.Type() == core.StrategyTypeRule {
			if err := e.executeStrategyWithRecovery(strategy, target, ctx); err != nil {
				// 检查是否超过最大错误数
				if !ctx.AddError(err.NewFieldErrorWithMessage(err.Error())) {
					break
				}
			}
			break
		}
	}

	// 返回验证结果
	if ctx.HasErrors() {
		return err.NewValidationError(e.errorFormatter).WithErrors(ctx.Errors())
	}

	return nil
}

// AddStrategy 添加验证策略
// 支持运行时动态添加策略
func (e *ValidatorEngine) AddStrategy(strategy core.IValidationStrategy) {
	e.strategies = append(e.strategies, strategy)
	// 重新排序
	sort.Slice(e.strategies, func(i, j int) bool {
		return e.strategies[i].Priority() < e.strategies[j].Priority()
	})
}

// AddListener 添加监听器
func (e *ValidatorEngine) AddListener(listener core.IValidationListener) {
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
		stats["register_count"] = e.typeRegistry.Stats()
	}
	return stats
}

// executeStrategyWithRecovery 执行策略并捕获 panic
func (e *ValidatorEngine) executeStrategyWithRecovery(
	strategy core.IValidationStrategy,
	target any,
	ctx core.IValidationContext,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("strategy panic: %v", r)
		}
	}()

	return strategy.Validate(target, ctx)
}

// executeBeforeHooks 执行前置钩子
func (e *ValidatorEngine) executeBeforeHooks(target any, ctx core.IValidationContext) error {
	if hooks, ok := target.(core.ILifecycleHooks); ok {
		return hooks.BeforeValidation(ctx)
	}
	return nil
}

// executeAfterHooks 执行后置钩子
func (e *ValidatorEngine) executeAfterHooks(target any, ctx core.IValidationContext) error {
	if hooks, ok := target.(core.ILifecycleHooks); ok {
		return hooks.AfterValidation(ctx)
	}
	return nil
}

// notifyValidationStart 通知验证开始
func (e *ValidatorEngine) notifyValidationStart(ctx core.IValidationContext) {
	for _, listener := range e.listeners {
		listener.OnValidationStart(ctx)
	}
}

// notifyValidationEnd 通知验证结束
func (e *ValidatorEngine) notifyValidationEnd(ctx core.IValidationContext) {
	for _, listener := range e.listeners {
		listener.OnValidationEnd(ctx)
	}
}
