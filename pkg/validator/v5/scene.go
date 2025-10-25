package v5

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

// SceneBitMatcher 位运算场景匹配器
type SceneBitMatcher struct{}

// NewSceneBitMatcher 创建默认场景匹配器
func NewSceneBitMatcher() *SceneBitMatcher {
	return &SceneBitMatcher{}
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

	result := make(map[string]string)

	// 遍历所有规则场景
	for scene, sceneRules := range rules {
		if m.Match(target, scene) {
			// 合并规则（后面的覆盖前面的）
			for field, rule := range sceneRules {
				result[field] = rule
			}
		}
	}

	return result
}
