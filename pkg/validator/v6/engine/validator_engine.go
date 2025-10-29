package engine

import (
	"katydid-common-account/pkg/validator/v6/context"
	"katydid-common-account/pkg/validator/v6/core"
	"katydid-common-account/pkg/validator/v6/errors"
)

// validatorEngine 验证引擎实现
// 职责：协调整个验证流程
// 设计原则：
// - 单一职责：只负责协调，不负责具体验证
// - 依赖倒置：依赖抽象接口，不依赖具体实现
// - 模板方法：定义验证流程模板
type validatorEngine struct {
	orchestrator     core.IStrategyOrchestrator
	interceptorChain core.IInterceptorChain
	hookExecutor     core.IHookExecutor
	listenerNotifier core.IListenerNotifier
	errorFormatter   core.IErrorFormatter
	maxErrors        int
	maxDepth         int
}

// NewValidatorEngine 创建验证引擎
func NewValidatorEngine(
	orchestrator core.IStrategyOrchestrator,
	opts ...EngineOption,
) core.IValidator {
	engine := &validatorEngine{
		orchestrator: orchestrator,
		maxErrors:    100,
		maxDepth:     50,
	}

	// 应用选项
	for _, opt := range opts {
		opt(engine)
	}

	// 设置默认值
	if engine.errorFormatter == nil {
		engine.errorFormatter = errors.NewDefaultFormatter()
	}

	return engine
}

// EngineOption 引擎选项
type EngineOption func(*validatorEngine)

// WithInterceptorChain 设置拦截器链
func WithInterceptorChain(chain core.IInterceptorChain) EngineOption {
	return func(e *validatorEngine) {
		e.interceptorChain = chain
	}
}

// WithHookExecutor 设置钩子执行器
func WithHookExecutor(executor core.IHookExecutor) EngineOption {
	return func(e *validatorEngine) {
		e.hookExecutor = executor
	}
}

// WithListenerNotifier 设置监听器通知器
func WithListenerNotifier(notifier core.IListenerNotifier) EngineOption {
	return func(e *validatorEngine) {
		e.listenerNotifier = notifier
	}
}

// WithErrorFormatter 设置错误格式化器
func WithErrorFormatter(formatter core.IErrorFormatter) EngineOption {
	return func(e *validatorEngine) {
		e.errorFormatter = formatter
	}
}

// WithMaxErrors 设置最大错误数
func WithMaxErrors(maxErrors int) EngineOption {
	return func(e *validatorEngine) {
		e.maxErrors = maxErrors
	}
}

// WithMaxDepth 设置最大深度
func WithMaxDepth(maxDepth int) EngineOption {
	return func(e *validatorEngine) {
		e.maxDepth = maxDepth
	}
}

// Validate 执行完整验证
// 模板方法：定义验证流程
func (e *validatorEngine) Validate(target any, scene core.Scene) core.IValidationError {
	if target == nil {
		return errors.NewValidationError(
			[]core.IFieldError{
				errors.NewFieldError("", "target", "required",
					errors.WithMessage("validation target cannot be nil")),
			},
			e.errorFormatter,
		)
	}

	// 创建上下文
	ctx := context.NewContext(scene)
	defer ctx.Release()

	// 创建错误收集器
	collector := errors.AcquireListCollector(e.maxErrors)
	defer errors.ReleaseListCollector(collector)

	// 执行验证（带拦截器）
	var validateErr error
	if e.interceptorChain != nil {
		validateErr = e.interceptorChain.Execute(ctx, target, func() error {
			return e.doValidate(target, ctx, collector)
		})
	} else {
		validateErr = e.doValidate(target, ctx, collector)
	}

	// 如果有执行错误，添加到收集器
	if validateErr != nil {
		collector.Collect(errors.NewFieldError("", "", "error",
			errors.WithMessage(validateErr.Error())))
	}

	// 返回验证结果
	if collector.HasErrors() {
		return errors.NewValidationError(collector.Errors(), e.errorFormatter)
	}

	return nil
}

// ValidateWithContext 使用自定义上下文执行验证
func (e *validatorEngine) ValidateWithContext(target any, ctx core.IContext) error {
	if target == nil {
		return errors.NewValidationError(
			[]core.IFieldError{
				errors.NewFieldError("", "target", "required",
					errors.WithMessage("validation target cannot be nil")),
			},
			e.errorFormatter,
		)
	}

	// 创建错误收集器
	collector := errors.AcquireListCollector(e.maxErrors)
	defer errors.ReleaseListCollector(collector)

	// 执行验证
	err := e.doValidate(target, ctx, collector)
	if err != nil {
		return err
	}

	// 返回验证结果
	if collector.HasErrors() {
		return errors.NewValidationError(collector.Errors(), e.errorFormatter)
	}

	return nil
}

// doValidate 执行实际的验证逻辑
// 私有方法：封装验证流程
func (e *validatorEngine) doValidate(target any, ctx core.IContext, collector core.IErrorCollector) error {
	// 1. 通知监听器：验证开始
	if e.listenerNotifier != nil {
		e.listenerNotifier.NotifyStart(ctx, target)
	}

	// 2. 执行前置钩子
	if e.hookExecutor != nil {
		if err := e.hookExecutor.ExecuteBefore(target, ctx); err != nil {
			return err
		}
	}

	// 3. 执行验证策略
	if err := e.orchestrator.Execute(target, ctx, collector); err != nil {
		return err
	}

	// 4. 执行后置钩子
	if e.hookExecutor != nil {
		if err := e.hookExecutor.ExecuteAfter(target, ctx); err != nil {
			return err
		}
	}

	// 5. 通知监听器：验证结束
	if e.listenerNotifier != nil {
		var resultErr error
		if collector.HasErrors() {
			resultErr = errors.NewValidationError(collector.Errors(), e.errorFormatter)
		}
		e.listenerNotifier.NotifyEnd(ctx, target, resultErr)

		// 通知每个错误
		for _, fieldErr := range collector.Errors() {
			e.listenerNotifier.NotifyError(ctx, fieldErr)
		}
	}

	return nil
}
