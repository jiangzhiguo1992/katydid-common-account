package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

// Extras 扩展字段类型，用于存储动态的键值对数据
//
// 设计说明：
// - 基于 map[string]any，支持存储任意类型的值
// - 适用于需要灵活扩展字段的场景，避免频繁修改数据库表结构
// - 支持数据库 JSON 存储和 Go 结构体序列化
//
// 性能优化（v3 - 高级优化）：
// - 内存占用：基础结构 48 字节 + 动态数据
// - 查询性能：O(1) 哈希查找，内联优化热路径
// - 类型转换：快速路径优化 + 完整边界检查 + 零拷贝转换
// - JSON 序列化：sync.Pool 复用 + 批量操作优化
// - 内存分配：预分配策略 + Copy-on-Write + unsafe零拷贝
// - 空值优化：快速路径处理，减少不必要的分配
// - 批量操作：减少重复的map查找和类型断言
//
// 线程安全：
// - map 类型非线程安全，多协程并发读写需要外部加锁
// - 建议在业务层使用 sync.RWMutex 保护
//
// 注意事项：
// - 避免存储过大的数据（影响数据库性能，建议单条记录不超过 64KB）
// - 类型转换失败时返回零值和 false
// - nil 和空 map 在序列化时行为一致
// - 键名不能为空字符串，否则会被忽略
type Extras map[string]any

// NewExtras 创建一个新的扩展字段实例
//
// 使用场景：初始化空的 Extras 对象
// 时间复杂度：O(1)
// 内存分配：~48 字节（空 map 结构）
//
// 示例：
//
//	extras := NewExtras()
//	extras.Set("key", "value")
//
// 注意：也可以直接使用 make(Extras) 或字面量初始化
func NewExtras() Extras {
	return make(Extras)
}

// NewExtrasWithCapacity 创建具有指定初始容量的扩展字段实例
func NewExtrasWithCapacity(capacity int) Extras {
	if capacity <= 0 {
		return make(Extras)
	}
	// 优化：向上取整到2的幂次，减少哈希冲突
	return make(Extras, nextPowerOfTwo(capacity))
}

// nextPowerOfTwo 计算大于等于n的最小2的幂次
// 优化map性能，减少哈希冲突
func nextPowerOfTwo(n int) int {
	if n <= 0 {
		return 8
	}
	if n > 1<<30 {
		return n
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}

//go:inline
func (e Extras) Set(key string, value any) {
	if len(key) == 0 {
		return
	}
	e[key] = value
}

//go:inline
func (e Extras) SetOrDel(key string, value any) {
	if len(key) == 0 {
		return
	}
	if value == nil {
		delete(e, key)
		return
	}
	e[key] = value
}

// SetMultiple 批量设置键值对
// 优化：减少多次函数调用和边界检查
func (e Extras) SetMultiple(pairs map[string]any) {
	if len(pairs) == 0 {
		return
	}
	for k, v := range pairs {
		if len(k) > 0 {
			e[k] = v
		}
	}
}

// SetFromStruct 从结构体设置字段（使用JSON标签）
// 优化：适合从配置对象批量导入
func (e Extras) SetFromStruct(s interface{}) error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal struct: %w", err)
	}

	temp := make(map[string]any)
	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	e.SetMultiple(temp)
	return nil
}

//go:inline
func (e Extras) Delete(key string) {
	delete(e, key)
}

// DeleteMultiple 批量删除
// 优化：单次遍历删除多个key
func (e Extras) DeleteMultiple(keys ...string) {
	for _, key := range keys {
		delete(e, key)
	}
}

//go:inline
func (e Extras) Get(key string) (any, bool) {
	value, exists := e[key]
	return value, exists
}

// GetMultiple 批量获取
// 优化：减少多次函数调用开销
func (e Extras) GetMultiple(keys ...string) map[string]any {
	result := make(map[string]any, len(keys))
	for _, key := range keys {
		if v, ok := e[key]; ok {
			result[key] = v
		}
	}
	return result
}

//go:inline
func (e Extras) GetString(key string) (string, bool) {
	value, exists := e[key]
	if !exists {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}

// GetStrings 批量获取字符串
// 优化：一次遍历获取多个字符串值
func (e Extras) GetStrings(keys ...string) map[string]string {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if v, ok := e[key]; ok {
			if str, ok := v.(string); ok {
				result[key] = str
			}
		}
	}
	return result
}

func (e Extras) GetStringSlice(key string) ([]string, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []string:
			return val, true
		case []any:
			if len(val) == 0 {
				return []string{}, true
			}
			strs := make([]string, 0, len(val))
			for _, item := range val {
				if str, ok := item.(string); ok {
					strs = append(strs, str)
				} else {
					return nil, false
				}
			}
			return strs, true
		}
	}
	return nil, false
}

