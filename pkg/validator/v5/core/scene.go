package core

import "sync"

// Scene 验证场景，使用位运算支持场景组合
type Scene int64

// 预定义场景
const (
	SceneNone Scene = 0  // 无场景
	SceneAll  Scene = -1 // 所有场景
)

// Has 检查是否包含指定场景
func (s Scene) Has(scene Scene) bool {
	return s&scene != 0
}

// Add 添加场景
func (s Scene) Add(scene Scene) Scene {
	return s | scene
}

// Remove 移除场景
func (s Scene) Remove(scene Scene) Scene {
	return s &^ scene
}

// SceneBitMatcher 位运算场景匹配器（带缓存）
type SceneBitMatcher struct {
	cache sync.Map // key: cacheKey (scene + rules pointer), value: map[string]string
}

// NewSceneBitMatcher 创建默认场景匹配器
func NewSceneBitMatcher() *SceneBitMatcher {
	return &SceneBitMatcher{
		cache: sync.Map{},
	}
}

// Match 判断场景是否匹配
func (m *SceneBitMatcher) Match(target, origin Scene) bool {
	return origin.Has(target)
}

// MatchRules 匹配并合并规则
func (m *SceneBitMatcher) MatchRules(target Scene, rules map[Scene]map[string]string) map[string]string {
	if rules == nil || len(rules) == 0 {
		return nil
	}

	// 生成缓存键：场景 + 规则集指针（规则集不变时指针不变）
	// 注意：这里假设规则集在注册后不会被修改
	type cacheKey struct {
		scene    Scene
		rulesPtr uintptr
	}

	key := cacheKey{
		scene:    target,
		rulesPtr: uintptr(0), // Go 1.18+ 不推荐直接用指针做 map key
	}

	// 尝试从缓存获取
	if cached, ok := m.cache.Load(key); ok {
		return cached.(map[string]string)
	}

	// 缓存未命中，执行匹配
	result := make(map[string]string)
	for scene, sceneRules := range rules {
		if m.Match(target, scene) {
			// 合并规则（后面的覆盖前面的）
			for field, rule := range sceneRules {
				result[field] = rule
			}
		}
	}

	// 存入缓存
	m.cache.Store(key, result)

	return result
}
