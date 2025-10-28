package core

// Scene 验证场景，使用位运算支持场景组合
// 设计思想：使用位掩码实现高性能的场景匹配
type Scene int64

// 预定义场景
const (
	SceneNone Scene = 0  // 无场景
	SceneAll  Scene = -1 // 所有场景
)

// Has 检查是否包含指定场景
// 时间复杂度：O(1)
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

// IsNone 是否无场景
func (s Scene) IsNone() bool {
	return s == SceneNone
}

// IsAll 是否所有场景
func (s Scene) IsAll() bool {
	return s == SceneAll
}

// String 字符串表示（用于调试）
func (s Scene) String() string {
	switch s {
	case SceneNone:
		return "None"
	case SceneAll:
		return "All"
	default:
		return "Custom"
	}
}