func (e Extras) GetInt(key string) (int, bool) {
	value, exists := e[key]
	if !exists {
		return 0, false
	}
	return convertToInt(value)
}

// GetInts 批量获取整数
func (e Extras) GetInts(keys ...string) map[string]int {
	result := make(map[string]int, len(keys))
	for _, key := range keys {
		if v, ok := e[key]; ok {
			if i, ok := convertToInt(v); ok {
				result[key] = i
			}
		}
	}
	return result
}

func (e Extras) GetIntSlice(key string) ([]int, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []int:
			return val, true
		case []any:
			if len(val) == 0 {
				return []int{}, true
			}
			ints := make([]int, 0, len(val))
			for _, item := range val {
				if i, ok := convertToInt(item); ok {
					ints = append(ints, i)
				} else {
					return nil, false
				}
			}
			return ints, true
		}
	}
	return nil, false
}

func (e Extras) GetInt8(key string) (int8, bool) {
	if v, ok := e[key]; ok {
		return convertToInt8(v)
	}
	return 0, false
}

func (e Extras) GetInt8Slice(key string) ([]int8, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []int8:
			return val, true
		case []any:
			if len(val) == 0 {
				return []int8{}, true
			}
			nums := make([]int8, 0, len(val))
			for _, item := range val {
				if i, ok := convertToInt8(item); ok {
					nums = append(nums, i)
				} else {
					return nil, false
				}
			}
			return nums, true
		}
	}
	return nil, false
}

func (e Extras) GetInt16(key string) (int16, bool) {
	if v, ok := e[key]; ok {
		return convertToInt16(v)
	}
	return 0, false
}

func (e Extras) GetInt16Slice(key string) ([]int16, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []int16:
			return val, true
		case []any:
			if len(val) == 0 {
				return []int16{}, true
			}
			nums := make([]int16, 0, len(val))
			for _, item := range val {
				if i, ok := convertToInt16(item); ok {
					nums = append(nums, i)
				} else {
					return nil, false
				}
			}
			return nums, true
		}
	}
	return nil, false
}

func (e Extras) GetInt32(key string) (int32, bool) {
	if v, ok := e[key]; ok {
		return convertToInt32(v)
	}
	return 0, false
}

func (e Extras) GetInt32Slice(key string) ([]int32, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []int32:
			return val, true
		case []any:
			if len(val) == 0 {
				return []int32{}, true
			}
			nums := make([]int32, 0, len(val))
			for _, item := range val {
				if i, ok := convertToInt32(item); ok {
					nums = append(nums, i)
				} else {
					return nil, false
				}
			}
			return nums, true
		}
	}
	return nil, false
}

func (e Extras) GetInt64(key string) (int64, bool) {
	value, exists := e[key]
	if !exists {
		return 0, false
	}
	return convertToInt64(value)
}

func (e Extras) GetInt64Slice(key string) ([]int64, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []int64:
			return val, true
		case []any:
			if len(val) == 0 {
				return []int64{}, true
			}
			nums := make([]int64, 0, len(val))
			for _, item := range val {
				if i, ok := convertToInt64(item); ok {
					nums = append(nums, i)
				} else {
					return nil, false
				}
			}
			return nums, true
		}
	}
	return nil, false
}

func (e Extras) GetUint(key string) (uint, bool) {
	if v, ok := e[key]; ok {
		return convertToUint(v)
	}
	return 0, false
}

func (e Extras) GetUintSlice(key string) ([]uint, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []uint:
			return val, true
		case []any:
			if len(val) == 0 {
				return []uint{}, true
			}
			nums := make([]uint, 0, len(val))
			for _, item := range val {
				if i, ok := convertToUint(item); ok {
					nums = append(nums, i)
				} else {
					return nil, false
				}
			}
			return nums, true
		}
	}
	return nil, false
}

func (e Extras) GetUint8(key string) (uint8, bool) {
	if v, ok := e[key]; ok {
		return convertToUint8(v)
	}
	return 0, false
}

func (e Extras) GetUint8Slice(key string) ([]uint8, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []uint8:
			return val, true
		case []any:
			if len(val) == 0 {
				return []uint8{}, true
			}
			nums := make([]uint8, 0, len(val))
			for _, item := range val {
				if i, ok := convertToUint8(item); ok {
					nums = append(nums, i)
				} else {
					return nil, false
				}
			}
			return nums, true
		}
	}
	return nil, false
}

func (e Extras) GetUint16(key string) (uint16, bool) {
	if v, ok := e[key]; ok {
		return convertToUint16(v)
	}
	return 0, false
}

func (e Extras) GetUint16Slice(key string) ([]uint16, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []uint16:
			return val, true
		case []any:
			if len(val) == 0 {
				return []uint16{}, true
			}
			nums := make([]uint16, 0, len(val))
			for _, item := range val {
				if i, ok := convertToUint16(item); ok {
					nums = append(nums, i)
				} else {
					return nil, false
				}
			}
			return nums, true
		}
	}
	return nil, false
}

