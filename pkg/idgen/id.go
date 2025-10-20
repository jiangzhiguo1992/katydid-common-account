package idgen

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ID 封装的ID类型，提供便捷的转换和序列化方法
type ID int64

// IDInfo ID解析后的信息结构体
type IDInfo struct {
	ID           int64     `json:"id"`            // 原始ID
	Timestamp    int64     `json:"timestamp"`     // 时间戳（毫秒）
	Time         time.Time `json:"time"`          // 时间对象
	DatacenterID int64     `json:"datacenter_id"` // 数据中心ID
	WorkerID     int64     `json:"worker_id"`     // 工作机器ID
	Sequence     int64     `json:"sequence"`      // 序列号
}

// NewID 创建新的ID实例
func NewID(value int64) ID {
	return ID(value)
}

// ParseID 从字符串解析ID（支持十进制、十六进制（0x前缀）和二进制（0b前缀）格式）
func ParseID(s string) (ID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty ID string")
	}

	var val int64
	var err error

	// 处理不同进制
	switch {
	case strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X"):
		// 十六进制
		val, err = strconv.ParseInt(s[2:], 16, 64)
	case strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B"):
		// 二进制
		val, err = strconv.ParseInt(s[2:], 2, 64)
	default:
		// 十进制
		val, err = strconv.ParseInt(s, 10, 64)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to parse ID: %w", err)
	}

	// 添加负数检查，确保ID为非负数
	if val < 0 {
		return 0, fmt.Errorf("invalid ID: must be non-negative, got %d", val)
	}

	return ID(val), nil
}

// Int64 转换为int64类型
func (id ID) Int64() int64 {
	return int64(id)
}

// String 转换为字符串（实现fmt.Stringer接口）
func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// Hex 转换为十六进制字符串（带0x前缀）
func (id ID) Hex() string {
	return fmt.Sprintf("0x%x", int64(id))
}

// Binary 转换为二进制字符串（带0b前缀）
func (id ID) Binary() string {
	return fmt.Sprintf("0b%b", int64(id))
}

// MarshalJSON 实现JSON序列化（将ID序列化为字符串，避免JavaScript中大整数精度丢失问题）
func (id ID) MarshalJSON() ([]byte, error) {
	// 使用字符串避免JavaScript的Number精度问题
	return json.Marshal(id.String())
}

// UnmarshalJSON 实现JSON反序列化（支持从字符串或数字反序列化）
func (id *ID) UnmarshalJSON(data []byte) error {
	// 尝试从字符串解析
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		val, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse ID from string: %w", err)
		}
		*id = ID(val)
		return nil
	}

	// 尝试从数字解析
	var num int64
	if err := json.Unmarshal(data, &num); err != nil {
		return fmt.Errorf("failed to parse ID from number: %w", err)
	}
	*id = ID(num)
	return nil
}

// IsZero 检查ID是否为零值
func (id ID) IsZero() bool {
	return id == 0
}

// IsValid 检查ID是否有效（大于0）
func (id ID) IsValid() bool {
	return id > 0
}

// Parse 解析ID信息（仅适用于Snowflake ID）
func (id ID) Parse() (*IDInfo, error) {
	if !id.IsValid() {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidSnowflakeID, id)
	}

	timestamp := (int64(id) >> TimestampShift) + Epoch
	datacenterID := (int64(id) >> DatacenterIDShift) & MaxDatacenterID
	workerID := (int64(id) >> WorkerIDShift) & MaxWorkerID
	sequence := int64(id) & MaxSequence

	return &IDInfo{
		ID:           int64(id),
		Timestamp:    timestamp,
		Time:         GetTimestamp(int64(id)),
		DatacenterID: datacenterID,
		WorkerID:     workerID,
		Sequence:     sequence,
	}, nil
}

// IDSlice ID切片类型，提供批量操作方法
type IDSlice []ID

// Int64Slice 转换为int64切片
func (ids IDSlice) Int64Slice() []int64 {
	result := make([]int64, len(ids))
	for i, id := range ids {
		result[i] = id.Int64()
	}
	return result
}

// StringSlice 转换为字符串切片
func (ids IDSlice) StringSlice() []string {
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = id.String()
	}
	return result
}

// Contains 检查是否包含指定ID
func (ids IDSlice) Contains(id ID) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// Deduplicate 去重（返回新的切片，不修改原切片（内存安全））
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

// Filter 过滤ID（返回新的切片，不修改原切片）
func (ids IDSlice) Filter(predicate func(ID) bool) IDSlice {
	if predicate == nil {
		// 如果predicate为nil，返回原切片的副本
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

// IDSet ID集合类型，提供集合操作（使用map实现，查找性能O(1)）
type IDSet map[ID]struct{}

// NewIDSet 创建新的ID集合
func NewIDSet(ids ...ID) IDSet {
	set := make(IDSet, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

// Add 添加ID到集合
func (s IDSet) Add(id ID) {
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

// Union 返回两个集合的并集（不修改原集合，返回新集合（内存安全））
func (s IDSet) Union(other IDSet) IDSet {
	// nil 检查：如果 other 为 nil，返回当前集合的副本
	if other == nil {
		result := make(IDSet, len(s))
		for id := range s {
			result[id] = struct{}{}
		}
		return result
	}

	result := make(IDSet, len(s)+len(other))
	for id := range s {
		result[id] = struct{}{}
	}
	for id := range other {
		result[id] = struct{}{}
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

// Clone 克隆集合，返回一个新的独立副本
func (s IDSet) Clone() IDSet {
	result := make(IDSet, len(s))
	for id := range s {
		result[id] = struct{}{}
	}
	return result
}

// Equal 检查两个集合是否相等
func (s IDSet) Equal(other IDSet) bool {
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
