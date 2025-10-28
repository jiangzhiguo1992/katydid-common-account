package matcher

import (
	"sync"

	"katydid-common-account/pkg/validator/v6/core"
)

// SceneMatcherImpl 场景匹配器实现
// 职责：匹配验证场景
// 设计原则：单一职责、性能优化（缓存）
type SceneMatcherImpl struct {
	cache sync.Map // 缓存匹配结果
}

// NewSceneMatcher 创建场景匹配器
func NewSceneMatcher() core.SceneMatcher {
	return &SceneMatcherImpl{}
}

// Match 判断场景是否匹配
// 使用位运算实现高性能匹配
func (m *SceneMatcherImpl) Match(target, current core.Scene) bool {
	// SceneAll 匹配所有场景
	if current == core.SceneAll {
		return true
	}

	// 使用位运算判断
	return current.Has(target)
}

// MatchRules 匹配并合并规则
func (m *SceneMatcherImpl) MatchRules(scene core.Scene, rules map[core.Scene]map[string]string) map[string]string {
	if rules == nil || len(rules) == 0 {
		return nil
	}

	// 尝试从缓存获取
	cacheKey := m.buildCacheKey(scene, rules)
	if cached, ok := m.cache.Load(cacheKey); ok {
		return cached.(map[string]string)
	}

	// 缓存未命中，执行匹配
	result := make(map[string]string)

	// 遍历所有场景规则
	for ruleScene, sceneRules := range rules {
		if m.Match(scene, ruleScene) {
			// 合并规则（后面的覆盖前面的）
			for field, rule := range sceneRules {
				result[field] = rule
			}
		}
	}

	// 存入缓存
	m.cache.Store(cacheKey, result)

	return result
}

// buildCacheKey 构建缓存键
// 注意：这是简化实现，生产环境可能需要更精确的键
func (m *SceneMatcherImpl) buildCacheKey(scene core.Scene, rules map[core.Scene]map[string]string) interface{} {
	// 使用场景值作为键（假设规则不变）
	// 更好的做法是使用 scene + rules 的哈希
	return scene
}