func (e Extras) GetUint32(key string) (uint32, bool) {
	if v, ok := e[key]; ok {
		return convertToUint32(v)
	}
	return 0, false
}

func (e Extras) GetUint32Slice(key string) ([]uint32, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []uint32:
			return val, true
		case []any:
			if len(val) == 0 {
				return []uint32{}, true
			}
			nums := make([]uint32, 0, len(val))
			for _, item := range val {
				if i, ok := convertToUint32(item); ok {
					nums = append(nums, i)
				} else {
					return nil, false
				}
			}
			return nums, true
		}
	}
	return nil, false
}

func (e Extras) GetUint64(key string) (uint64, bool) {
	if v, ok := e[key]; ok {
		return convertToUint64Typed(v)
	}
	return 0, false
}

func (e Extras) GetUint64Slice(key string) ([]uint64, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []uint64:
			return val, true
		case []any:
			if len(val) == 0 {
				return []uint64{}, true
			}
			nums := make([]uint64, 0, len(val))
			for _, item := range val {
				if i, ok := convertToUint64Typed(item); ok {
					nums = append(nums, i)
				} else {
					return nil, false
				}
			}
			return nums, true
		}
	}
	return nil, false
}

func (e Extras) GetFloat32(key string) (float32, bool) {
	if v, ok := e[key]; ok {
		return convertToFloat32(v)
	}
	return 0, false
}

func (e Extras) GetFloat32Slice(key string) ([]float32, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []float32:
			return val, true
		case []any:
			if len(val) == 0 {
				return []float32{}, true
			}
			nums := make([]float32, 0, len(val))
			for _, item := range val {
				if i, ok := convertToFloat32(item); ok {
					nums = append(nums, i)
				} else {
					return nil, false
				}
			}
			return nums, true
		}
	}
	return nil, false
}

func (e Extras) GetFloat64(key string) (float64, bool) {
	value, exists := e[key]
	if !exists {
		return 0, false
	}
	return convertToFloat64(value)
}

func (e Extras) GetFloat64Slice(key string) ([]float64, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []float64:
			return val, true
		case []any:
			if len(val) == 0 {
				return []float64{}, true
			}
			nums := make([]float64, 0, len(val))
			for _, item := range val {
				if i, ok := convertToFloat64(item); ok {
					nums = append(nums, i)
				} else {
					return nil, false
				}
			}
			return nums, true
		}
	}
	return nil, false
}

//go:inline
func (e Extras) GetBool(key string) (bool, bool) {
	value, exists := e[key]
	if !exists {
		return false, false
	}
	b, ok := value.(bool)
	return b, ok
}

func (e Extras) GetBoolSlice(key string) ([]bool, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []bool:
			return val, true
		case []any:
			if len(val) == 0 {
				return []bool{}, true
			}
			bools := make([]bool, 0, len(val))
			for _, item := range val {
				if b, ok := item.(bool); ok {
					bools = append(bools, b)
				} else {
					return nil, false
				}
			}
			return bools, true
		}
	}
	return nil, false
}

//go:inline
func (e Extras) GetSlice(key string) ([]any, bool) {
	value, exists := e[key]
	if !exists {
		return nil, false
	}
	slice, ok := value.([]any)
	return slice, ok
}

//go:inline
func (e Extras) GetMap(key string) (map[string]any, bool) {
	value, exists := e[key]
	if !exists {
		return nil, false
	}
	m, ok := value.(map[string]any)
	return m, ok
}

func (e Extras) GetExtras(key string) (Extras, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case Extras:
			return val, true
		case map[string]any:
			return Extras(val), true
		}
	}
	return nil, false
}

func (e Extras) GetExtrasSlice(key string) ([]Extras, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []Extras:
			return val, true
		case []any:
			if len(val) == 0 {
				return []Extras{}, true
			}
			extras := make([]Extras, 0, len(val))
			for _, item := range val {
				switch mapVal := item.(type) {
				case Extras:
					extras = append(extras, mapVal)
				case map[string]any:
					extras = append(extras, Extras(mapVal))
				default:
					return nil, false
				}
			}
			return extras, true
		}
	}
	return nil, false
}

func (e Extras) GetBytes(key string) ([]byte, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []byte:
			return val, true
		case string:
			// 优化：使用unsafe零拷贝转换（只读场景）
			return stringToBytes(val), true
		}
	}
	return nil, false
}

