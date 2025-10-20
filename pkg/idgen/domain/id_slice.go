package domain

import "fmt"

const (
	// maxSliceLength 最大切片长度
	// 说明：限制切片大小，防止内存耗尽
	// 用途：用于所有切片和集合的容量限制
	maxSliceLength = 1_000_000
)

// IDSlice ID切片类型
//
// 特性：
//   - 支持类型转换（int64切片、字符串切片）
//   - 支持集合操作（包含、去重、过滤）
//   - 支持批量验证
//   - 提供便捷的访问方法
type IDSlice []ID

// NewIDSlice 创建新的ID切片
// 说明：创建切片的副本，避免外部修改影响
func NewIDSlice(ids ...ID) IDSlice {
	if ids == nil {
		return IDSlice{}
	}
	// 长度限制：防止内存耗尽
	if len(ids) > maxSliceLength {
		ids = ids[:maxSliceLength]
	}
	// 创建副本：避免外部修改
	result := make(IDSlice, len(ids))
	copy(result, ids)
	return result
}

// Int64Slice 转换为int64切片
// 说明：创建新的int64切片，包含所有ID的int64值
func (ids IDSlice) Int64Slice() []int64 {
	result := make([]int64, len(ids))
	for i, id := range ids {
		result[i] = id.Int64()
	}
	return result
}

// StringSlice 转换为字符串切片
// 说明：创建新的字符串切片，包含所有ID的字符串表示
func (ids IDSlice) StringSlice() []string {
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = id.String()
	}
	return result
}

// Contains 检查是否包含指定ID
// 说明：线性查找，时间复杂度O(n)
func (ids IDSlice) Contains(id ID) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// Len 返回切片长度
func (ids IDSlice) Len() int {
	return len(ids)
}

// IsEmpty 检查切片是否为空
func (ids IDSlice) IsEmpty() bool {
	return len(ids) == 0
}

// First 获取第一个元素
func (ids IDSlice) First() (ID, bool) {
	if len(ids) == 0 {
		return 0, false
	}
	return ids[0], true
}

// Last 获取最后一个元素
func (ids IDSlice) Last() (ID, bool) {
	if len(ids) == 0 {
		return 0, false
	}
	return ids[len(ids)-1], true
}

// Deduplicate 去重
func (ids IDSlice) Deduplicate() IDSlice {
	if len(ids) == 0 {
		return IDSlice{} // 返回新的空切片而不是原切片引用
	}

	seen := make(map[ID]bool, len(ids))
	result := make(IDSlice, 0, len(ids))

	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}

	return result
}

// Filter 过滤ID
func (ids IDSlice) Filter(predicate func(ID) bool) IDSlice {
	if predicate == nil {
		// predicate为nil时，返回原切片的副本
		result := make(IDSlice, len(ids))
		copy(result, ids)
		return result
	}

	result := make(IDSlice, 0, len(ids))
	for _, id := range ids {
		if predicate(id) {
			result = append(result, id)
		}
	}
	return result
}

// ValidateAll 验证切片中所有ID的有效性
func (ids IDSlice) ValidateAll() error {
	for i, id := range ids {
		if err := id.Validate(); err != nil {
			return fmt.Errorf("invalid ID at index %d: %w", i, err)
		}
	}
	return nil
}
