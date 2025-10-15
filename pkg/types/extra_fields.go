package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// Extras 扩展字段，用于存储键值对形式的额外数据
// 支持存储各种类型的值（string, number, bool, array, object等）
type Extras map[string]interface{}

// NewExtras 创建一个新的扩展字段实例
func NewExtras() Extras {
	return make(Extras)
}

// Set 设置键值对
func (e Extras) Set(key string, value interface{}) {
	e[key] = value
}

// Get 获取指定键的值
func (e Extras) Get(key string) (interface{}, bool) {
	value, exists := e[key]
	return value, exists
}

// GetString 获取字符串类型的值
func (e Extras) GetString(key string) (string, bool) {
	value, exists := e[key]
	if !exists {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}

// GetInt 获取整数类型的值
func (e Extras) GetInt(key string) (int, bool) {
	value, exists := e[key]
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
func (e Extras) GetInt64(key string) (int64, bool) {
	value, exists := e[key]
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
func (e Extras) GetFloat64(key string) (float64, bool) {
	value, exists := e[key]
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
func (e Extras) GetBool(key string) (bool, bool) {
	value, exists := e[key]
	if !exists {
		return false, false
	}
	b, ok := value.(bool)
	return b, ok
}

// GetSlice 获取切片类型的值
func (e Extras) GetSlice(key string) ([]interface{}, bool) {
	value, exists := e[key]
	if !exists {
		return nil, false
	}
	slice, ok := value.([]interface{})
	return slice, ok
}

// GetMap 获取map类型的值
func (e Extras) GetMap(key string) (map[string]interface{}, bool) {
	value, exists := e[key]
	if !exists {
		return nil, false
	}
	m, ok := value.(map[string]interface{})
	return m, ok
}

// Delete 删除指定的键
func (e Extras) Delete(key string) {
	delete(e, key)
}

// Has 检查是否存在指定的键
func (e Extras) Has(key string) bool {
	_, exists := e[key]
	return exists
}

// Keys 返回所有的键
func (e Extras) Keys() []string {
	keys := make([]string, 0, len(e))
	for k := range e {
		keys = append(keys, k)
	}
	return keys
}

// Len 返回键值对的数量
func (e Extras) Len() int {
	return len(e)
}

// IsEmpty 检查是否为空
func (e Extras) IsEmpty() bool {
	return len(e) == 0
}

// Clear 清空所有键值对
func (e Extras) Clear() {
	for k := range e {
		delete(e, k)
	}
}

// Clone 创建一个副本
func (e Extras) Clone() Extras {
	clone := make(Extras, len(e))
	for k, v := range e {
		clone[k] = v
	}
	return clone
}

// Merge 合并另一个Extras，相同的键会被覆盖
func (e Extras) Merge(other Extras) {
	for k, v := range other {
		e[k] = v
	}
}

// Value 实现 driver.Valuer 接口，用于数据库写入
// 将Extras序列化为JSON存储到数据库
func (e Extras) Value() (driver.Value, error) {
	if e == nil || len(e) == 0 {
		return nil, nil
	}
	return json.Marshal(e)
}

// Scan 实现 sql.Scanner 接口，用于数据库读取
// 从数据库读取JSON并反序列化为Extras
func (e *Extras) Scan(value interface{}) error {
	if value == nil {
		*e = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("failed to scan Extras: unsupported type")
	}

	if len(bytes) == 0 {
		*e = nil
		return nil
	}

	result := make(Extras)
	err := json.Unmarshal(bytes, &result)
	if err != nil {
		return err
	}

	*e = result
	return nil
}

// MarshalJSON 实现 json.Marshaler 接口
func (e Extras) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}
	return json.Marshal(map[string]interface{}(e))
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (e *Extras) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*e = nil
		return nil
	}

	m := make(map[string]interface{})
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	*e = Extras(m)
	return nil
}
