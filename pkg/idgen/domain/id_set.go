package domain

import "fmt"

// IDSet ID集合类型，提供集合操作（使用map实现，查找性能O(1)）
type IDSet map[ID]struct{}

// NewIDSet 创建新的ID集合
func NewIDSet(ids ...ID) IDSet {
	if ids == nil {
		return make(IDSet)
	}

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

// ToSlice 转换为ID切片
func (s IDSet) ToSlice() IDSlice {
	result := make(IDSlice, 0, len(s))
	for id := range s {
		result = append(result, id)
	}
	return result
}

// Union 返回两个集合的并集（不修改原集合，返回新集合（不可变性））
func (s IDSet) Union(other IDSet) IDSet {
	// nil 检查：如果 other 为 nil，返回当前集合的副本
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
	for id := range s {
		if count >= maxSliceLength {
			break
		}
		result[id] = struct{}{}
		count++
	}
	for id := range other {
		if count >= maxSliceLength {
			break
		}
		result[id] = struct{}{}
		count++
	}
	return result
}

// Intersect 返回两个集合的交集（不修改原集合，返回新集合）
func (s IDSet) Intersect(other IDSet) IDSet {
	// nil 检查：如果 other 为 nil，交集为空
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

// Difference 返回两个集合的差集（s中有但other中没有的）（不修改原集合，返回新集合）
func (s IDSet) Difference(other IDSet) IDSet {
	// nil 检查：如果 other 为 nil，返回当前集合的副本
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

// Clone 克隆集合，返回一个新的独立副本（不可变性）
func (s IDSet) Clone() IDSet {
	result := make(IDSet, len(s))
	for id := range s {
		result[id] = struct{}{}
	}
	return result
}

// Equal 检查两个集合是否相等
func (s IDSet) Equal(other IDSet) bool {
	if s == nil && other == nil {
		return true
	}
	if s == nil || other == nil {
		return false
	}

	if len(s) != len(other) {
		return false
	}
	for id := range s {
		if !other.Contains(id) {
			return false
		}
	}
	return true
}

// ValidateAll 验证集合中所有ID的有效性
func (s IDSet) ValidateAll() error {
	for id := range s {
		if err := id.Validate(); err != nil {
			return fmt.Errorf("invalid ID %d: %w", id, err)
		}
	}
	return nil
}
