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
	// 策略编排器
	orchestrator core.IStrategyOrchestrator
	// 拦截器链
	interceptorChain core.IInterceptorChain
	// 错误格式化器
	errorFormatter core.IErrorFormatter
	// 最大错误数
	maxErrors int
	// 最大验证深度
	maxDepth int
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
				errors.NewFieldError("Struct", "", "required"),
			}, e.errorFormatter)
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
			return e.orchestrator.Execute(target, ctx, collector)
		})
	} else {
		validateErr = e.orchestrator.Execute(target, ctx, collector)
	}

	// 如果有执行错误，添加到收集器
	if validateErr != nil {
		collector.Collect(errors.NewFieldErrorWithMessage(validateErr.Error()))
	}

	// 返回验证结果
	if collector.HasErrors() {
		return errors.NewValidationError(collector.Errors(), e.errorFormatter)
	}

	return nil
}

// ValidateWithContext 使用自定义上下文执行验证
// TODO:GG 镶嵌策略用不到的话就删了吧
func (e *validatorEngine) ValidateWithContext(target any, ctx core.IContext) error {
	if target == nil {
		return errors.NewValidationError(
			[]core.IFieldError{
				errors.NewFieldError("Struct", "", "required"),
			}, e.errorFormatter)
	}

	// 创建错误收集器
	collector := errors.AcquireListCollector(e.maxErrors)
	defer errors.ReleaseListCollector(collector)

	// 执行验证
	err := e.orchestrator.Execute(target, ctx, collector)
	if err != nil {
		return err
	}

	// 返回验证结果
	if collector.HasErrors() {
		return errors.NewValidationError(collector.Errors(), e.errorFormatter)
	}

	return nil
}