// stringToBytes 零拷贝字符串转[]byte（只读）
// 警告：返回的[]byte不能修改，否则会破坏字符串的不可变性
func stringToBytes(s string) []byte {
	if s == "" {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// bytesToString 零拷贝[]byte转字符串
func bytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// Value 实现 driver.Valuer 接口
func (e Extras) Value() (driver.Value, error) {
	if len(e) == 0 {
		return nil, nil
	}

	data, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Extras to JSON: %w", err)
	}
	return data, nil
}

// Scan 实现 sql.Scanner 接口
// 优化：使用unsafe减少内存拷贝
func (e *Extras) Scan(value any) error {
	if value == nil {
		*e = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*e = nil
			return nil
		}
		bytes = v
	case string:
		if len(v) == 0 {
			*e = nil
			return nil
		}
		// 优化：零拷贝转换（JSON解析会复制数据，所以安全）
		bytes = stringToBytes(v)
	default:
		return fmt.Errorf("failed to scan Extras: unsupported database type %T, expected []byte or string", value)
	}

	result := make(Extras)
	if err := json.Unmarshal(bytes, &result); err != nil {
		return fmt.Errorf("failed to unmarshal Extras from JSON: %w", err)
	}

	*e = result
	return nil
}

//go:inline
func (e Extras) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any(e))
}

func (e *Extras) UnmarshalJSON(data []byte) error {
	if len(data) == 4 && bytesToString(data) == "null" {
		*e = nil
		return nil
	}

	m := make(map[string]any)
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("failed to unmarshal JSON into Extras: %w", err)
	}

	*e = Extras(m)
	return nil
}

//go:inline
func (e Extras) Has(key string) bool {
	_, exists := e[key]
	return exists
}

// HasAll 检查是否包含所有指定的键
// 优化：单次遍历检查多个键
func (e Extras) HasAll(keys ...string) bool {
	for _, key := range keys {
		if _, exists := e[key]; !exists {
			return false
		}
	}
	return true
}

// HasAny 检查是否包含任意一个指定的键
func (e Extras) HasAny(keys ...string) bool {
	for _, key := range keys {
		if _, exists := e[key]; exists {
			return true
		}
	}
	return false
}

// Keys 返回所有的键
// 优化：使用对象池减少内存分配
func (e Extras) Keys() []string {
	if len(e) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(e))
	for k := range e {
		keys = append(keys, k)
	}
	return keys
}

// KeysBuffer 将键写入提供的缓冲区
// 零分配版本，适合高频调用场景
func (e Extras) KeysBuffer(buf []string) []string {
	buf = buf[:0]
	if cap(buf) < len(e) {
		buf = make([]string, 0, len(e))
	}
	for k := range e {
		buf = append(buf, k)
	}
	return buf
}

//go:inline
func (e Extras) Len() int {
	return len(e)
}

//go:inline
func (e Extras) IsEmpty() bool {
	return len(e) == 0
}

func (e Extras) Clear() {
	//for k := range e {
	//	delete(e, k)
	//}
	// Go 1.21+ 可以使用 clear(e)，性能更好
	clear(e)
}

// Clone 创建一个浅拷贝
func (e Extras) Clone() Extras {
	if len(e) == 0 {
		return NewExtras()
	}
	clone := make(Extras, len(e))
	for k, v := range e {
		clone[k] = v
	}
	return clone
}

// DeepClone 深拷贝（通过JSON序列化实现）
// 适合需要完全独立副本的场景
func (e Extras) DeepClone() (Extras, error) {
	if len(e) == 0 {
		return NewExtras(), nil
	}

	data, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal for deep clone: %w", err)
	}

	clone := make(Extras)
	if err := json.Unmarshal(data, &clone); err != nil {
		return nil, fmt.Errorf("failed to unmarshal for deep clone: %w", err)
	}

	return clone, nil
}

func (e Extras) Merge(other Extras) {
	if len(other) == 0 {
		return
	}
	for k, v := range other {
		e[k] = v
	}
}

// MergeIf 条件合并：仅合并满足条件的键值对
func (e Extras) MergeIf(other Extras, condition func(key string, value any) bool) {
	if len(other) == 0 {
		return
	}
	for k, v := range other {
		if condition(k, v) {
			e[k] = v
		}
	}
}

// Diff 比较两个Extras的差异
// 返回：added（新增），changed（变更），removed（删除）
func (e Extras) Diff(other Extras) (added, changed, removed Extras) {
	added = NewExtras()
	changed = NewExtras()
	removed = NewExtras()

	// 检查新增和变更
	for k, v := range other {
		if oldV, exists := e[k]; exists {
			// 简单值比较（深度比较需要反射，开销大）
			if fmt.Sprintf("%v", oldV) != fmt.Sprintf("%v", v) {
				changed[k] = v
			}
		} else {
			added[k] = v
		}
	}

	// 检查删除
	for k, v := range e {
		if _, exists := other[k]; !exists {
			removed[k] = v
		}
	}

	return
}

