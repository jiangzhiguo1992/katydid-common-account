package v2

import (
	"reflect"
	"sync"
)

// ============================================================================
// 类型缓存 - 性能优化
// ============================================================================

// typeCache 类型信息缓存
type typeCache struct {
	// 接口实现标记
	isRuleProvider    bool
	isCustomValidator bool

	// 缓存的验证规则
	validationRules map[Scene]map[string]string
}

// typeCacheManager 类型缓存管理器
type typeCacheManager struct {
	cache sync.Map // key: reflect.Type, value: *typeCache
}

// newTypeCacheManager 创建类型缓存管理器
func newTypeCacheManager() *typeCacheManager {
	return &typeCacheManager{}
}

// Get 获取类型缓存
func (m *typeCacheManager) Get(typ reflect.Type) (*typeCache, bool) {
	if value, ok := m.cache.Load(typ); ok {
		return value.(*typeCache), true
	}
	return nil, false
}

// Set 设置类型缓存
func (m *typeCacheManager) Set(typ reflect.Type, cache *typeCache) {
	m.cache.Store(typ, cache)
}

// Clear 清空缓存
func (m *typeCacheManager) Clear() {
	m.cache = sync.Map{}
}

// getOrCacheTypeInfo 获取或缓存类型信息
func (m *typeCacheManager) getOrCacheTypeInfo(obj interface{}) *typeCache {
	objType := reflect.TypeOf(obj)
	if objType == nil {
		return &typeCache{}
	}

	// 尝试从缓存获取
	if cache, ok := m.Get(objType); ok {
		return cache
	}

	// 创建新的缓存
	cache := &typeCache{}

	// 检查接口实现
	if provider, ok := obj.(RuleProvider); ok {
		cache.isRuleProvider = true
		cache.validationRules = provider.RuleValidation()
	}

	if _, ok := obj.(CustomValidator); ok {
		cache.isCustomValidator = true
	}

	// 存入缓存
	m.Set(objType, cache)

	return cache
}
