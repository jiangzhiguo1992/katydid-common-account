package v5_refactored

import "sync"

// ============================================================================
// 钩子管理器
// ============================================================================

// DefaultHookManager 默认钩子管理器
// 职责：管理和执行生命周期钩子
// 设计原则：单一职责 - 只负责钩子管理
type DefaultHookManager struct {
	// hooks 钩子映射（类型 -> 钩子实例）
	hooks map[any]LifecycleHooks

	// mu 保护钩子映射
	mu sync.RWMutex
}

// NewDefaultHookManager 创建默认钩子管理器
func NewDefaultHookManager() *DefaultHookManager {
	return &DefaultHookManager{
		hooks: make(map[any]LifecycleHooks),
	}
}

// ExecuteBefore 执行前置钩子
func (m *DefaultHookManager) ExecuteBefore(target any, ctx *ValidationContext) error {
	// 1. 检查是否实现了 LifecycleHooks 接口
	if hooks, ok := target.(LifecycleHooks); ok {
		if err := m.safeExecute(func() error {
			return hooks.BeforeValidation(ctx)
		}); err != nil {
			return err
		}
	}

	// 2. 检查是否有注册的钩子
	m.mu.RLock()
	defer m.mu.RUnlock()

	if hooks, ok := m.hooks[target]; ok {
		if err := m.safeExecute(func() error {
			return hooks.BeforeValidation(ctx)
		}); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteAfter 执行后置钩子
func (m *DefaultHookManager) ExecuteAfter(target any, ctx *ValidationContext) error {
	// 1. 检查是否实现了 LifecycleHooks 接口
	if hooks, ok := target.(LifecycleHooks); ok {
		if err := m.safeExecute(func() error {
			return hooks.AfterValidation(ctx)
		}); err != nil {
			return err
		}
	}

	// 2. 检查是否有注册的钩子
	m.mu.RLock()
	defer m.mu.RUnlock()

	if hooks, ok := m.hooks[target]; ok {
		if err := m.safeExecute(func() error {
			return hooks.AfterValidation(ctx)
		}); err != nil {
			return err
		}
	}

	return nil
}

// RegisterHook 注册钩子
func (m *DefaultHookManager) RegisterHook(target any, hooks LifecycleHooks) {
	if target == nil || hooks == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.hooks[target] = hooks
}

// UnregisterHook 取消注册钩子
func (m *DefaultHookManager) UnregisterHook(target any) {
	if target == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.hooks, target)
}

// safeExecute 安全执行钩子（捕获 panic）
func (m *DefaultHookManager) safeExecute(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// 将 panic 转换为 error
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = NewFieldErrorWithMessage("hook panic").WithValue(r)
			}
		}
	}()

	return fn()
}

// ============================================================================
// 空钩子管理器（用于禁用钩子）
// ============================================================================

// NoOpHookManager 空钩子管理器
// 职责：什么都不做的钩子管理器，用于性能优化
type NoOpHookManager struct{}

// NewNoOpHookManager 创建空钩子管理器
func NewNoOpHookManager() *NoOpHookManager {
	return &NoOpHookManager{}
}

// ExecuteBefore 执行前置钩子（什么都不做）
func (m *NoOpHookManager) ExecuteBefore(target any, ctx *ValidationContext) error {
	return nil
}

// ExecuteAfter 执行后置钩子（什么都不做）
func (m *NoOpHookManager) ExecuteAfter(target any, ctx *ValidationContext) error {
	return nil
}

// RegisterHook 注册钩子（什么都不做）
func (m *NoOpHookManager) RegisterHook(target any, hooks LifecycleHooks) {}

// UnregisterHook 取消注册钩子（什么都不做）
func (m *NoOpHookManager) UnregisterHook(target any) {}
