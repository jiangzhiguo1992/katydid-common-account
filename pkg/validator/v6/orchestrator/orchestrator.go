package orchestrator

import (
	"fmt"

	"katydid-common-account/pkg/validator/v6/context"
	"katydid-common-account/pkg/validator/v6/core"
)

// OrchestratorImpl 验证编排器实现
// 职责：编排验证流程，协调各组件
// 设计原则：单一职责 - 只负责编排，不执行具体验证
type OrchestratorImpl struct {
	strategies      []core.ValidationStrategy
	executor        core.StrategyExecutor
	eventDispatcher core.EventDispatcher
	plugins         []core.Plugin
	maxErrors       int
	maxDepth        int
}

// OrchestratorOption 编排器选项
type OrchestratorOption func(*OrchestratorImpl)

// WithStrategies 设置验证策略
func WithStrategies(strategies ...core.ValidationStrategy) OrchestratorOption {
	return func(o *OrchestratorImpl) {
		o.strategies = append(o.strategies, strategies...)
	}
}

// WithExecutor 设置策略执行器
func WithExecutor(executor core.StrategyExecutor) OrchestratorOption {
	return func(o *OrchestratorImpl) {
		o.executor = executor
	}
}

// WithEventDispatcher 设置事件分发器
func WithEventDispatcher(dispatcher core.EventDispatcher) OrchestratorOption {
	return func(o *OrchestratorImpl) {
		o.eventDispatcher = dispatcher
	}
}

// WithPlugins 设置插件
func WithPlugins(plugins ...core.Plugin) OrchestratorOption {
	return func(o *OrchestratorImpl) {
		o.plugins = append(o.plugins, plugins...)
	}
}

// WithMaxErrors 设置最大错误数
func WithMaxErrors(max int) OrchestratorOption {
	return func(o *OrchestratorImpl) {
		o.maxErrors = max
	}
}

// WithMaxDepth 设置最大深度
func WithMaxDepth(max int) OrchestratorOption {
	return func(o *OrchestratorImpl) {
		o.maxDepth = max
	}
}

// NewOrchestrator 创建验证编排器
func NewOrchestrator(opts ...OrchestratorOption) core.ValidationOrchestrator {
	o := &OrchestratorImpl{
		strategies: make([]core.ValidationStrategy, 0),
		plugins:    make([]core.Plugin, 0),
		maxErrors:  100,
		maxDepth:   100,
	}

	// 应用选项
	for _, opt := range opts {
		opt(o)
	}

	// 如果没有设置执行器，使用默认执行器
	if o.executor == nil {
		o.executor = NewStrategyExecutor()
	}

	return o
}

// Orchestrate 编排验证流程
func (o *OrchestratorImpl) Orchestrate(req *core.ValidationRequest) (*core.ValidationResult, error) {
	// 1. 验证输入
	if req == nil || req.Target == nil {
		return nil, fmt.Errorf("validation request or target is nil")
	}

	// 2. 创建验证上下文
	ctx := context.NewValidationContext(req, o.maxErrors)
	defer ctx.Release()

	// 3. 分发验证开始事件
	o.dispatchEvent(newValidationEvent(core.EventTypeValidationStart, ctx))

	// 4. 执行插件前置钩子
	if err := o.executePluginsBefore(ctx); err != nil {
		return nil, fmt.Errorf("plugin before hook failed: %w", err)
	}

	// 5. 执行所有验证策略
	if err := o.executor.ExecuteAll(o.strategies, req, ctx); err != nil {
		// 策略执行失败
		ctx.ErrorCollector().Add(core.NewFieldError("", "strategy_error").WithMessage(err.Error()))
	}

	// 6. 执行插件后置钩子
	if err := o.executePluginsAfter(ctx); err != nil {
		return nil, fmt.Errorf("plugin after hook failed: %w", err)
	}

	// 7. 分发验证结束事件
	o.dispatchEvent(newValidationEvent(core.EventTypeValidationEnd, ctx))

	// 8. 构建验证结果
	result := o.buildResult(ctx)

	return result, nil
}

// executePluginsBefore 执行插件前置钩子
func (o *OrchestratorImpl) executePluginsBefore(ctx core.ValidationContext) error {
	for _, plugin := range o.plugins {
		if !plugin.Enabled() {
			continue
		}
		if err := plugin.BeforeValidate(ctx); err != nil {
			return fmt.Errorf("plugin %s before hook failed: %w", plugin.Name(), err)
		}
	}
	return nil
}

// executePluginsAfter 执行插件后置钩子
func (o *OrchestratorImpl) executePluginsAfter(ctx core.ValidationContext) error {
	for _, plugin := range o.plugins {
		if !plugin.Enabled() {
			continue
		}
		if err := plugin.AfterValidate(ctx); err != nil {
			return fmt.Errorf("plugin %s after hook failed: %w", plugin.Name(), err)
		}
	}
	return nil
}

// dispatchEvent 分发事件
func (o *OrchestratorImpl) dispatchEvent(event core.ValidationEvent) {
	if o.eventDispatcher != nil {
		o.eventDispatcher.Dispatch(event)
	}
}

// buildResult 构建验证结果
func (o *OrchestratorImpl) buildResult(ctx core.ValidationContext) *core.ValidationResult {
	errors := ctx.ErrorCollector().GetAll()
	result := core.NewValidationResult(!ctx.ErrorCollector().HasErrors())
	result.WithErrors(errors)
	return result
}

// ValidationEventImpl 验证事件实现
type ValidationEventImpl struct {
	eventType core.EventType
	ctx       core.ValidationContext
	timestamp int64
}

// newValidationEvent 创建验证事件
func newValidationEvent(eventType core.EventType, ctx core.ValidationContext) core.ValidationEvent {
	return &ValidationEventImpl{
		eventType: eventType,
		ctx:       ctx,
		timestamp: 0, // 可以使用 time.Now().Unix()
	}
}

// Type 事件类型
func (e *ValidationEventImpl) Type() core.EventType {
	return e.eventType
}

// Context 获取上下文
func (e *ValidationEventImpl) Context() core.ValidationContext {
	return e.ctx
}

// Timestamp 时间戳
func (e *ValidationEventImpl) Timestamp() int64 {
	return e.timestamp
}
