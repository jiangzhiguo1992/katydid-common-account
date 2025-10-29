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
	interceptors []core.Interceptor
}

// NewInterceptorChain 创建拦截器链
func NewInterceptorChain() core.InterceptorChain {
	return &interceptorChain{
		interceptors: make([]core.Interceptor, 0),
	}
}

// Add 添加拦截器
func (c *interceptorChain) Add(interceptor core.Interceptor) {
	c.interceptors = append(c.interceptors, interceptor)
}

// Execute 执行拦截器链
func (c *interceptorChain) Execute(ctx core.Context, target any, validator func() error) error {
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
func NewLoggingInterceptor(logger Logger) core.Interceptor {
	return &loggingInterceptor{
		logger: logger,
	}
}

// Intercept 实现拦截器接口
func (i *loggingInterceptor) Intercept(ctx core.Context, target any, next func() error) error {
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
type InterceptorFunc func(ctx core.Context, target any, next func() error) error

// Intercept 实现拦截器接口
func (f InterceptorFunc) Intercept(ctx core.Context, target any, next func() error) error {
	return f(ctx, target, next)
}

// ============================================================================
// 钩子执行器实现
// ============================================================================

// hookExecutor 钩子执行器实现
// 职责：执行生命周期钩子
type hookExecutor struct {
	inspector core.TypeInspector
}

// NewHookExecutor 创建钩子执行器
func NewHookExecutor(inspector core.TypeInspector) core.HookExecutor {
	return &hookExecutor{
		inspector: inspector,
	}
}

// ExecuteBefore 执行前置钩子
func (h *hookExecutor) ExecuteBefore(target any, ctx core.Context) error {
	// 检查类型信息
	typeInfo := h.inspector.Inspect(target)
	if typeInfo == nil || !typeInfo.IsLifecycleHooks() {
		return nil
	}

	// 执行钩子
	if hooks, ok := target.(core.LifecycleHooks); ok {
		return hooks.BeforeValidation(ctx)
	}

	return nil
}

// ExecuteAfter 执行后置钩子
func (h *hookExecutor) ExecuteAfter(target any, ctx core.Context) error {
	// 检查类型信息
	typeInfo := h.inspector.Inspect(target)
	if typeInfo == nil || !typeInfo.IsLifecycleHooks() {
		return nil
	}

	// 执行钩子
	if hooks, ok := target.(core.LifecycleHooks); ok {
		return hooks.AfterValidation(ctx)
	}

	return nil
}

// ============================================================================
// 监听器通知器实现
// ============================================================================

// listenerNotifier 监听器通知器实现
// 职责：通知所有监听器
type listenerNotifier struct {
	listeners []core.ValidationListener
}

// NewListenerNotifier 创建监听器通知器
func NewListenerNotifier() core.ListenerNotifier {
	return &listenerNotifier{
		listeners: make([]core.ValidationListener, 0),
	}
}

// Register 注册监听器
func (n *listenerNotifier) Register(listener core.ValidationListener) {
	n.listeners = append(n.listeners, listener)
}

// Unregister 注销监听器
func (n *listenerNotifier) Unregister(listener core.ValidationListener) {
	filtered := make([]core.ValidationListener, 0, len(n.listeners))
	for _, l := range n.listeners {
		if l != listener {
			filtered = append(filtered, l)
		}
	}
	n.listeners = filtered
}

// NotifyStart 通知验证开始
func (n *listenerNotifier) NotifyStart(ctx core.Context, target any) {
	for _, listener := range n.listeners {
		listener.OnValidationStart(ctx, target)
	}
}

// NotifyEnd 通知验证结束
func (n *listenerNotifier) NotifyEnd(ctx core.Context, target any, err error) {
	for _, listener := range n.listeners {
		listener.OnValidationEnd(ctx, target, err)
	}
}

// NotifyError 通知错误
func (n *listenerNotifier) NotifyError(ctx core.Context, fieldErr core.FieldError) {
	for _, listener := range n.listeners {
		listener.OnError(ctx, fieldErr)
	}
}
