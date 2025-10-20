package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"katydid-common-account/pkg/idgen/core"
	"katydid-common-account/pkg/idgen/snowflake"
)

const (
	// JavaScript最大安全整数 (2^53 - 1)
	maxSafeInteger = 9007199254740991
	// 解析ID字符串的最大长度（防止DoS攻击）
	maxParseIDStringLength = 100
)

// ID ID类型定义（强类型，类型安全）
type ID int64

// NewID 创建新的ID
func NewID(val int64) ID {
	return ID(val)
}

// ParseID 从字符串解析ID（支持十进制、十六进制、二进制）
func ParseID(s string) (ID, error) {
	// 防止超长字符串导致的资源消耗
	if len(s) > maxParseIDStringLength {
		return 0, fmt.Errorf("ID string too long: max %d characters, got %d",
			maxParseIDStringLength, len(s))
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
	if len(data) == 0 {
		return fmt.Errorf("empty JSON data")
	}

	if len(data) > maxParseIDStringLength {
		return fmt.Errorf("JSON data too large")
	}

	// 尝试从字符串解析
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		val, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ID string format: %w", err)
		}
		if val < 0 {
			return fmt.Errorf("invalid ID: must be non-negative")
		}
		*id = ID(val)
		return nil
	}

	// 尝试从数字解析
	var num int64
	if err := json.Unmarshal(data, &num); err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}
	if num < 0 {
		return fmt.Errorf("invalid ID: must be non-negative")
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

// IsSafeForJavaScript 检查ID是否在JavaScript安全整数范围内
func (id ID) IsSafeForJavaScript() bool {
	return int64(id) >= 0 && int64(id) <= maxSafeInteger
}

// Validate 验证ID的有效性（包含Snowflake格式验证）
func (id ID) Validate() error {
	return snowflake.ValidateSnowflakeID(int64(id))
}

// Parse 解析ID信息（仅适用于Snowflake ID）
func (id ID) Parse() (*core.IDInfo, error) {
	if !id.IsValid() {
		return nil, fmt.Errorf("%w: got %d", core.ErrInvalidSnowflakeID, id)
	}

	parser := snowflake.NewParser()
	return parser.Parse(int64(id))
}

// ExtractTime 提取时间戳
func (id ID) ExtractTime() time.Time {
	parser := snowflake.NewParser()
	return parser.ExtractTimestamp(int64(id))
}

// ExtractDatacenterID 提取数据中心ID
func (id ID) ExtractDatacenterID() int64 {
	parser := snowflake.NewParser()
	return parser.ExtractDatacenterID(int64(id))
}

// ExtractWorkerID 提取工作机器ID
func (id ID) ExtractWorkerID() int64 {
	parser := snowflake.NewParser()
	return parser.ExtractWorkerID(int64(id))
}

// ExtractSequence 提取序列号
func (id ID) ExtractSequence() int64 {
	parser := snowflake.NewParser()
	return parser.ExtractSequence(int64(id))
}
