package idgen

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ID 封装的ID类型，提供便捷的转换和序列化方法
// 遵循单一职责原则：只负责ID的表示和转换
type ID int64

// NewID 创建新的ID实例
//
// 参数:
//
//	value: ID值
//
// 返回:
//
//	ID: ID实例
func NewID(value int64) ID {
	return ID(value)
}

// Int64 转换为int64类型
//
// 返回:
//
//	int64: ID的int64表示
func (id ID) Int64() int64 {
	return int64(id)
}

// String 转换为字符串
// 实现fmt.Stringer接口
//
// 返回:
//
//	string: ID的字符串表示
func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// Hex 转换为十六进制字符串
//
// 返回:
//
//	string: ID的十六进制表示（带0x前缀）
func (id ID) Hex() string {
	return fmt.Sprintf("0x%x", int64(id))
}

// Binary 转换为二进制字符串
//
// 返回:
//
//	string: ID的二进制表示（带0b前缀）
func (id ID) Binary() string {
	return fmt.Sprintf("0b%b", int64(id))
}

// MarshalJSON 实现JSON序列化
// 将ID序列化为字符串，避免JavaScript中大整数精度丢失问题
//
// 返回:
//
//	[]byte: JSON序列化结果
//	error: 序列化错误
func (id ID) MarshalJSON() ([]byte, error) {
	// 使用字符串避免JavaScript的Number精度问题
	return json.Marshal(id.String())
}

// UnmarshalJSON 实现JSON反序列化
// 支持从字符串或数字反序列化
//
// 参数:
//
//	data: JSON数据
//
// 返回:
//
//	error: 反序列化错误
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
//
// 返回:
//
//	bool: 如果ID为0则返回true
func (id ID) IsZero() bool {
	return id == 0
}

// IsValid 检查ID是否有效（大于0）
//
// 返回:
//
//	bool: 如果ID大于0则返回true
func (id ID) IsValid() bool {
	return id > 0
}

// Parse 解析ID信息（仅适用于Snowflake ID）
//
// 返回:
//
//	*IDInfo: ID信息结构体
//	error: 解析失败时返回错误
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

// ParseID 从字符串解析ID
// 支持十进制、十六进制（0x前缀）和二进制（0b前缀）格式
//
// 参数:
//
//	s: ID字符串
//
// 返回:
//
//	ID: 解析后的ID
//	error: 解析失败时返回错误
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

	return ID(val), nil
}

// IDSlice ID切片类型，提供批量操作方法
type IDSlice []ID

// Int64Slice 转换为int64切片
//
// 返回:
//
//	[]int64: int64切片
func (ids IDSlice) Int64Slice() []int64 {
	result := make([]int64, len(ids))
	for i, id := range ids {
		result[i] = id.Int64()
	}
	return result
}

// StringSlice 转换为字符串切片
//
// 返回:
//
//	[]string: 字符串切片
func (ids IDSlice) StringSlice() []string {
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = id.String()
	}
	return result
}

