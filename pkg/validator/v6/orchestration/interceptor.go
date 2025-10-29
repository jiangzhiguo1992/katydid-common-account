package orchestration

import (
	"katydid-common-account/pkg/validator/v6/core"
)

// ============================================================================
// 拦截器链实现
// ============================================================================

// interceptorChain 拦截器链实现
// 设计模式：责任链模式
type interceptorChain struct {
	interceptors []core.IInterceptor
}

// NewInterceptorChain 创建拦截器链
func NewInterceptorChain() core.IInterceptorChain {
	return &interceptorChain{
		interceptors: make([]core.IInterceptor, 0),
	}
}

// Add 添加拦截器
func (c *interceptorChain) Add(interceptor core.IInterceptor) {
	c.interceptors = append(c.interceptors, interceptor)
}

// Execute 执行拦截器链
func (c *interceptorChain) Execute(ctx core.IContext, target any, validator func() error) error {
	if len(c.interceptors) == 0 {
		return validator()
	}

	// 构建责任链
	index := 0
	var next func() error

	next = func() error {
		if index >= len(c.interceptors) {
			// 所有拦截器都执行完毕，执行实际的验证
			return validator()
		}

		// 执行当前拦截器
		currentInterceptor := c.interceptors[index]
		index++
		return currentInterceptor.Intercept(ctx, target, next)
	}

	return next()
}

// Clear 清空拦截器链
func (c *interceptorChain) Clear() {
	c.interceptors = c.interceptors[:0]
}

// ============================================================================
// 预定义拦截器
// ============================================================================

// loggingInterceptor 日志拦截器
type loggingInterceptor struct {
	logger Logger
}

// Logger 日志接口
type Logger interface {
	Logf(format string, args ...any)
}

// NewLoggingInterceptor 创建日志拦截器
func NewLoggingInterceptor(logger Logger) core.IInterceptor {
	return &loggingInterceptor{
		logger: logger,
	}
}

// Intercept 实现拦截器接口
func (i *loggingInterceptor) Intercept(ctx core.IContext, target any, next func() error) error {
	i.logger.Logf("验证开始: scene=%v", ctx.Scene())
	err := next()
	if err != nil {
		i.logger.Logf("验证失败: %v", err)
	} else {
		i.logger.Logf("验证成功")
	}
	return err
}

// ============================================================================
// 函数式拦截器
// ============================================================================

// InterceptorFunc 拦截器函数类型
type InterceptorFunc func(ctx core.IContext, target any, next func() error) error

// Intercept 实现拦截器接口
func (f InterceptorFunc) Intercept(ctx core.IContext, target any, next func() error) error {
	return f(ctx, target, next)
}
