package v2

import (
	"reflect"
	"sync"
)

// ============================================================================
// 类型缓存实现 - 单一职责原则（SRP）
// ============================================================================

// typeCache 类型信息缓存的默认实现
type typeCache struct {
	cache sync.Map // key: reflect.Type, value: *TypeMetadata
}

// NewTypeCache 创建类型缓存（工厂方法）
func NewTypeCache() TypeInfoCache {
	return &typeCache{}
}

// Get 获取类型信息
func (c *typeCache) Get(obj any) *TypeMetadata {
	if obj == nil {
		return &TypeMetadata{}
	}

	typ := reflect.TypeOf(obj)
	if typ == nil {
		return &TypeMetadata{}
	}

	// 尝试从缓存获取
	if cached, ok := c.cache.Load(typ); ok {
		return cached.(*TypeMetadata)
	}

	// 构建类型元数据
	metadata := c.buildMetadata(obj)

	// 存入缓存
	actual, _ := c.cache.LoadOrStore(typ, metadata)
	return actual.(*TypeMetadata)
}

// buildMetadata 构建类型元数据
func (c *typeCache) buildMetadata(obj any) *TypeMetadata {
	metadata := &TypeMetadata{}

	// 检查是否实现了 RuleProvider 接口
	if ruleProvider, ok := obj.(RuleProvider); ok {
		metadata.IsRuleProvider = true
		metadata.Rules = ruleProvider.GetRules()
	}

	// 检查是否实现了 BusinessValidator 接口
	_, metadata.IsBusinessValidator = obj.(BusinessValidator)

	return metadata
}

// Clear 清除缓存
func (c *typeCache) Clear() {
	c.cache = sync.Map{}
}
