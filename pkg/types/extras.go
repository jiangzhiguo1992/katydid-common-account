package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"math"
)

// Extras 扩展字段，用于存储键值对形式的额外数据
// 支持存储各种类型的值（string, number, bool, array, object等）
type Extras map[string]any

// NewExtras 创建一个新的扩展字段实例
func NewExtras() Extras {
	return make(Extras)
}

// Set 设置键值对
func (e Extras) Set(key string, value any) {
	e[key] = value
}

func (e Extras) SetOrDel(key string, value any) {
	if value == nil {
		delete(e, key)
		return
	}
	e[key] = value
}

// Delete 删除指定的键
func (e Extras) Delete(key string) {
	delete(e, key)
}

// Get 获取指定键的值
func (e Extras) Get(key string) (any, bool) {
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

// GetStringSlice 获取[]string类型值
func (e Extras) GetStringSlice(key string) ([]string, bool) {
	if v, ok := e[key]; ok {
		switch val := v.(type) {
		case []string:
			return val, true
		case []any:
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

// GetInt 获取整数类型的值
func (e Extras) GetInt(key string) (int, bool) {
	value, exists := e[key]
	if !exists {
		return 0, false
	}
	return convertToInt(value)
}

// GetIntSlice 获取[]int类型值
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

// GetInt8 获取int8类型值
func (e Extras) GetInt8(key string) (int8, bool) {
	if v, ok := e[key]; ok {
		return convertToInt8(v)
	}
	return 0, false
}

// GetInt8Slice 获取[]int8类型值
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

// GetInt16 获取int16类型值
func (e Extras) GetInt16(key string) (int16, bool) {
	if v, ok := e[key]; ok {
		return convertToInt16(v)
	}
	return 0, false
}

// GetInt16Slice 获取[]int16类型值
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

// GetInt32 获取int32类型值
func (e Extras) GetInt32(key string) (int32, bool) {
	if v, ok := e[key]; ok {
		return convertToInt32(v)
	}
	return 0, false
}

// GetInt32Slice 获取[]int32类型值
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

// GetInt64 获取int64类型的值
func (e Extras) GetInt64(key string) (int64, bool) {
	value, exists := e[key]
	if !exists {
		return 0, false
	}
	return convertToInt64(value)
}

// GetInt64Slice 获取[]int64类型值
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

// GetUint 获取uint类型值
func (e Extras) GetUint(key string) (uint, bool) {
	if v, ok := e[key]; ok {
		return convertToUint(v)
	}
	return 0, false
}

// GetUintSlice 获取[]uint类型值
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

// GetUint8 获取uint8类型值
func (e Extras) GetUint8(key string) (uint8, bool) {
	if v, ok := e[key]; ok {
		return convertToUint8(v)
	}
	return 0, false
}

// GetUint8Slice 获取[]uint8类型值
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

// GetUint16 获取uint16类型值
func (e Extras) GetUint16(key string) (uint16, bool) {
	if v, ok := e[key]; ok {
		return convertToUint16(v)
	}
	return 0, false
}

// GetUint16Slice 获取[]uint16类型值
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

// GetUint32 获取uint32类型值
func (e Extras) GetUint32(key string) (uint32, bool) {
	if v, ok := e[key]; ok {
		return convertToUint32(v)
	}
	return 0, false
}

// GetUint32Slice 获取[]uint32类型值
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

// GetUint64 获取uint64类型值
func (e Extras) GetUint64(key string) (uint64, bool) {
	if v, ok := e[key]; ok {
		return convertToUint64Typed(v)
	}
	return 0, false
}

// GetUint64Slice 获取[]uint64类型值
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

// convertToUint64 辅助函数：转换无符号整数到uint64
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

// toInt64 辅助函数：转换有符号整数到int64
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

// convertToInt64 尝试将任意数值类型转换为int64
func convertToInt64(v any) (int64, bool) {
	switch val := v.(type) {
	case int64:
		return val, true
	case int, int8, int16, int32:
		return toInt64(val), true
	case uint, uint8, uint16, uint32:
		return int64(convertToUint64(val)), true
	case uint64:
		if val <= math.MaxInt64 {
			return int64(val), true
		}
	case float32:
		if val >= float32(math.MinInt64) && val <= float32(math.MaxInt64) && val == float32(int64(val)) {
			return int64(val), true
		}
	case float64:
		if val >= float64(math.MinInt64) && val <= float64(math.MaxInt64) && val == float64(int64(val)) {
			return int64(val), true
		}
	}
	return 0, false
}

// convertToInt 尝试将任意数值类型转换为int
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

// convertToInt8 尝试将任意数值类型转换为int8
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

// convertToInt16 尝试将任意数值类型转换为int16
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

// convertToInt32 尝试将任意数值类型转换为int32
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

// convertToUint64Typed 尝试将任意数值类型转换为uint64
func convertToUint64Typed(v any) (uint64, bool) {
	switch val := v.(type) {
	case uint64:
		return val, true
	case uint, uint8, uint16, uint32:
		return convertToUint64(val), true
	case int, int8, int16, int32, int64:
		if iVal := toInt64(val); iVal >= 0 {
			return uint64(iVal), true
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

// convertToUint 尝试将任意数值类型转换为uint
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
		if iVal := toInt64(val); iVal >= 0 {
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

// convertToUint8 尝试将任意数值类型转换为uint8
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

// convertToUint16 尝试将任意数值类型转换为uint16
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

// convertToUint32 尝试将任意数值类型转换为uint32
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

// GetFloat32 获取float32类型值
func (e Extras) GetFloat32(key string) (float32, bool) {
	if v, ok := e[key]; ok {
		return convertToFloat32(v)
	}
	return 0, false
}

// GetFloat32Slice 获取[]float32类型值
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

// convertToFloat32 尝试将任意数值类型转换为float32
func convertToFloat32(v any) (float32, bool) {
	switch val := v.(type) {
	case float32:
		return val, true
	case float64:
		if val >= -math.MaxFloat32 && val <= math.MaxFloat32 {
			return float32(val), true
		}
	case int, int8, int16, int32:
		return float32(toInt64(val)), true
	case int64:
		return float32(val), true
	case uint, uint8, uint16, uint32:
		return float32(convertToUint64(val)), true
	case uint64:
		return float32(val), true
	}
	return 0, false
}

// GetFloat64 获取浮点数类型的值
func (e Extras) GetFloat64(key string) (float64, bool) {
	value, exists := e[key]
	if !exists {
		return 0, false
	}
	return convertToFloat64(value)
}

// GetFloat64Slice 获取[]float64类型值
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

// convertToFloat64 尝试将任意数值类型转换为float64
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

// GetBool 获取布尔类型的值
func (e Extras) GetBool(key string) (bool, bool) {
	value, exists := e[key]
	if !exists {
		return false, false
	}
	b, ok := value.(bool)
	return b, ok
}

// GetBoolSlice 获取[]bool类型值
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

// GetSlice 获取切片类型的值
func (e Extras) GetSlice(key string) ([]any, bool) {
	value, exists := e[key]
	if !exists {
		return nil, false
	}
	slice, ok := value.([]any)
	return slice, ok
}

// GetMap 获取map类型的值
func (e Extras) GetMap(key string) (map[string]any, bool) {
	value, exists := e[key]
	if !exists {
		return nil, false
	}
	m, ok := value.(map[string]any)
	return m, ok
}

// GetExtras 获取Extras类型值
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

// GetExtrasSlice 获取[]Extras类型值
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

// GetBytes 获取[]byte类型值，支持string自动转换
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
// 将Extras序列化为JSON存储到数据库
func (e Extras) Value() (driver.Value, error) {
	if e == nil || len(e) == 0 {
		return nil, nil
	}
	return json.Marshal(e)
}

// Scan 实现 sql.Scanner 接口，用于数据库读取
// 从数据库读取JSON并反序列化为Extras
func (e *Extras) Scan(value any) error {
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
	return json.Marshal(map[string]any(e))
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (e *Extras) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*e = nil
		return nil
	}

	m := make(map[string]any)
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	*e = Extras(m)
	return nil
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
