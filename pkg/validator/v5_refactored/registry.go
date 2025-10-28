package v5_refactored

import (
	"reflect"
	"sync"
	"sync/atomic"
)

// ============================================================================
// 类型注册表实现
// ============================================================================

// DefaultTypeRegistry 默认类型注册表
// 职责：分析和缓存类型信息
// 设计原则：单一职责 - 类型信息管理
type DefaultTypeRegistry struct {
	// cache 类型缓存
	cache sync.Map // key: reflect.Type, value: *TypeInfo

	// stats 缓存统计
	hitCount  int64
	missCount int64
}

// NewDefaultTypeRegistry 创建默认类型注册表
func NewDefaultTypeRegistry() *DefaultTypeRegistry {
	return &DefaultTypeRegistry{}
}

// Register 注册并缓存类型信息
func (r *DefaultTypeRegistry) Register(target any) *TypeInfo {
	if target == nil {
		return nil
	}

	typ := reflect.TypeOf(target)
	// 如果是指针，获取其元素类型
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// 先尝试从缓存获取
	if info, ok := r.Get(typ); ok {
		atomic.AddInt64(&r.hitCount, 1)
		return info
	}

	// 缓存未命中，分析类型
	atomic.AddInt64(&r.missCount, 1)
	info := r.Analyze(target)

	// 存入缓存
	r.Set(typ, info)

	return info
}

// Get 获取类型信息
func (r *DefaultTypeRegistry) Get(typ reflect.Type) (*TypeInfo, bool) {
	if val, ok := r.cache.Load(typ); ok {
		return val.(*TypeInfo), true
	}
	return nil, false
}

// Set 设置类型信息
func (r *DefaultTypeRegistry) Set(typ reflect.Type, info *TypeInfo) {
	r.cache.Store(typ, info)
}

// Clear 清空缓存
func (r *DefaultTypeRegistry) Clear() {
	r.cache.Range(func(key, value interface{}) bool {
		r.cache.Delete(key)
		return true
	})
	atomic.StoreInt64(&r.hitCount, 0)
	atomic.StoreInt64(&r.missCount, 0)
}

// Stats 获取统计信息
func (r *DefaultTypeRegistry) Stats() CacheStats {
	size := 0
	r.cache.Range(func(key, value interface{}) bool {
		size++
		return true
	})

	return CacheStats{
		HitCount:  atomic.LoadInt64(&r.hitCount),
		MissCount: atomic.LoadInt64(&r.missCount),
		Size:      size,
	}
}

// Analyze 分析类型
func (r *DefaultTypeRegistry) Analyze(target any) *TypeInfo {
	if target == nil {
		return nil
	}

	info := NewTypeInfo()
	info.Type = reflect.TypeOf(target)

	// 检查是否实现了 RuleProvider 接口
	if provider, ok := target.(RuleProvider); ok {
		info.IsRuleProvider = true
		info.RuleProvider = provider
		// 缓存所有场景的规则
		r.cacheRules(info, provider)
	}

	// 检查是否实现了 BusinessValidator 接口
	if validator, ok := target.(BusinessValidator); ok {
		info.IsBusinessValidator = true
		info.BusinessValidator = validator
	}

	// 检查是否实现了 LifecycleHooks 接口
	if hooks, ok := target.(LifecycleHooks); ok {
		info.IsLifecycleHooks = true
		info.LifecycleHooks = hooks
	}

	return info
}

// cacheRules 缓存规则
func (r *DefaultTypeRegistry) cacheRules(info *TypeInfo, provider RuleProvider) {
	// 预定义的场景列表
	scenes := []Scene{
		SceneCreate,
		SceneUpdate,
		SceneDelete,
		SceneQuery,
		SceneImport,
		SceneExport,
		SceneAll,
	}

	for _, scene := range scenes {
		rules := provider.GetRules(scene)
		if len(rules) > 0 {
			info.Rules[scene] = rules
		}
	}
}

// ============================================================================
// 多级缓存类型注册表（性能优化）
// ============================================================================

