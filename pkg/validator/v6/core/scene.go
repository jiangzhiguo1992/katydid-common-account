package core

// Scene 验证场景，使用位运算支持场景组合
// 设计原则：值对象模式，不可变且线程安全
type Scene int64

// 预定义场景
const (
	SceneNone Scene = 0  // 无场景
	SceneAll  Scene = -1 // 所有场景（全 1）
)

// Has 检查是否包含指定场景
// 使用位运算，性能优异 O(1)
func (s Scene) Has(scene Scene) bool {
	// 特殊处理：SceneAll 包含所有场景
	if s == SceneAll {
		return true
	}
	return s&scene != 0
}

// Add 添加场景（返回新场景，不修改原值）
// 不可变设计，避免副作用
func (s Scene) Add(scene Scene) Scene {
	return s | scene
}

// Remove 移除场景（返回新场景，不修改原值）
func (s Scene) Remove(scene Scene) Scene {
	return s &^ scene
}
