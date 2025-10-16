package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
)

// Extras 扩展字段类型，用于存储动态的键值对数据
//
// 设计说明：
// - 基于 map[string]any，支持存储任意类型的值
// - 适用于需要灵活扩展字段的场景，避免频繁修改数据库表结构
// - 支持数据库 JSON 存储和 Go 结构体序列化
//
// 性能特点：
// - 内存占用：基础结构 48 字节 + 动态数据
// - 查询性能：O(1) 哈希查找
// - 类型转换：支持自动类型转换，带边界检查
//
// 线程安全：
// - map 类型非线程安全，多协程并发读写需要外部加锁
// - 建议在业务层使用 sync.RWMutex 保护
//
// 注意事项：
// - 避免存储过大的数据（影响数据库性能）
// - 类型转换失败时返回零值和 false
// - nil 和空 map 在序列化时行为一致
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

// Set 设置键值对
//
// 使用场景：添加或更新扩展字段
// 时间复杂度：O(1) 平均情况，O(n) 最坏情况（哈希冲突）
// 内存分配：首次插入 key 时分配
//
// 示例：
//
//	extras.Set("username", "alice")
//	extras.Set("age", 25)
//	extras.Set("tags", []string{"vip", "active"})
//
// 注意：value 为 nil 时仍会存储 nil 值，使用 SetOrDel 可自动删除
func (e Extras) Set(key string, value any) {
	e[key] = value
}

// SetOrDel 设置键值对，如果值为 nil 则删除键
//
// 使用场景：条件性设置，自动清理 nil 值
// 时间复杂度：O(1)
// 内存分配：0（删除时）或分配（新增时）
//
// 示例：
//
//	extras.SetOrDel("optional", getValue())  // 值为 nil 时自动删除
//
// 业务优势：避免存储无意义的 nil 值，节省存储空间
func (e Extras) SetOrDel(key string, value any) {
	if value == nil {
		delete(e, key)
		return
	}
	e[key] = value
}

// Delete 删除指定的键
//
// 使用场景：移除不需要的字段
// 时间复杂度：O(1)
// 内存分配：0
//
// 示例：
//
//	extras.Delete("temporary_field")
//
// 注意：删除不存在的键不会报错
func (e Extras) Delete(key string) {
	delete(e, key)
}

// Get 获取指定键的值（原始类型）
//
// 使用场景：获取任意类型的值，需要手动类型断言
// 时间复杂度：O(1)
// 内存分配：0
//
// 返回值：
// - value: 键对应的值
// - exists: 键是否存在
//
// 示例：
//
//	if value, ok := extras.Get("custom_data"); ok {
//	    // 需要手动类型断言
//	    if str, ok := value.(string); ok {
//	        // 使用 str
//	    }
//	}
func (e Extras) Get(key string) (any, bool) {
	value, exists := e[key]
	return value, exists
}

// GetString 获取字符串类型的值
//
// 使用场景：获取文本类型的扩展字段
// 时间复杂度：O(1)
// 内存分配：0
//
// 返回值：
// - 成功：返回字符串值和 true
// - 失败：返回空字符串和 false（键不存在或类型不匹配）
//
// 注意：仅支持精确的 string 类型，不会自动转换其他类型
func (e Extras) GetString(key string) (string, bool) {
	value, exists := e[key]
	if !exists {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}

// GetStringSlice 获取字符串切片类型的值
//
// 使用场景：获取标签、分类等列表数据
// 时间复杂度：O(1) + O(n) 转换（n 为切片长度）
// 内存分配：转换 []any 时需要分配新切片
//
// 支持的类型：
// - []string: 直接返回
// - []any: 尝试转换每个元素为 string
//
// 返回值：转换失败时返回 nil 和 false
func (e Extras) GetStringSlice(key string) ([]string, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []string:
			return val, true
		case []any:
			// 预分配切片，避免多次扩容
			strs := make([]string, 0, len(val))
			for _, item := range val {
				if str, ok := item.(string); ok {
					strs = append(strs, str)
				} else {
					// 任意元素转换失败，返回失败
					return nil, false
				}
			}
			return strs, true
		}
	}
	return nil, false
}

