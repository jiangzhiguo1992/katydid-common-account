package infrastructure

import "katydid-common-account/pkg/validator/v6/core"

// ============================================================================
// 位运算场景匹配器
// ============================================================================

// bitSceneMatcher 位运算场景匹配器
// 使用位运算实现高性能场景匹配
type bitSceneMatcher struct{}

// NewBitSceneMatcher 创建位运算场景匹配器
func NewBitSceneMatcher() core.SceneMatcher {
	return &bitSceneMatcher{}
}

// Match 判断场景是否匹配
func (m *bitSceneMatcher) Match(target, current core.Scene) bool {
	// SceneAll 匹配所有场景
	if current == core.SceneAll {
		return true
	}
	// 使用位运算检查
	return current.Has(target)
}

// MergeRules 合并多个场景的规则
func (m *bitSceneMatcher) MergeRules(current core.Scene, rules map[core.Scene]map[string]string) map[string]string {
	if rules == nil || len(rules) == 0 {
		return nil
	}

	result := make(map[string]string)

	// 遍历所有场景规则
	for scene, sceneRules := range rules {
		// 检查场景是否匹配
		if m.Match(current, scene) {
			// 合并规则（后面的覆盖前面的）
			for field, rule := range sceneRules {
				result[field] = rule
			}
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// ============================================================================
// 精确场景匹配器
// ============================================================================

// exactSceneMatcher 精确场景匹配器
// 只匹配完全相等的场景
type exactSceneMatcher struct{}

// NewExactSceneMatcher 创建精确场景匹配器
func NewExactSceneMatcher() core.SceneMatcher {
	return &exactSceneMatcher{}
}

// Match 判断场景是否匹配（精确匹配）
func (m *exactSceneMatcher) Match(target, current core.Scene) bool {
	return target.Equals(current) || current == core.SceneAll
}

// MergeRules 合并规则（精确匹配）
func (m *exactSceneMatcher) MergeRules(current core.Scene, rules map[core.Scene]map[string]string) map[string]string {
	if rules == nil || len(rules) == 0 {
		return nil
	}

	// 先查找精确匹配
	if sceneRules, ok := rules[current]; ok {
		return sceneRules
	}

	// 再查找 SceneAll
	if sceneRules, ok := rules[core.SceneAll]; ok {
		return sceneRules
	}

	return nil
}

// ============================================================================
// 缓存场景匹配器
// ============================================================================

// cachedSceneMatcher 带缓存的场景匹配器
// 装饰器模式：为其他匹配器添加缓存功能
type cachedSceneMatcher struct {
	matcher core.SceneMatcher
	cache   core.CacheManager
}

// NewCachedSceneMatcher 创建带缓存的场景匹配器
func NewCachedSceneMatcher(matcher core.SceneMatcher, cache core.CacheManager) core.SceneMatcher {
	if cache == nil {
		cache = NewSimpleCache()
	}
	return &cachedSceneMatcher{
		matcher: matcher,
		cache:   cache,
	}
}

// Match 判断场景是否匹配
func (m *cachedSceneMatcher) Match(target, current core.Scene) bool {
	return m.matcher.Match(target, current)
}

// MergeRules 合并规则（带缓存）
func (m *cachedSceneMatcher) MergeRules(current core.Scene, rules map[core.Scene]map[string]string) map[string]string {
	// 生成缓存键
	cacheKey := m.buildCacheKey(current, rules)

	// 尝试从缓存获取
	if cached, ok := m.cache.Get(cacheKey); ok {
		return cached.(map[string]string)
	}

	// 缓存未命中，执行匹配
	result := m.matcher.MergeRules(current, rules)

	// 存入缓存
	if result != nil {
		m.cache.Set(cacheKey, result)
	}

	return result
}

// buildCacheKey 构建缓存键
func (m *cachedSceneMatcher) buildCacheKey(scene core.Scene, rules map[core.Scene]map[string]string) any {
	// 简单策略：使用场景值作为键
	// 更好的做法是使用 scene + rules 的哈希
	type cacheKey struct {
		scene     core.Scene
		rulesSize int
	}

	return cacheKey{
		scene:     scene,
		rulesSize: len(rules),
	}
}