// Contains 检查是否包含指定ID
//
// 参数:
//
//	id: 要查找的ID
//
// 返回:
//
//	bool: 如果包含返回true
func (ids IDSlice) Contains(id ID) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// Deduplicate 去重
// 返回新的切片，不修改原切片（内存安全）
//
// 返回:
//
//	IDSlice: 去重后的ID切片
func (ids IDSlice) Deduplicate() IDSlice {
	if len(ids) == 0 {
		return ids
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
// 返回新的切片，不修改原切片
//
// 参数:
//
//	predicate: 过滤条件函数
//
// 返回:
//
//	IDSlice: 过滤后的ID切片
func (ids IDSlice) Filter(predicate func(ID) bool) IDSlice {
	result := make(IDSlice, 0, len(ids))
	for _, id := range ids {
		if predicate(id) {
			result = append(result, id)
		}
	}
	return result
}

// IDSet ID集合类型，提供集合操作
// 使用map实现，查找性能O(1)
type IDSet map[ID]struct{}

// NewIDSet 创建新的ID集合
//
// 参数:
//
//	ids: 初始ID列表（可选）
//
// 返回:
//
//	IDSet: ID集合实例
func NewIDSet(ids ...ID) IDSet {
	set := make(IDSet, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

// Add 添加ID到集合
//
// 参数:
//
//	id: 要添加的ID
func (s IDSet) Add(id ID) {
	s[id] = struct{}{}
}

// Remove 从集合中移除ID
//
// 参数:
//
//	id: 要移除的ID
func (s IDSet) Remove(id ID) {
	delete(s, id)
}

// Contains 检查集合是否包含指定ID
//
// 参数:
//
//	id: 要查找的ID
//
// 返回:
//
//	bool: 如果包含返回true
func (s IDSet) Contains(id ID) bool {
	_, exists := s[id]
	return exists
}

// Size 获取集合大小
//
// 返回:
//
//	int: 集合中ID的数量
func (s IDSet) Size() int {
	return len(s)
}

// ToSlice 转换为ID切片
//
// 返回:
//
//	IDSlice: ID切片
func (s IDSet) ToSlice() IDSlice {
	result := make(IDSlice, 0, len(s))
	for id := range s {
		result = append(result, id)
	}
	return result
}

// Union 返回两个集合的并集
// 不修改原集合，返回新集合（内存安全）
//
// 参数:
//
//	other: 另一个ID集合
//
// 返回:
//
//	IDSet: 并集结果
func (s IDSet) Union(other IDSet) IDSet {
	result := make(IDSet, len(s)+len(other))
	for id := range s {
		result[id] = struct{}{}
	}
	for id := range other {
		result[id] = struct{}{}
	}
	return result
}

// Intersect 返回两个集合的交集
// 不修改原集合，返回新集合
//
// 参数:
//
//	other: 另一个ID集合
//
// 返回:
//
//	IDSet: 交集结果
func (s IDSet) Intersect(other IDSet) IDSet {
	result := make(IDSet)
	for id := range s {
		if other.Contains(id) {
			result[id] = struct{}{}
		}
	}
	return result
}

// Difference 返回两个集合的差集（s中有但other中没有的）
// 不修改原集合，返回新集合
//
// 参数:
//
//	other: 另一个ID集合
//
// 返回:
//
//	IDSet: 差集结果
func (s IDSet) Difference(other IDSet) IDSet {
	result := make(IDSet)
	for id := range s {
		if !other.Contains(id) {
			result[id] = struct{}{}
		}
	}
	return result
}

// BatchIDGenerator 批量ID生成器
// 提供批量生成ID的便捷方法，减少锁竞争
type BatchIDGenerator struct {
	generator IDGenerator
}

// NewBatchIDGenerator 创建批量ID生成器
//
// 参数:
//
//	generator: 底层ID生成器
//
// 返回:
//
//	*BatchIDGenerator: 批量生成器实例
func NewBatchIDGenerator(generator IDGenerator) *BatchIDGenerator {
	return &BatchIDGenerator{
		generator: generator,
	}
}

// Generate 批量生成ID
//
// 参数:
//
//	count: 要生成的ID数量
//
// 返回:
//
//	[]int64: 生成的ID列表
//	error: 生成失败时返回错误
func (b *BatchIDGenerator) Generate(count int) ([]int64, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be positive, got %d", count)
	}

	ids := make([]int64, 0, count)
	for i := 0; i < count; i++ {
		id, err := b.generator.NextID()
		if err != nil {
			return ids, fmt.Errorf("failed to generate ID at index %d: %w", i, err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// GenerateIDs 批量生成ID的便捷函数
//
// 参数:
//
//	count: 要生成的ID数量
//
// 返回:
//
//	[]int64: 生成的ID列表
//	error: 生成失败时返回错误
func GenerateIDs(count int) ([]int64, error) {
	gen, err := GetDefaultGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to get default generator: %w", err)
	}

	batch := NewBatchIDGenerator(gen)
	return batch.Generate(count)
}
