package domain

import "fmt"

// IDSet ID集合类型
//
// 特性：
//   - 自动去重（map的天然特性）
//   - 高效查找（O(1)时间复杂度）
//   - 支持标准集合操作（并集、交集、差集）
//   - 可转换为切片
//
// 使用场景：
//   - 需要频繁查找ID是否存在
//   - 需要对ID进行集合运算
//   - 自动去重的ID容器
type IDSet map[ID]struct{}

// NewIDSet 创建新的ID集合
// 说明：从可变参数列表创建集合，自动去重
func NewIDSet(ids ...ID) IDSet {
	if ids == nil {
		return make(IDSet)
	}

	// 长度限制：防止内存耗尽
	if len(ids) > maxSliceLength {
		ids = ids[:maxSliceLength]
	}

	set := make(IDSet, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

// Add 添加ID到集合
func (s IDSet) Add(id ID) {
	// 容量限制：防止内存耗尽
	if len(s) >= maxSliceLength {
		return
	}
	s[id] = struct{}{}
}

// Remove 从集合中移除ID
func (s IDSet) Remove(id ID) {
	delete(s, id)
}

// Contains 检查集合是否包含指定ID
func (s IDSet) Contains(id ID) bool {
	_, exists := s[id]
	return exists
}

// Size 获取集合大小
func (s IDSet) Size() int {
	return len(s)
}

// IsEmpty 检查集合是否为空
func (s IDSet) IsEmpty() bool {
	return len(s) == 0
}

// Clear 清空集合中的所有元素
func (s IDSet) Clear() {
	for id := range s {
		delete(s, id)
	}
}

// ToSlice 转换为ID切片
// 说明：创建新的切片，包含集合中的所有ID
func (s IDSet) ToSlice() IDSlice {
	result := make(IDSlice, 0, len(s))
	for id := range s {
		result = append(result, id)
	}
	return result
}

// Clone 克隆集合
// 设计原则：不可变性 - 返回独立副本
func (s IDSet) Clone() IDSet {
	result := make(IDSet, len(s))
	for id := range s {
		result[id] = struct{}{}
	}
	return result
}

// Union 返回两个集合的并集
func (s IDSet) Union(other IDSet) IDSet {
	// nil检查：如果other为nil，返回当前集合的副本
	if other == nil {
		result := make(IDSet, len(s))
		for id := range s {
			result[id] = struct{}{}
		}
		return result
	}

	// 防止结果集合过大
	estimatedSize := len(s) + len(other)
	if estimatedSize > maxSliceLength {
		estimatedSize = maxSliceLength
	}

	result := make(IDSet, estimatedSize)
	count := 0

	// 添加当前集合的所有元素
	for id := range s {
		if count >= maxSliceLength {
			break
		}
		result[id] = struct{}{}
		count++
	}

	// 添加另一个集合的所有元素（map自动去重）
	for id := range other {
		if count >= maxSliceLength {
			break
		}
		result[id] = struct{}{}
		count++
	}

	return result
}

// Intersect 返回两个集合的交集
func (s IDSet) Intersect(other IDSet) IDSet {
	// nil检查：如果other为nil，交集为空
	if other == nil {
		return make(IDSet)
	}

	// 优化：选择较小的集合进行遍历
	smaller, larger := s, other
	if len(other) < len(s) {
		smaller, larger = other, s
	}

	result := make(IDSet)
	for id := range smaller {
		if larger.Contains(id) {
			result[id] = struct{}{}
		}
	}
	return result
}

// Difference 返回两个集合的差集
func (s IDSet) Difference(other IDSet) IDSet {
	// nil检查：如果other为nil，返回当前集合的副本
	if other == nil {
		result := make(IDSet, len(s))
		for id := range s {
			result[id] = struct{}{}
		}
		return result
	}

	result := make(IDSet)
	for id := range s {
		if !other.Contains(id) {
			result[id] = struct{}{}
		}
	}
	return result
}

// Equal 检查两个集合是否相等
func (s IDSet) Equal(other IDSet) bool {
	// nil处理
	if s == nil && other == nil {
		return true
	}
	if s == nil || other == nil {
		return false
	}

	// 大小不同，必定不相等
	if len(s) != len(other) {
		return false
	}

	// 逐一检查元素
	for id := range s {
		if !other.Contains(id) {
			return false
		}
	}
	return true
}

// ValidateAll 验证集合中所有ID的有效性
// 说明：遇到第一个无效ID时立即返回错误（快速失败）
func (s IDSet) ValidateAll() error {
	for id := range s {
		if err := id.Validate(); err != nil {
			return fmt.Errorf("invalid ID %d: %w", id, err)
		}
	}
	return nil
}