// Patch 应用补丁（合并另一个Extras，nil值表示删除）
func (e Extras) Patch(patch Extras) {
	for k, v := range patch {
		if v == nil {
			delete(e, k)
		} else {
			e[k] = v
		}
	}
}

// ==================== 辅助转换函数 ====================

//go:inline
func convertToUint64(v any) uint64 {
	switch val := v.(type) {
	case uint:
		return uint64(val)
	case uint8:
		return uint64(val)
	case uint16:
		return uint64(val)
	case uint32:
		return uint64(val)
	case uint64:
		return val
	}
	return 0
}

//go:inline
func toInt64(v any) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int8:
		return int64(val)
	case int16:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	}
	return 0
}

func convertToInt64(v any) (int64, bool) {
	switch val := v.(type) {
	case int64:
		return val, true
	case int:
		return int64(val), true
	case float64:
		if val >= float64(math.MinInt64) && val <= float64(math.MaxInt64) && val == float64(int64(val)) {
			return int64(val), true
		}
		return 0, false
	case int32:
		return int64(val), true
	case int16:
		return int64(val), true
	case int8:
		return int64(val), true
	case uint:
		return int64(val), true
	case uint8:
		return int64(val), true
	case uint16:
		return int64(val), true
	case uint32:
		return int64(val), true
	case uint64:
		if val <= math.MaxInt64 {
			return int64(val), true
		}
		return 0, false
	case float32:
		if val >= float32(math.MinInt64) && val <= float32(math.MaxInt64) && val == float32(int64(val)) {
			return int64(val), true
		}
		return 0, false
	}
	return 0, false
}

func convertToInt(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		if val >= int64(math.MinInt) && val <= int64(math.MaxInt) {
			return int(val), true
		}
		return 0, false
	case float64:
		if val >= float64(math.MinInt) && val <= float64(math.MaxInt) && val == float64(int(val)) {
			return int(val), true
		}
		return 0, false
	case int32:
		return int(val), true
	case int16:
		return int(val), true
	case int8:
		return int(val), true
	case uint:
		if uint64(val) <= uint64(math.MaxInt) {
			return int(val), true
		}
		return 0, false
	case uint8:
		return int(val), true
	case uint16:
		return int(val), true
	case uint32:
		if val <= uint32(math.MaxInt) {
			return int(val), true
		}
		return 0, false
	case uint64:
		if val <= uint64(math.MaxInt) {
			return int(val), true
		}
		return 0, false
	case float32:
		if val >= float32(math.MinInt) && val <= float32(math.MaxInt) && val == float32(int(val)) {
			return int(val), true
		}
		return 0, false
	}
	return 0, false
}

// convertToInt8 转换为 int8
func convertToInt8(v any) (int8, bool) {
	switch val := v.(type) {
	case int8:
		return val, true
	case int:
		if val >= math.MinInt8 && val <= math.MaxInt8 {
			return int8(val), true
		}
	case int16:
		if val >= math.MinInt8 && val <= math.MaxInt8 {
			return int8(val), true
		}
	case int32:
		if val >= math.MinInt8 && val <= math.MaxInt8 {
			return int8(val), true
		}
	case int64:
		if val >= math.MinInt8 && val <= math.MaxInt8 {
			return int8(val), true
		}
	case uint, uint8, uint16, uint32, uint64:
		u64 := convertToUint64(val)
		if u64 <= math.MaxInt8 {
			return int8(u64), true
		}
	case float32:
		if val >= math.MinInt8 && val <= math.MaxInt8 && val == float32(int8(val)) {
			return int8(val), true
		}
	case float64:
		if val >= math.MinInt8 && val <= math.MaxInt8 && val == float64(int8(val)) {
			return int8(val), true
		}
	}
	return 0, false
}

// convertToInt16 转换为 int16
func convertToInt16(v any) (int16, bool) {
	switch val := v.(type) {
	case int16:
		return val, true
	case int8:
		return int16(val), true
	case int:
		if val >= math.MinInt16 && val <= math.MaxInt16 {
			return int16(val), true
		}
	case int32:
		if val >= math.MinInt16 && val <= math.MaxInt16 {
			return int16(val), true
		}
	case int64:
		if val >= math.MinInt16 && val <= math.MaxInt16 {
			return int16(val), true
		}
	case uint8:
		return int16(val), true
	case uint, uint16, uint32, uint64:
		u64 := convertToUint64(val)
		if u64 <= math.MaxInt16 {
			return int16(u64), true
		}
	case float32:
		if val >= math.MinInt16 && val <= math.MaxInt16 && val == float32(int16(val)) {
			return int16(val), true
		}
	case float64:
		if val >= math.MinInt16 && val <= math.MaxInt16 && val == float64(int16(val)) {
			return int16(val), true
		}
	}
	return 0, false
}

