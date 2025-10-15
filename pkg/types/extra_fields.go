package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// ExtraFields 扩展字段，用于存储键值对形式的额外数据
// 支持存储各种类型的值（string, number, bool, array, object等）
type ExtraFields map[string]interface{}

// NewExtraFields 创建一个新的扩展字段实例
func NewExtraFields() ExtraFields {
	return make(ExtraFields)
}

// Set 设置键值对
func (ef ExtraFields) Set(key string, value interface{}) {
	ef[key] = value
}

// Get 获取指定键的值
func (ef ExtraFields) Get(key string) (interface{}, bool) {
	value, exists := ef[key]
	return value, exists
}

// GetString 获取字符串类型的值
func (ef ExtraFields) GetString(key string) (string, bool) {
	value, exists := ef[key]
	if !exists {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}

// GetInt 获取整数类型的值
func (ef ExtraFields) GetInt(key string) (int, bool) {
	value, exists := ef[key]
	if !exists {
		return 0, false
	}

	// 处理JSON反序列化后的数字类型（通常是float64）
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// GetInt64 获取int64类型的值
func (ef ExtraFields) GetInt64(key string) (int64, bool) {
	value, exists := ef[key]
	if !exists {
		return 0, false
	}

	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

// GetFloat64 获取浮点数类型的值
func (ef ExtraFields) GetFloat64(key string) (float64, bool) {
	value, exists := ef[key]
	if !exists {
		return 0, false
	}

	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

// GetBool 获取布尔类型的值
func (ef ExtraFields) GetBool(key string) (bool, bool) {
	value, exists := ef[key]
	if !exists {
		return false, false
	}
	b, ok := value.(bool)
	return b, ok
}

// GetSlice 获取切片类型的值
func (ef ExtraFields) GetSlice(key string) ([]interface{}, bool) {
	value, exists := ef[key]
	if !exists {
		return nil, false
	}
	slice, ok := value.([]interface{})
	return slice, ok
}

// GetMap 获取map类型的值
func (ef ExtraFields) GetMap(key string) (map[string]interface{}, bool) {
	value, exists := ef[key]
	if !exists {
		return nil, false
	}
	m, ok := value.(map[string]interface{})
	return m, ok
}

// Delete 删除指定的键
func (ef ExtraFields) Delete(key string) {
	delete(ef, key)
}

// Has 检查是否存在指定的键
func (ef ExtraFields) Has(key string) bool {
	_, exists := ef[key]
	return exists
}

// Keys 返回所有的键
func (ef ExtraFields) Keys() []string {
	keys := make([]string, 0, len(ef))
	for k := range ef {
		keys = append(keys, k)
	}
	return keys
}

// Len 返回键值对的数量
func (ef ExtraFields) Len() int {
	return len(ef)
}

// IsEmpty 检查是否为空
func (ef ExtraFields) IsEmpty() bool {
	return len(ef) == 0
}

// Clear 清空所有键值对
func (ef ExtraFields) Clear() {
	for k := range ef {
		delete(ef, k)
	}
}

// Clone 创建一个副本
func (ef ExtraFields) Clone() ExtraFields {
	clone := make(ExtraFields, len(ef))
	for k, v := range ef {
		clone[k] = v
	}
	return clone
}

// Merge 合并另一个ExtraFields，相同的键会被覆盖
func (ef ExtraFields) Merge(other ExtraFields) {
	for k, v := range other {
		ef[k] = v
	}
}

// Value 实现 driver.Valuer 接口，用于数据库写入
// 将ExtraFields序列化为JSON存储到数据库
func (ef ExtraFields) Value() (driver.Value, error) {
	if ef == nil || len(ef) == 0 {
		return nil, nil
	}
	return json.Marshal(ef)
}

// Scan 实现 sql.Scanner 接口，用于数据库读取
// 从数据库读取JSON并反序列化为ExtraFields
func (ef *ExtraFields) Scan(value interface{}) error {
	if value == nil {
		*ef = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("failed to scan ExtraFields: unsupported type")
	}

	if len(bytes) == 0 {
		*ef = nil
		return nil
	}

	result := make(ExtraFields)
	err := json.Unmarshal(bytes, &result)
	if err != nil {
		return err
	}

	*ef = result
	return nil
}

// MarshalJSON 实现 json.Marshaler 接口
func (ef ExtraFields) MarshalJSON() ([]byte, error) {
	if ef == nil {
		return []byte("null"), nil
	}
	return json.Marshal(map[string]interface{}(ef))
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (ef *ExtraFields) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*ef = nil
		return nil
	}

	m := make(map[string]interface{})
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	*ef = ExtraFields(m)
	return nil
}
