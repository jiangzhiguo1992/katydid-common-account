package idgen

import (
	"database/sql/driver"
	"fmt"
	"strconv"
)

// ID 分布式ID类型，用于所有数据库模型
type ID int64

// NewID 生成新的分布式ID
// 返回:
//
//	ID: 生成的分布式ID
//	error: 生成失败时返回错误
func NewID() (ID, error) {
	id, err := NextID()
	if err != nil {
		return 0, err
	}
	return ID(id), nil
}

// MustNewID 生成新的分布式ID，出错时panic
// 注意: 该方法仅适用于初始化等无法处理错误的场景，一般情况下应使用NewID
// 返回:
//
//	ID: 生成的分布式ID
func MustNewID() ID {
	id, err := NewID()
	if err != nil {
		panic(err)
	}
	return id
}

// Int64 返回int64类型的ID值
func (id ID) Int64() int64 {
	return int64(id)
}

// String 返回字符串类型的ID值
func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// IsZero 判断ID是否为零值
func (id ID) IsZero() bool {
	return id == 0
}

// MarshalJSON 实现JSON序列化接口
// 将ID序列化为字符串格式，避免JavaScript中的大整数精度丢失问题
func (id ID) MarshalJSON() ([]byte, error) {
	// 预分配足够的容量，避免动态扩容
	// int64最大长度为20位（包括符号），加上两个引号共22字节
	buf := make([]byte, 0, 24)
	buf = append(buf, '"')
	buf = strconv.AppendInt(buf, int64(id), 10)
	buf = append(buf, '"')
	return buf, nil
}

// UnmarshalJSON 实现JSON反序列化接口
// 支持字符串和数字两种格式的输入
func (id *ID) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		*id = 0
		return nil
	}

	str := string(data)
	// 去掉引号
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	if str == "" || str == "0" {
		*id = 0
		return nil
	}

	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}
	*id = ID(val)
	return nil
}

// Scan 实现sql.Scanner接口，用于从数据库读取
// 支持int64、[]byte、string等多种数据库类型
func (id *ID) Scan(value interface{}) error {
	if value == nil {
		*id = 0
		return nil
	}

	switch v := value.(type) {
	case int64:
		*id = ID(v)
	case []byte:
		if len(v) == 0 {
			*id = 0
			return nil
		}
		val, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return fmt.Errorf("failed to scan ID: %w", err)
		}
		*id = ID(val)
	case string:
		if v == "" {
			*id = 0
			return nil
		}
		val, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to scan ID: %w", err)
		}
		*id = ID(val)
	default:
		return fmt.Errorf("cannot scan type %T into ID", value)
	}
	return nil
}

// Value 实现driver.Valuer接口，用于写入数据库
func (id ID) Value() (driver.Value, error) {
	return int64(id), nil
}

// ParseIDFromString 从字符串解析ID
// 参数:
//
//	s: 要解析的字符串
//
// 返回:
//
//	ID: 解析出的ID，空字符串返回0
//	error: 解析失败时返回错误
func ParseIDFromString(s string) (ID, error) {
	if s == "" {
		return 0, nil
	}
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid ID string: %w", err)
	}
	return ID(val), nil
}
