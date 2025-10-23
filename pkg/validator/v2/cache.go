package v2

import (
	"reflect"
	"sync"
)

// ============================================================================
// TypeCache 实现 - 线程安全的类型缓存
// ============================================================================

// DefaultTypeCache 默认类型缓存实现
// 设计原则：
//   - 单一职责：只负责类型信息的缓存
//   - 线程安全：使用 sync.Map 保证并发安全
//   - 性能优化：避免重复的反射和接口检查操作
type DefaultTypeCache struct {
	cache sync.Map // key: reflect.Type, value: *TypeInfo
}

// NewTypeCache 创建类型缓存 - 工厂方法
func NewTypeCache() *DefaultTypeCache {
	return &DefaultTypeCache{}
}

// Get 获取类型信息 - 实现 TypeCache 接口
// 如果缓存不存在，会自动创建并缓存
func (c *DefaultTypeCache) Get(obj any) *TypeInfo {
	if obj == nil {
		return NewTypeInfo(nil)
	}

	typ := reflect.TypeOf(obj)
	if typ == nil {
		return NewTypeInfo(nil)
	}

	// 尝试从缓存获取
	if cached, ok := c.cache.Load(typ); ok {
		return cached.(*TypeInfo)
	}

	// 缓存未命中，创建新的类型信息
	info := c.buildTypeInfo(obj, typ)

	// 存入缓存（使用 LoadOrStore 避免并发时的重复存储）
	actual, _ := c.cache.LoadOrStore(typ, info)
	return actual.(*TypeInfo)
}

// Clear 清空缓存 - 实现 TypeCache 接口
func (c *DefaultTypeCache) Clear() {
	c.cache = sync.Map{}
}

// buildTypeInfo 构建类型信息 - 私有方法
// 通过接口检查确定类型实现了哪些验证接口
func (c *DefaultTypeCache) buildTypeInfo(obj any, typ reflect.Type) *TypeInfo {
	info := NewTypeInfo(typ)

	// 检查是否实现了 RuleProvider 接口
	if provider, ok := obj.(RuleProvider); ok {
		info.IsRuleProvider = true
		// 缓存规则，避免重复调用
		info.Rules = provider.ProvideRules()
	}

	// 检查是否实现了 CustomValidator 接口
	_, info.IsCustomValidator = obj.(CustomValidator)

	return info
}

// ============================================================================
// RegistryManager 实现 - 注册状态管理
// ============================================================================

// DefaultRegistryManager 默认注册管理器实现
// 设计原则：
//   - 单一职责：只负责记录类型是否已注册
//   - 线程安全：使用 sync.Map 保证并发安全
type DefaultRegistryManager struct {
	registry sync.Map // key: reflect.Type, value: bool
}

// NewRegistryManager 创建注册管理器 - 工厂方法
func NewRegistryManager() *DefaultRegistryManager {
	return &DefaultRegistryManager{}
}

// IsRegistered 检查类型是否已注册 - 实现 RegistryManager 接口
func (m *DefaultRegistryManager) IsRegistered(obj any) bool {
	if obj == nil {
		return false
	}

	typ := reflect.TypeOf(obj)
	if typ == nil {
		return false
	}

	_, registered := m.registry.Load(typ)
	return registered
}

// MarkRegistered 标记类型已注册 - 实现 RegistryManager 接口
func (m *DefaultRegistryManager) MarkRegistered(obj any) {
	if obj == nil {
		return
	}

	typ := reflect.TypeOf(obj)
	if typ == nil {
		return
	}

	m.registry.Store(typ, true)
}

// Clear 清空注册记录 - 实现 RegistryManager 接口
func (m *DefaultRegistryManager) Clear() {
	m.registry = sync.Map{}
}
