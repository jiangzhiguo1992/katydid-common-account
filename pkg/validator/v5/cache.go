package v5

import (
	"reflect"
	"sync"
)

// ============================================================================
// TypeRegistry 实现 - 类型注册表
// ============================================================================

// TypeCacheRegistry 默认类型注册表实现
// 职责：管理类型信息缓存
// 设计原则：单一职责、线程安全
type TypeCacheRegistry struct {
	cache sync.Map // key: reflect.Type, value: *TypeInfo
	mu    sync.RWMutex
}

// NewTypeCacheRegistry 创建默认类型注册表
func NewTypeCacheRegistry() *TypeCacheRegistry {
	return &TypeCacheRegistry{}
}

// Register 注册类型信息
func (r *TypeCacheRegistry) Register(target any) *TypeInfo {
	if target == nil {
		return &TypeInfo{}
	}

	typ := reflect.TypeOf(target)
	if typ == nil {
		return &TypeInfo{}
	}

	// 尝试从缓存获取（热路径）
	if cached, ok := r.cache.Load(typ); ok {
		return cached.(*TypeInfo)
	}

	// 缓存未命中，创建新的缓存项（冷路径）
	info := &TypeInfo{}

	// 检查接口实现
	var ruleProvider RuleValidation
	if ruleProvider, info.IsRuleValidator = target.(RuleValidation); info.IsRuleValidator {
		// 预加载常用场景的规则，不用深拷贝验证规则，外部不会修改影响缓存
		info.Rules = ruleProvider.ValidateRules()
	}
	_, info.IsBusinessValidator = target.(BusinessValidation)
	_, info.IsLifecycleHooks = target.(LifecycleHooks)

	// 存入缓存（使用 LoadOrStore 避免并发时的重复存储）
	actual, _ := r.cache.LoadOrStore(typ, info)
	return actual.(*TypeInfo)
}

// Get 获取类型信息
func (r *TypeCacheRegistry) Get(target any) (*TypeInfo, bool) {
	if target == nil {
		return nil, false
	}

	typ := reflect.TypeOf(target)
	if typ == nil {
		return nil, false
	}

	if cached, ok := r.cache.Load(typ); ok {
		return cached.(*TypeInfo), true
	}

	return nil, false
}

// Clear 清除缓存
func (r *TypeCacheRegistry) Clear() {
	r.cache = sync.Map{}
}

// Stats 获取统计信息
func (r *TypeCacheRegistry) Stats() int {
	count := 0
	r.cache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}