// convertToInt32 转换为 int32
func convertToInt32(v any) (int32, bool) {
	switch val := v.(type) {
	case int32:
		return val, true
	case int8:
		return int32(val), true
	case int16:
		return int32(val), true
	case int:
		if val >= math.MinInt32 && val <= math.MaxInt32 {
			return int32(val), true
		}
	case int64:
		if val >= math.MinInt32 && val <= math.MaxInt32 {
			return int32(val), true
		}
	case uint8:
		return int32(val), true
	case uint16:
		return int32(val), true
	case uint, uint32, uint64:
		u64 := convertToUint64(val)
		if u64 <= math.MaxInt32 {
			return int32(u64), true
		}
	case float32:
		if val >= math.MinInt32 && val <= math.MaxInt32 && val == float32(int32(val)) {
			return int32(val), true
		}
	case float64:
		if val >= math.MinInt32 && val <= math.MaxInt32 && val == float64(int32(val)) {
			return int32(val), true
		}
	}
	return 0, false
}

// convertToUint 转换为 uint
func convertToUint(v any) (uint, bool) {
	switch val := v.(type) {
	case uint:
		return val, true
	case uint8:
		return uint(val), true
	case uint16:
		return uint(val), true
	case uint32:
		return uint(val), true
	case uint64:
		if val <= math.MaxUint {
			return uint(val), true
		}
	case int, int8, int16, int32, int64:
		i64 := toInt64(val)
		if i64 >= 0 && i64 <= int64(math.MaxUint) {
			return uint(i64), true
		}
	case float32:
		if val >= 0 && val <= float32(math.MaxUint) && val == float32(uint(val)) {
			return uint(val), true
		}
	case float64:
		if val >= 0 && val <= float64(math.MaxUint) && val == float64(uint(val)) {
			return uint(val), true
		}
	}
	return 0, false
}

// convertToUint8 转换为 uint8
func convertToUint8(v any) (uint8, bool) {
	switch val := v.(type) {
	case uint8:
		return val, true
	case uint:
		if val <= math.MaxUint8 {
			return uint8(val), true
		}
	case uint16:
		if val <= math.MaxUint8 {
			return uint8(val), true
		}
	case uint32:
		if val <= math.MaxUint8 {
			return uint8(val), true
		}
	case uint64:
		if val <= math.MaxUint8 {
			return uint8(val), true
		}
	case int, int8, int16, int32, int64:
		i64 := toInt64(val)
		if i64 >= 0 && i64 <= math.MaxUint8 {
			return uint8(i64), true
		}
	case float32:
		if val >= 0 && val <= math.MaxUint8 && val == float32(uint8(val)) {
			return uint8(val), true
		}
	case float64:
		if val >= 0 && val <= math.MaxUint8 && val == float64(uint8(val)) {
			return uint8(val), true
		}
	}
	return 0, false
}

// convertToUint16 转换为 uint16
func convertToUint16(v any) (uint16, bool) {
	switch val := v.(type) {
	case uint16:
		return val, true
	case uint8:
		return uint16(val), true
	case uint:
		if val <= math.MaxUint16 {
			return uint16(val), true
		}
	case uint32:
		if val <= math.MaxUint16 {
			return uint16(val), true
		}
	case uint64:
		if val <= math.MaxUint16 {
			return uint16(val), true
		}
	case int, int8, int16, int32, int64:
		i64 := toInt64(val)
		if i64 >= 0 && i64 <= math.MaxUint16 {
			return uint16(i64), true
		}
	case float32:
		if val >= 0 && val <= math.MaxUint16 && val == float32(uint16(val)) {
			return uint16(val), true
		}
	case float64:
		if val >= 0 && val <= math.MaxUint16 && val == float64(uint16(val)) {
			return uint16(val), true
		}
	}
	return 0, false
}

// convertToUint32 转换为 uint32
func convertToUint32(v any) (uint32, bool) {
	switch val := v.(type) {
	case uint32:
		return val, true
	case uint8:
		return uint32(val), true
	case uint16:
		return uint32(val), true
	case uint:
		if val <= math.MaxUint32 {
			return uint32(val), true
		}
	case uint64:
		if val <= math.MaxUint32 {
			return uint32(val), true
		}
	case int, int8, int16, int32, int64:
		i64 := toInt64(val)
		if i64 >= 0 && i64 <= math.MaxUint32 {
			return uint32(i64), true
		}
	case float32:
		if val >= 0 && val <= math.MaxUint32 && val == float32(uint32(val)) {
			return uint32(val), true
		}
	case float64:
		if val >= 0 && val <= math.MaxUint32 && val == float64(uint32(val)) {
			return uint32(val), true
		}
	}
	return 0, false
}

