package v2

import (
	"reflect"
	"sync"
)

// ============================================================================
// 类型缓存管理 - 单一职责：缓存类型元数据，避免重复反射
// 设计原则：性能优化，线程安全
// ============================================================================

// TypeCache 类型信息缓存
type TypeCache struct {
	// IsRuleProvider 是否实现了 RuleProvider 接口
	IsRuleProvider bool

	// IsCustomValidator 是否实现了 CustomValidator 接口
	IsCustomValidator bool

	// IsErrorMessageProvider 是否实现了 ErrorMessageProvider 接口
	IsErrorMessageProvider bool

	// Rules 缓存的验证规则（按场景）
	Rules map[Scene]map[string]string

	// Type 反射类型
	Type reflect.Type
}

// TypeCacheManager 类型缓存管理器
type TypeCacheManager struct {
	cache sync.Map // key: reflect.Type, value: *TypeCache
	stats TypeCacheStats
	mu    sync.RWMutex
}

// TypeCacheStats 类型缓存统计信息
type TypeCacheStats struct {
	// Hits 缓存命中次数
	Hits int64

	// Misses 缓存未命中次数
	Misses int64

	// Size 缓存大小
	Size int64
}

// NewTypeCacheManager 创建类型缓存管理器
func NewTypeCacheManager() *TypeCacheManager {
	return &TypeCacheManager{}
}

// Get 获取类型缓存
func (m *TypeCacheManager) Get(t reflect.Type) (*TypeCache, bool) {
	if t == nil {
		return nil, false
	}

	if cached, ok := m.cache.Load(t); ok {
		m.incrementHits()
		return cached.(*TypeCache), true
	}

	m.incrementMisses()
	return nil, false
}

// GetOrCreate 获取或创建类型缓存
func (m *TypeCacheManager) GetOrCreate(obj interface{}) *TypeCache {
	if obj == nil {
		return &TypeCache{}
	}

	t := reflect.TypeOf(obj)
	if t == nil {
		return &TypeCache{}
	}

	// 尝试从缓存获取
	if cached, ok := m.Get(t); ok {
		return cached
	}

	// 创建新的缓存项
	cache := m.buildTypeCache(obj, t)

	// 存入缓存（使用 LoadOrStore 避免并发重复创建）
	actual, _ := m.cache.LoadOrStore(t, cache)
	m.incrementSize()

	return actual.(*TypeCache)
}

// buildTypeCache 构建类型缓存
func (m *TypeCacheManager) buildTypeCache(obj interface{}, t reflect.Type) *TypeCache {
	cache := &TypeCache{
		Type:  t,
		Rules: make(map[Scene]map[string]string),
	}

	// 检查接口实现
	if provider, ok := obj.(RuleProvider); ok {
		cache.IsRuleProvider = true
		// 预加载常用场景的规则
		cache.Rules[SceneCreate] = provider.GetRules(SceneCreate)
		cache.Rules[SceneUpdate] = provider.GetRules(SceneUpdate)
	}

	_, cache.IsCustomValidator = obj.(CustomValidator)
	_, cache.IsErrorMessageProvider = obj.(ErrorMessageProvider)

	return cache
}

// Set 设置类型缓存
func (m *TypeCacheManager) Set(t reflect.Type, cache *TypeCache) {
	if t == nil || cache == nil {
		return
	}

	if _, loaded := m.cache.LoadOrStore(t, cache); !loaded {
		m.incrementSize()
	}
}

// Clear 清空缓存
func (m *TypeCacheManager) Clear() {
	m.cache = sync.Map{}
	m.mu.Lock()
	m.stats = TypeCacheStats{}
	m.mu.Unlock()
}

// Remove 移除指定类型的缓存
func (m *TypeCacheManager) Remove(t reflect.Type) {
	if t == nil {
		return
	}

	if _, loaded := m.cache.LoadAndDelete(t); loaded {
		m.decrementSize()
	}
}

// GetStats 获取统计信息
func (m *TypeCacheManager) GetStats() TypeCacheStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// Size 获取缓存大小
func (m *TypeCacheManager) Size() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats.Size
}

// incrementHits 增加命中次数
func (m *TypeCacheManager) incrementHits() {
	m.mu.Lock()
	m.stats.Hits++
	m.mu.Unlock()
}

// incrementMisses 增加未命中次数
func (m *TypeCacheManager) incrementMisses() {
	m.mu.Lock()
	m.stats.Misses++
	m.mu.Unlock()
}

// incrementSize 增加缓存大小
func (m *TypeCacheManager) incrementSize() {
	m.mu.Lock()
	m.stats.Size++
	m.mu.Unlock()
}

// decrementSize 减少缓存大小
func (m *TypeCacheManager) decrementSize() {
	m.mu.Lock()
	if m.stats.Size > 0 {
		m.stats.Size--
	}
	m.mu.Unlock()
}

// HitRate 获取缓存命中率
func (m *TypeCacheManager) HitRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.stats.Hits + m.stats.Misses
	if total == 0 {
		return 0
	}

	return float64(m.stats.Hits) / float64(total)
}

// ============================================================================
// 全局类型缓存管理器
// ============================================================================

var globalTypeCacheManager = NewTypeCacheManager()

// GetGlobalTypeCacheManager 获取全局类型缓存管理器
func GetGlobalTypeCacheManager() *TypeCacheManager {
	return globalTypeCacheManager
}

// ClearTypeCache 清空全局类型缓存
func ClearTypeCache() {
	globalTypeCacheManager.Clear()
}

// GetTypeCacheStats 获取全局类型缓存统计信息
func GetTypeCacheStats() TypeCacheStats {
	return globalTypeCacheManager.GetStats()
}