// GetInt 获取整数类型的值
//
// 使用场景：获取计数、ID 等整数字段
// 时间复杂度：O(1)
// 内存分配：0
//
// 支持的类型转换：
// - int, int8, int16, int32, int64 → int
// - uint, uint8, uint16, uint32, uint64 → int（需在 int 范围内）
// - float32, float64 → int（仅支持整数值）
//
// 返回值：
// - 成功：返回转换后的 int 值和 true
// - 失败：返回 0 和 false（类型不支持或溢出）
//
// 注意：浮点数转换时会检查是否为整数值，如 1.5 会失败
func (e Extras) GetInt(key string) (int, bool) {
	value, exists := e[key]
	if !exists {
		return 0, false
	}
	return convertToInt(value)
}

// GetIntSlice 获取整数切片类型的值
//
// 使用场景：获取 ID 列表、数值数组等
// 时间复杂度：O(1) + O(n)
// 内存分配：转换时需要分配新切片
func (e Extras) GetIntSlice(key string) ([]int, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []int:
			return val, true
		case []any:
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

// GetInt8 获取 int8 类型值
//
// 范围：-128 到 127
// 使用场景：小范围整数，如状态码、等级等
func (e Extras) GetInt8(key string) (int8, bool) {
	if v, ok := e[key]; ok {
		return convertToInt8(v)
	}
	return 0, false
}

// GetInt8Slice 获取 int8 切片类型值
func (e Extras) GetInt8Slice(key string) ([]int8, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []int8:
			return val, true
		case []any:
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

// GetInt16 获取 int16 类型值
//
// 范围：-32768 到 32767
func (e Extras) GetInt16(key string) (int16, bool) {
	if v, ok := e[key]; ok {
		return convertToInt16(v)
	}
	return 0, false
}

// GetInt16Slice 获取 int16 切片类型值
func (e Extras) GetInt16Slice(key string) ([]int16, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []int16:
			return val, true
		case []any:
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

// GetInt32 获取 int32 类型值
//
// 范围：-2147483648 到 2147483647
func (e Extras) GetInt32(key string) (int32, bool) {
	if v, ok := e[key]; ok {
		return convertToInt32(v)
	}
	return 0, false
}

// GetInt32Slice 获取 int32 切片类型值
func (e Extras) GetInt32Slice(key string) ([]int32, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []int32:
			return val, true
		case []any:
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

// GetInt64 获取 int64 类型的值
//
// 范围：-9223372036854775808 到 9223372036854775807
// 使用场景：大整数、时间戳等
func (e Extras) GetInt64(key string) (int64, bool) {
	value, exists := e[key]
	if !exists {
		return 0, false
	}
	return convertToInt64(value)
}

// GetInt64Slice 获取 int64 切片类型值
func (e Extras) GetInt64Slice(key string) ([]int64, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []int64:
			return val, true
		case []any:
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

// GetUint 获取无符号整数类型值
//
// 范围：0 到 math.MaxUint64（64 位系统）
func (e Extras) GetUint(key string) (uint, bool) {
	if v, ok := e[key]; ok {
		return convertToUint(v)
	}
	return 0, false
}

// GetUintSlice 获取 uint 切片类型值
func (e Extras) GetUintSlice(key string) ([]uint, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []uint:
			return val, true
		case []any:
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

// GetUint8 获取 uint8 类型值
//
// 范围：0 到 255
func (e Extras) GetUint8(key string) (uint8, bool) {
	if v, ok := e[key]; ok {
		return convertToUint8(v)
	}
	return 0, false
}

// GetUint8Slice 获取 uint8 切片类型值
func (e Extras) GetUint8Slice(key string) ([]uint8, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []uint8:
			return val, true
		case []any:
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

// GetUint16 获取 uint16 类型值
//
// 范围：0 到 65535
func (e Extras) GetUint16(key string) (uint16, bool) {
	if v, ok := e[key]; ok {
		return convertToUint16(v)
	}
	return 0, false
}

// GetUint16Slice 获取 uint16 切片类型值
func (e Extras) GetUint16Slice(key string) ([]uint16, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []uint16:
			return val, true
		case []any:
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

// GetUint32 获取 uint32 类型值
//
// 范围：0 到 4294967295
func (e Extras) GetUint32(key string) (uint32, bool) {
	if v, ok := e[key]; ok {
		return convertToUint32(v)
	}
	return 0, false
}

// GetUint32Slice 获取 uint32 切片类型值
func (e Extras) GetUint32Slice(key string) ([]uint32, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []uint32:
			return val, true
		case []any:
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

// GetUint64 获取 uint64 类型值
//
// 范围：0 到 math.MaxUint64
func (e Extras) GetUint64(key string) (uint64, bool) {
	if v, ok := e[key]; ok {
		return convertToUint64Typed(v)
	}
	return 0, false
}

// GetUint64Slice 获取 uint64 切片类型值
func (e Extras) GetUint64Slice(key string) ([]uint64, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []uint64:
			return val, true
		case []any:
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

// GetFloat32 获取 float32 类型值
//
// 精度：约 6-7 位十进制数字
// 范围：±1.18e-38 到 ±3.4e38
func (e Extras) GetFloat32(key string) (float32, bool) {
	if v, ok := e[key]; ok {
		return convertToFloat32(v)
	}
	return 0, false
}

// GetFloat32Slice 获取 float32 切片类型值
func (e Extras) GetFloat32Slice(key string) ([]float32, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []float32:
			return val, true
		case []any:
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

// GetFloat64 获取浮点数类型的值
//
// 精度：约 15-16 位十进制数字
// 范围：±2.23e-308 到 ±1.8e308
// 使用场景：价格、坐标、科学计算等
func (e Extras) GetFloat64(key string) (float64, bool) {
	value, exists := e[key]
	if !exists {
		return 0, false
	}
	return convertToFloat64(value)
}

// GetFloat64Slice 获取 float64 切片类型值
func (e Extras) GetFloat64Slice(key string) ([]float64, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []float64:
			return val, true
		case []any:
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

// GetBool 获取布尔类型的值
//
// 使用场景：开关、标记等布尔字段
// 注意：仅支持精确的 bool 类型，不会将 0/1 或 "true"/"false" 自动转换
func (e Extras) GetBool(key string) (bool, bool) {
	value, exists := e[key]
	if !exists {
		return false, false
	}
	b, ok := value.(bool)
	return b, ok
}

// GetBoolSlice 获取布尔切片类型值
func (e Extras) GetBoolSlice(key string) ([]bool, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []bool:
			return val, true
		case []any:
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

// GetSlice 获取任意类型切片
//
// 使用场景：获取混合类型的数组数据
func (e Extras) GetSlice(key string) ([]any, bool) {
	value, exists := e[key]
	if !exists {
		return nil, false
	}
	slice, ok := value.([]any)
	return slice, ok
}

// GetMap 获取嵌套的 map 类型值
//
// 使用场景：获取复杂的嵌套对象
func (e Extras) GetMap(key string) (map[string]any, bool) {
	value, exists := e[key]
	if !exists {
		return nil, false
	}
	m, ok := value.(map[string]any)
	return m, ok
}

// GetExtras 获取嵌套的 Extras 类型值
//
// 使用场景：获取子级扩展字段
// 支持自动转换 map[string]any 为 Extras
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

// GetExtrasSlice 获取 Extras 切片类型值
func (e Extras) GetExtrasSlice(key string) ([]Extras, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []Extras:
			return val, true
		case []any:
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

// GetBytes 获取字节切片类型值
//
// 使用场景：获取二进制数据、序列化内容等
// 支持 string 自动转换为 []byte
func (e Extras) GetBytes(key string) ([]byte, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []byte:
			return val, true
		case string:
			return []byte(val), true
		}
	}
	return nil, false
}

// Value 实现 driver.Valuer 接口，用于数据库写入
//
// 数据库存储：将 Extras 序列化为 JSON 字符串
// 时间复杂度：O(n)，n 为数据量
// 内存分配：JSON 序列化需要分配内存
//
// 特殊处理：
// - nil 或空 map 都会返回 NULL
// - 序列化失败返回错误
//
// 注意：存储大对象时会影响数据库性能，建议限制大小
func (e Extras) Value() (driver.Value, error) {
	// 空值优化：避免存储无意义的空 JSON
	if e == nil || len(e) == 0 {
		return nil, nil
	}

	// JSON 序列化
	data, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Extras to JSON: %w", err)
	}
	return data, nil
}

// Scan 实现 sql.Scanner 接口，用于数据库读取
//
// 支持的数据库类型：
// - []byte: JSON 字节流
// - string: JSON 字符串
// - nil: NULL 值
//
// 时间复杂度：O(n)
// 内存分配：反序列化时分配新 map
//
// 错误处理：
// - NULL 值映射为 nil
// - 空字符串/字节流映射为 nil
// - JSON 格式错误返回解析错误
func (e *Extras) Scan(value any) error {
	// 处理 NULL 值
	if value == nil {
		*e = nil
		return nil
	}

	// 类型转换：统一转为 []byte 处理
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("failed to scan Extras: unsupported type %T, expected []byte or string", value)
	}

	// 空值优化
	if len(bytes) == 0 {
		*e = nil
		return nil
	}

	// JSON 反序列化
	result := make(Extras)
	if err := json.Unmarshal(bytes, &result); err != nil {
		return fmt.Errorf("failed to unmarshal Extras from JSON: %w", err)
	}

	*e = result
	return nil
}

// MarshalJSON 实现 json.Marshaler 接口
//
// JSON 格式：标准的 JSON 对象
// nil 值会序列化为 null
func (e Extras) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}
	return json.Marshal(map[string]any(e))
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
//
// 支持的格式：
// - null: 映射为 nil
// - {}: 映射为空 map
// - {...}: 标准 JSON 对象
func (e *Extras) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
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

// Has 检查键是否存在
//
// 使用场景：判断字段是否设置（区分未设置和设置为 nil）
// 时间复杂度：O(1)
func (e Extras) Has(key string) bool {
	_, exists := e[key]
	return exists
}

// Keys 返回所有的键
//
// 使用场景：遍历所有字段名
// 时间复杂度：O(n)
// 内存分配：分配新切片
//
// 注意：返回的键顺序是随机的（map 无序）
func (e Extras) Keys() []string {
	keys := make([]string, 0, len(e))
	for k := range e {
		keys = append(keys, k)
	}
	return keys
}

// Len 返回键值对的数量
//
// 时间复杂度：O(1)
func (e Extras) Len() int {
	return len(e)
}

// IsEmpty 检查是否为空
//
// 使用场景：判断是否有任何扩展字段
// 时间复杂度：O(1)
func (e Extras) IsEmpty() bool {
	return len(e) == 0
}

// Clear 清空所有键值对
//
// 使用场景：重置扩展字段
// 时间复杂度：O(n)
//
// 注意：不会释放底层内存，如需释放请重新赋值为 nil
func (e Extras) Clear() {
	for k := range e {
		delete(e, k)
	}
}

// Clone 创建一个浅拷贝
//
// 使用场景：复制扩展字段，避免意外修改
// 时间复杂度：O(n)
// 内存分配：分配新 map + n 个键值对
//
// 注意：这是浅拷贝，嵌套的引用类型（slice、map）仍指向原对象
func (e Extras) Clone() Extras {
	if e == nil {
		return nil
	}
	clone := make(Extras, len(e))
	for k, v := range e {
		clone[k] = v
	}
	return clone
}

// Merge 合并另一个 Extras
//
// 使用场景：组合多个扩展字段
// 时间复杂度：O(n)，n 为 other 的大小
//
// 注意：相同的键会被 other 的值覆盖
func (e Extras) Merge(other Extras) {
	for k, v := range other {
		e[k] = v
	}
}

// ==================== 内部辅助函数 ====================
// 以下函数用于类型转换，带完整的边界检查和溢出保护

// convertToUint64 辅助函数：转换无符号整数到 uint64
// 内部函数，不执行边界检查（调用方保证类型正确）
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

// toInt64 辅助函数：转换有符号整数到 int64
// 内部函数，不执行边界检查
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

// convertToInt64 尝试将任意数值类型转换为 int64
//
// 支持的类型：int 系列、uint 系列、float 系列
// 边界检查：
// - uint64 超过 int64 最大值会失败
// - float 必须是整数值且在范围内
func convertToInt64(v any) (int64, bool) {
	switch val := v.(type) {
	case int64:
		return val, true
	case int, int8, int16, int32:
		return toInt64(val), true
	case uint, uint8, uint16, uint32:
		return int64(convertToUint64(val)), true
	case uint64:
		// 溢出检查：uint64 可能超过 int64 最大值
		if val <= math.MaxInt64 {
			return int64(val), true
		}
	case float32:
		// 边界检查：必须在 int64 范围内且为整数值
		if val >= float32(math.MinInt64) && val <= float32(math.MaxInt64) && val == float32(int64(val)) {
			return int64(val), true
		}
	case float64:
		// 边界检查：必须在 int64 范围内且为整数值
		if val >= float64(math.MinInt64) && val <= float64(math.MaxInt64) && val == float64(int64(val)) {
			return int64(val), true
		}
	}
	return 0, false
}

// convertToInt 尝试将任意数值类型转换为 int
//
// 边界检查：考虑不同平台的 int 大小（32 位或 64 位）
func convertToInt(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int8:
		return int(val), true
	case int16:
		return int(val), true
	case int32:
		return int(val), true
	case int64:
		// 平台相关：int 在 32 位系统是 32 位，64 位系统是 64 位
		if val >= int64(math.MinInt) && val <= int64(math.MaxInt) {
			return int(val), true
		}
	case uint:
		if uint64(val) <= uint64(math.MaxInt) {
			return int(val), true
		}
	case uint8:
		return int(val), true
	case uint16:
		return int(val), true
	case uint32:
		if uint64(val) <= uint64(math.MaxInt) {
			return int(val), true
		}
	case uint64:
		if val <= uint64(math.MaxInt) {
			return int(val), true
		}
	case float32:
		if val >= float32(math.MinInt) && val <= float32(math.MaxInt) && val == float32(int(val)) {
			return int(val), true
		}
	case float64:
		if val >= float64(math.MinInt) && val <= float64(math.MaxInt) && val == float64(int(val)) {
			return int(val), true
		}
	}
	return 0, false
}

// convertToInt8 尝试将任意数值类型转换为 int8
// 范围：-128 到 127
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
		if uVal := convertToUint64(val); uVal <= math.MaxInt8 {
			return int8(uVal), true
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

// convertToInt16 尝试将任意数值类型转换为 int16
// 范围：-32768 到 32767
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
		if uVal := convertToUint64(val); uVal <= math.MaxInt16 {
			return int16(uVal), true
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

// convertToInt32 尝试将任意数值类型转换为 int32
// 范围：-2147483648 到 2147483647
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
		if uVal := convertToUint64(val); uVal <= math.MaxInt32 {
			return int32(uVal), true
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

// convertToUint64Typed 尝试将任意数值类型转换为 uint64
// 边界检查：负数会失败
func convertToUint64Typed(v any) (uint64, bool) {
	switch val := v.(type) {
	case uint64:
		return val, true
	case uint, uint8, uint16, uint32:
		return convertToUint64(val), true
	case int, int8, int16, int32, int64:
		// 负数检查
		if iVal := toInt64(val); iVal >= 0 {
			return uint64(iVal), true
		}
	case float32:
		// 范围检查：必须非负且在 uint64 范围内
		if val >= 0 && val <= float32(math.MaxUint64) && val == float32(uint64(val)) {
			return uint64(val), true
		}
	case float64:
		// 精度检查：float64 精度限制可能导致大 uint64 值不精确
		if val >= 0 && val <= float64(math.MaxUint64) && val == float64(uint64(val)) {
			return uint64(val), true
		}
	}
	return 0, false
}

// convertToUint 尝试将任意数值类型转换为 uint
// 平台相关：uint 在 32 位系统是 32 位，64 位系统是 64 位
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
		if val <= uint64(math.MaxUint) {
			return uint(val), true
		}
	case int, int8, int16, int32, int64:
		if iVal := toInt64(val); iVal >= 0 && iVal <= int64(math.MaxInt64) {
			return uint(iVal), true
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

// convertToUint8 尝试将任意数值类型转换为 uint8
// 范围：0 到 255
func convertToUint8(v any) (uint8, bool) {
	switch val := v.(type) {
	case uint8:
		return val, true
	case uint, uint16, uint32, uint64:
		if uVal := convertToUint64(val); uVal <= math.MaxUint8 {
			return uint8(uVal), true
		}
	case int, int8, int16, int32, int64:
		if iVal := toInt64(val); iVal >= 0 && iVal <= math.MaxUint8 {
			return uint8(iVal), true
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

// convertToUint16 尝试将任意数值类型转换为 uint16
// 范围：0 到 65535
func convertToUint16(v any) (uint16, bool) {
	switch val := v.(type) {
	case uint16:
		return val, true
	case uint8:
		return uint16(val), true
	case uint, uint32, uint64:
		if uVal := convertToUint64(val); uVal <= math.MaxUint16 {
			return uint16(uVal), true
		}
	case int, int8, int16, int32, int64:
		if iVal := toInt64(val); iVal >= 0 && iVal <= math.MaxUint16 {
			return uint16(iVal), true
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

// convertToUint32 尝试将任意数值类型转换为 uint32
// 范围：0 到 4294967295
func convertToUint32(v any) (uint32, bool) {
	switch val := v.(type) {
	case uint32:
		return val, true
	case uint8:
		return uint32(val), true
	case uint16:
		return uint32(val), true
	case uint, uint64:
		if uVal := convertToUint64(val); uVal <= math.MaxUint32 {
			return uint32(uVal), true
		}
	case int, int8, int16, int32, int64:
		if iVal := toInt64(val); iVal >= 0 && iVal <= math.MaxUint32 {
			return uint32(iVal), true
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

// convertToFloat32 尝试将任意数值类型转换为 float32
// 精度：约 6-7 位十进制数字
// 注意：大整数转换可能损失精度
func convertToFloat32(v any) (float32, bool) {
	switch val := v.(type) {
	case float32:
		return val, true
	case float64:
		// 溢出检查
		if val >= -math.MaxFloat32 && val <= math.MaxFloat32 {
			return float32(val), true
		}
	case int, int8, int16, int32:
		return float32(toInt64(val)), true
	case int64:
		// 大整数可能损失精度，但仍然转换
		return float32(val), true
	case uint, uint8, uint16, uint32:
		return float32(convertToUint64(val)), true
	case uint64:
		// 大整数可能损失精度
		return float32(val), true
	}
	return 0, false
}

// convertToFloat64 尝试将任意数值类型转换为 float64
// 精度：约 15-16 位十进制数字
func convertToFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int, int8, int16, int32, int64:
		return float64(toInt64(val)), true
	case uint, uint8, uint16, uint32, uint64:
		return float64(convertToUint64(val)), true
	}
	return 0, false
}