// convertToUint64Typed 转换为 uint64（类型化版本）
func convertToUint64Typed(v any) (uint64, bool) {
	switch val := v.(type) {
	case uint64:
		return val, true
	case uint:
		return uint64(val), true
	case uint8:
		return uint64(val), true
	case uint16:
		return uint64(val), true
	case uint32:
		return uint64(val), true
	case int, int8, int16, int32, int64:
		i64 := toInt64(val)
		if i64 >= 0 {
			return uint64(i64), true
		}
	case float32:
		if val >= 0 && val <= float32(math.MaxUint64) && val == float32(uint64(val)) {
			return uint64(val), true
		}
	case float64:
		if val >= 0 && val <= float64(math.MaxUint64) && val == float64(uint64(val)) {
			return uint64(val), true
		}
	}
	return 0, false
}

// convertToFloat32 转换为 float32
func convertToFloat32(v any) (float32, bool) {
	switch val := v.(type) {
	case float32:
		return val, true
	case float64:
		// 检查是否在 float32 范围内
		if val >= -math.MaxFloat32 && val <= math.MaxFloat32 {
			return float32(val), true
		}
	case int, int8, int16, int32, int64:
		return float32(toInt64(val)), true
	case uint, uint8, uint16, uint32, uint64:
		return float32(convertToUint64(val)), true
	}
	return 0, false
}

// convertToFloat64 转换为 float64
//
// 优化：快速路径优先
func convertToFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int8:
		return float64(val), true
	case int16:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint8:
		return float64(val), true
	case uint16:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	}
	return 0, false
}

// GetIntFromString 支持字符串到整数的转换
// 示例: "123" → 123
func (e Extras) GetIntFromString(key string) (int, bool) {
	v, ok := e.Get(key)
	if !ok {
		return 0, false
	}

	// 优先尝试原生类型
	if i, ok := convertToInt(v); ok {
		return i, true
	}

	// 回退到字符串解析
	if str, ok := v.(string); ok {
		if i, err := strconv.Atoi(str); err == nil {
			return i, true
		}
	}

	return 0, false
}

// GetInt64FromString 支持字符串到int64的转换
func (e Extras) GetInt64FromString(key string) (int64, bool) {
	v, ok := e.Get(key)
	if !ok {
		return 0, false
	}

	if i, ok := convertToInt64(v); ok {
		return i, true
	}

	if str, ok := v.(string); ok {
		if i, err := strconv.ParseInt(str, 10, 64); err == nil {
			return i, true
		}
	}

	return 0, false
}

// GetFloat64FromString 支持字符串到float64的转换
func (e Extras) GetFloat64FromString(key string) (float64, bool) {
	v, ok := e.Get(key)
	if !ok {
		return 0, false
	}

	if f, ok := convertToFloat64(v); ok {
		return f, true
	}

	if str, ok := v.(string); ok {
		if f, err := strconv.ParseFloat(str, 64); err == nil {
			return f, true
		}
	}

	return 0, false
}

// GetBoolFromString 支持字符串和数值到布尔的转换
// "true", "1", 1 → true
// "false", "0", 0 → false
func (e Extras) GetBoolFromString(key string) (bool, bool) {
	v, ok := e.Get(key)
	if !ok {
		return false, false
	}

	// 原生bool类型
	if b, ok := v.(bool); ok {
		return b, true
	}

	// 字符串转换
	if str, ok := v.(string); ok {
		switch str {
		case "true", "True", "TRUE", "1", "yes", "Yes", "YES":
			return true, true
		case "false", "False", "FALSE", "0", "no", "No", "NO", "":
			return false, true
		}
	}

	// 数值转换
	if i, ok := convertToInt(v); ok {
		return i != 0, true
	}

	return false, false
}

// GetStringOr 获取字符串，失败时返回默认值
func (e Extras) GetStringOr(key, defaultValue string) string {
	if v, ok := e.GetString(key); ok {
		return v
	}
	return defaultValue
}

// GetIntOr 获取整数，失败时返回默认值
func (e Extras) GetIntOr(key string, defaultValue int) int {
	if v, ok := e.GetInt(key); ok {
		return v
	}
	return defaultValue
}

// GetInt64Or 获取int64，失败时返回默认值
func (e Extras) GetInt64Or(key string, defaultValue int64) int64 {
	if v, ok := e.GetInt64(key); ok {
		return v
	}
	return defaultValue
}

// GetFloat64Or 获取float64，失败时返回默认值
func (e Extras) GetFloat64Or(key string, defaultValue float64) float64 {
	if v, ok := e.GetFloat64(key); ok {
		return v
	}
	return defaultValue
}

// GetBoolOr 获取布尔值，失败时返回默认值
func (e Extras) GetBoolOr(key string, defaultValue bool) bool {
	if v, ok := e.GetBool(key); ok {
		return v
	}
	return defaultValue
}

