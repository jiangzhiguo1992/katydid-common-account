package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Status 状态类型，使用位运算支持多状态叠加，最多支持63种状态，比uint64少一位
// TODO:GG 测试数据库里走不走索引
type Status int64

// 预定义的常用状态位
const (
	StatusNone Status = 0 // 无状态

	StatusSysDeleted  Status = 1 << iota // 删除状态（系统级别）
	StatusAdmDeleted                     // 删除状态（管理员级别）
	StatusUserDeleted                    // 删除状态（用户级别）

	StatusSysDisabled  // 禁用状态（系统级别）
	StatusAdmDisabled  // 禁用状态（管理员级别）
	StatusUserDisabled // 禁用状态（用户级别）

	StatusSysHidden  // 隐藏状态（系统级别）
	StatusAdmHidden  // 隐藏状态（管理员级别）
	StatusUserHidden // 隐藏状态（用户级别）

	StatusSysUnverified  // 未验证状态（系统级别）
	StatusAdmUnverified  // 未验证状态（管理员级别）
	StatusUserUnverified // 未验证状态（用户级别）

	StatusExpand50 // 预留扩展位，还剩50位(63-4*3-1)可用
)

// Set 设置指定的状态位
func (s *Status) Set(flag Status) {
	*s |= flag
}

// Unset 取消指定的状态位
func (s *Status) Unset(flag Status) {
	*s &^= flag
}

// Toggle 切换指定的状态位
func (s *Status) Toggle(flag Status) {
	*s ^= flag
}

// Merge 保留与指定状态位相同的部分，其他位清除
func (s *Status) Merge(flag Status) {
	*s &= flag
}

// Contain 检查是否包含指定的状态位
func (s Status) Contain(flag Status) bool {
	return s&flag == flag
}

// HasAny 检查是否包含任意一个指定的状态位
func (s Status) HasAny(flags ...Status) bool {
	for _, flag := range flags {
		if s&flag != 0 {
			return true
		}
	}
	return false
}

// HasAll 检查是否包含所有指定的状态位
func (s Status) HasAll(flags ...Status) bool {
	for _, flag := range flags {
		if s&flag != flag {
			return false
		}
	}
	return true
}

// Clear 清除所有状态位
func (s *Status) Clear() {
	*s = StatusNone
}

// Equal 检查状态是否完全匹配
func (s Status) Equal(status Status) bool {
	return s == status
}

// SetMultiple 批量设置多个状态位
func (s *Status) SetMultiple(flags ...Status) {
	for _, flag := range flags {
		s.Set(flag)
	}
}

// UnsetMultiple 批量取消多个状态位
func (s *Status) UnsetMultiple(flags ...Status) {
	for _, flag := range flags {
		s.Unset(flag)
	}
}

// IsDeleted 是否删除
func (s Status) IsDeleted() bool {
	return s.HasAny(StatusSysDeleted, StatusAdmDeleted, StatusUserDeleted)
}

// IsDisable 是否禁用
func (s Status) IsDisable() bool {
	return s.HasAny(StatusSysDisabled, StatusAdmDisabled, StatusUserDisabled)
}

// IsHidden 是否隐藏
func (s Status) IsHidden() bool {
	return s.HasAny(StatusSysHidden, StatusAdmHidden, StatusUserHidden)
}

// IsUnverified 是否未验证
func (s Status) IsUnverified() bool {
	return s.HasAny(StatusSysUnverified, StatusAdmUnverified, StatusUserUnverified)
}

// CanEnable 是否为启用状态
func (s Status) CanEnable() bool {
	return !s.IsDeleted() && !s.IsDisable()
}

// CanVisible 是否为可见状态
func (s Status) CanVisible() bool {
	return s.CanEnable() && !s.IsHidden()
}

// CanVerified 是否为已验证状态
func (s Status) CanVerified() bool {
	return s.CanVisible() && !s.IsUnverified()
}

// Value 实现 driver.Valuer 接口，用于数据库存储
func (s Status) Value() (driver.Value, error) {
	return int64(s), nil
}

// Scan 实现 sql.Scanner 接口，用于从数据库读取
func (s *Status) Scan(value interface{}) error {
	if value == nil {
		*s = StatusNone
		return nil
	}

	switch v := value.(type) {
	case int64:
		*s = Status(v)
	case int:
		*s = Status(v)
	case uint64:
		*s = Status(v)
	case []byte:
		var num int64
		if err := json.Unmarshal(v, &num); err != nil {
			return err
		}
		*s = Status(num)
	default:
		return fmt.Errorf("cannot scan type %T into Status", value)
	}
	return nil
}

// MarshalJSON 实现 json.Marshaler 接口
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(s))
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (s *Status) UnmarshalJSON(data []byte) error {
	var num int64
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	*s = Status(num)
	return nil
}
