package v3

// Scene 验证场景类型 - 使用位掩码支持组合场景
type Scene uint64

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