// MultiLevelTypeRegistry 多级缓存类型注册表
// 职责：通过多级缓存提升性能
// 设计原则：热点数据优化
type MultiLevelTypeRegistry struct {
	// l1Cache 一级缓存（热点数据，无锁）
	l1Cache sync.Map

	// l2Cache 二级缓存（完整数据，读写锁）
	l2Cache map[reflect.Type]*TypeInfo
	l2Mu    sync.RWMutex

	// maxL1Size 一级缓存最大大小
	maxL1Size int
	l1Size    int64

	// stats 统计
	hitCount  int64
	missCount int64
}

// NewMultiLevelTypeRegistry 创建多级缓存类型注册表
func NewMultiLevelTypeRegistry(maxL1Size int) *MultiLevelTypeRegistry {
	if maxL1Size <= 0 {
		maxL1Size = 100 // 默认一级缓存 100 个
	}

	return &MultiLevelTypeRegistry{
		l2Cache:   make(map[reflect.Type]*TypeInfo),
		maxL1Size: maxL1Size,
	}
}

// Register 注册并缓存类型信息
func (r *MultiLevelTypeRegistry) Register(target any) *TypeInfo {
	if target == nil {
		return nil
	}

	typ := reflect.TypeOf(target)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// 先查询缓存
	if info, ok := r.Get(typ); ok {
		atomic.AddInt64(&r.hitCount, 1)
		return info
	}

	// 缓存未命中，分析类型
	atomic.AddInt64(&r.missCount, 1)
	analyzer := NewDefaultTypeRegistry()
	info := analyzer.Analyze(target)

	// 存入缓存
	r.Set(typ, info)

	return info
}

// Get 获取类型信息
func (r *MultiLevelTypeRegistry) Get(typ reflect.Type) (*TypeInfo, bool) {
	// 1. 先查询一级缓存
	if val, ok := r.l1Cache.Load(typ); ok {
		return val.(*TypeInfo), true
	}

	// 2. 查询二级缓存
	r.l2Mu.RLock()
	info, ok := r.l2Cache[typ]
	r.l2Mu.RUnlock()

	if ok {
		// 提升到一级缓存
		r.promoteToL1(typ, info)
		return info, true
	}

	return nil, false
}

// Set 设置类型信息
func (r *MultiLevelTypeRegistry) Set(typ reflect.Type, info *TypeInfo) {
	// 同时存入两级缓存
	r.l2Mu.Lock()
	r.l2Cache[typ] = info
	r.l2Mu.Unlock()

	r.promoteToL1(typ, info)
}

// promoteToL1 提升到一级缓存
func (r *MultiLevelTypeRegistry) promoteToL1(typ reflect.Type, info *TypeInfo) {
	// 检查一级缓存大小
	if atomic.LoadInt64(&r.l1Size) >= int64(r.maxL1Size) {
		// 一级缓存已满，不再添加（或实现 LRU 淘汰）
		return
	}

	r.l1Cache.Store(typ, info)
	atomic.AddInt64(&r.l1Size, 1)
}

// Clear 清空缓存
func (r *MultiLevelTypeRegistry) Clear() {
	r.l1Cache.Range(func(key, value interface{}) bool {
		r.l1Cache.Delete(key)
		return true
	})

	r.l2Mu.Lock()
	clear(r.l2Cache)
	r.l2Mu.Unlock()

	atomic.StoreInt64(&r.l1Size, 0)
	atomic.StoreInt64(&r.hitCount, 0)
	atomic.StoreInt64(&r.missCount, 0)
}

// Stats 获取统计信息
func (r *MultiLevelTypeRegistry) Stats() CacheStats {
	r.l2Mu.RLock()
	size := len(r.l2Cache)
	r.l2Mu.RUnlock()

	return CacheStats{
		HitCount:  atomic.LoadInt64(&r.hitCount),
		MissCount: atomic.LoadInt64(&r.missCount),
		Size:      size,
	}
}

// Analyze 分析类型
func (r *MultiLevelTypeRegistry) Analyze(target any) *TypeInfo {
	analyzer := NewDefaultTypeRegistry()
	return analyzer.Analyze(target)
}