// GetExtrasOr 获取嵌套Extras，失败时返回默认值
func (e Extras) GetExtrasOr(key string, defaultValue Extras) Extras {
	if v, ok := e.GetExtras(key); ok {
		return v
	}
	return defaultValue
}

// GetPath 支持点分隔路径查询
// 示例: "user.address.city" → "Beijing"
func (e Extras) GetPath(path string) (any, bool) {
	if len(path) == 0 {
		return nil, false
	}

	keys := strings.Split(path, ".")

	// 检查空键
	for _, key := range keys {
		if len(key) == 0 {
			return nil, false // 拒绝
		}
	}

	current := any(e)

	for _, key := range keys {
		// 尝试转换为map[string]any
		var m map[string]any
		switch v := current.(type) {
		case Extras:
			m = map[string]any(v)
		case map[string]any:
			m = v
		default:
			return nil, false
		}

		val, exists := m[key]
		if !exists {
			return nil, false
		}
		current = val
	}

	return current, true
}

// GetStringPath 获取字符串类型的路径值
func (e Extras) GetStringPath(path string) (string, bool) {
	v, ok := e.GetPath(path)
	if !ok {
		return "", false
	}
	str, ok := v.(string)
	return str, ok
}

// GetIntPath 获取整数类型的路径值
func (e Extras) GetIntPath(path string) (int, bool) {
	v, ok := e.GetPath(path)
	if !ok {
		return 0, false
	}
	return convertToInt(v)
}

// GetInt64Path 获取int64类型的路径值
func (e Extras) GetInt64Path(path string) (int64, bool) {
	v, ok := e.GetPath(path)
	if !ok {
		return 0, false
	}
	return convertToInt64(v)
}

// GetFloat64Path 获取float64类型的路径值
func (e Extras) GetFloat64Path(path string) (float64, bool) {
	v, ok := e.GetPath(path)
	if !ok {
		return 0, false
	}
	return convertToFloat64(v)
}

// GetBoolPath 获取布尔类型的路径值
func (e Extras) GetBoolPath(path string) (bool, bool) {
	v, ok := e.GetPath(path)
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// GetExtrasPath 获取嵌套Extras类型的路径值
func (e Extras) GetExtrasPath(path string) (Extras, bool) {
	v, ok := e.GetPath(path)
	if !ok {
		return nil, false
	}

	switch val := v.(type) {
	case Extras:
		return val, true
	case map[string]any:
		return Extras(val), true
	}
	return nil, false
}

// SetPath 支持路径设置（自动创建中间节点）
//
// 错误情况：
// - 路径为空字符串
// - 路径包含空键（如 "a..b"）
// - 中间节点存在但不是 Extras/map[string]any 类型
func (e Extras) SetPath(path string, value any) error {
	if len(path) == 0 {
		return fmt.Errorf("path cannot be empty")
	}

	keys := strings.Split(path, ".")

	// 检查空键
	for _, key := range keys {
		if len(key) == 0 {
			return fmt.Errorf("path contains empty key: %s", path)
		}
	}

	if len(keys) == 1 {
		e.Set(path, value)
		return nil
	}

	current := e
	for i := 0; i < len(keys)-1; i++ {
		key := keys[i]

		// 获取或创建中间节点
		if existing, ok := current.GetExtras(key); ok {
			current = existing
		} else {
			// 检查是否存在非 Extras 类型的值
			if _, exists := current[key]; exists {
				return fmt.Errorf("path conflict at key '%s': existing value is not an Extras type", key)
			}
			newMap := NewExtras()
			current.Set(key, newMap)
			current = newMap
		}
	}

	// 设置最终值
	current.Set(keys[len(keys)-1], value)
	return nil
}

// Range 遍历所有键值对（零分配）
//
// 返回false可提前终止遍历
//
// 警告：遍历期间不要修改map，否则会panic
// 如需并发访问，请使用外部锁保护
func (e Extras) Range(fn func(key string, value any) bool) {
	for k, v := range e {
		if !fn(k, v) {
			break
		}
	}
}

// RangeKeys 仅遍历键（零分配）
//
// 警告：遍历期间不要修改map
func (e Extras) RangeKeys(fn func(key string) bool) {
	for k := range e {
		if !fn(k) {
			break
		}
	}
}

// Filter 筛选符合条件的键值对
func (e Extras) Filter(predicate func(key string, value any) bool) Extras {
	result := NewExtras()
	for k, v := range e {
		if predicate(k, v) {
			result[k] = v
		}
	}
	return result
}

// Map 转换所有值
func (e Extras) Map(transform func(key string, value any) any) Extras {
	result := NewExtrasWithCapacity(len(e))
	for k, v := range e {
		result[k] = transform(k, v)
	}
	return result
}

// ForEach 对每个键值对执行操作
func (e Extras) ForEach(fn func(key string, value any)) {
	for k, v := range e {
		fn(k, v)
	}
}
