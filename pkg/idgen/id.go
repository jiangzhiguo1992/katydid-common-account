package idgen

import (
	"database/sql/driver"
	"fmt"
	"strconv"
)

// ID 分布式ID类型，用于所有数据库模型
type ID int64

// NewID 生成新的分布式ID
func NewID() (ID, error) {
	id, err := NextID()
	if err != nil {
		return 0, err
	}
	return ID(id), nil
}

// MustNewID 生成新的分布式ID，出错时panic
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
func (id ID) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%d"`, id)), nil
}

// UnmarshalJSON 实现JSON反序列化接口
func (id *ID) UnmarshalJSON(data []byte) error {
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
func (id *ID) Scan(value interface{}) error {
	if value == nil {
		*id = 0
		return nil
	}

	switch v := value.(type) {
	case int64:
		*id = ID(v)
	case []byte:
		val, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return fmt.Errorf("failed to scan ID: %w", err)
		}
		*id = ID(val)
	case string:
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
