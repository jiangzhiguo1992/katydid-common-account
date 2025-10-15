package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Status 状态类型，使用位运算支持多状态叠加
type Status int64

// 预定义的常用状态位
const (
	StatusNone      Status = 0         // 无状态
	StatusEnabled   Status = 1 << iota // 启用状态
	StatusVisible                      // 可见状态
	StatusLocked                       // 锁定状态
	StatusDeleted                      // 删除状态
	StatusActive                       // 活跃状态
	StatusVerified                     // 已验证状态
	StatusPublished                    // 已发布状态
	StatusArchived                     // 已归档状态
	StatusFeatured                     // 特色/推荐状态
	StatusPinned                       // 置顶状态
	StatusHidden                       // 隐藏状态
	StatusSuspended                    // 暂停/冻结状态
	StatusPending                      // 待处理状态
	StatusApproved                     // 已批准状态
	StatusRejected                     // 已拒绝状态
	StatusDraft                        // 草稿状态
)

// 常用的状态组合
const (
	StatusNormal        Status = StatusEnabled | StatusVisible // 正常状态（启用+可见）
	StatusPublicActive  Status = StatusEnabled | StatusVisible | StatusActive | StatusPublished
	StatusPendingReview Status = StatusEnabled | StatusPending
	StatusSoftDeleted   Status = StatusDeleted | StatusHidden
)

// Set 设置指定的状态位
func (s *Status) Set(flag Status) {
	*s |= flag
}

// Unset 取消指定的状态位
func (s *Status) Unset(flag Status) {
	*s &^= flag
}

// Just 清除所有状态，仅保留指定的状态
func (s *Status) Just(flag Status) {
	*s &= flag
}

// Toggle 切换指定的状态位
func (s *Status) Toggle(flag Status) {
	*s ^= flag
}

// Has 检查是否包含指定的状态位
func (s Status) Has(flag Status) bool {
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

// IsEnabled 是否启用
func (s Status) IsEnabled() bool {
	return s.Has(StatusEnabled)
}

// IsVisible 是否可见
func (s Status) IsVisible() bool {
	return s.Has(StatusVisible)
}

// IsLocked 是否锁定
func (s Status) IsLocked() bool {
	return s.Has(StatusLocked)
}

// IsDeleted 是否已删除
func (s Status) IsDeleted() bool {
	return s.Has(StatusDeleted)
}

// IsActive 是否活跃
func (s Status) IsActive() bool {
	return s.Has(StatusActive)
}

// IsVerified 是否已验证
func (s Status) IsVerified() bool {
	return s.Has(StatusVerified)
}

// IsPublished 是否已发布
func (s Status) IsPublished() bool {
	return s.Has(StatusPublished)
}

// IsNormal 是否为正常状态（启用+可见）
func (s Status) IsNormal() bool {
	return s.Has(StatusEnabled) && s.Has(StatusVisible)
}

// String 返回状态的字符串表示
func (s Status) String() string {
	if s == StatusNone {
		return "none"
	}

	var flags []string
	statusMap := map[Status]string{
		StatusEnabled:   "enabled",
		StatusVisible:   "visible",
		StatusLocked:    "locked",
		StatusDeleted:   "deleted",
		StatusActive:    "active",
		StatusVerified:  "verified",
		StatusPublished: "published",
		StatusArchived:  "archived",
		StatusFeatured:  "featured",
		StatusPinned:    "pinned",
		StatusHidden:    "hidden",
		StatusSuspended: "suspended",
		StatusPending:   "pending",
		StatusApproved:  "approved",
		StatusRejected:  "rejected",
		StatusDraft:     "draft",
	}

	for flag, name := range statusMap {
		if s.Has(flag) {
			flags = append(flags, name)
		}
	}

	if len(flags) == 0 {
		return fmt.Sprintf("unknown(%d)", s)
	}

	result := flags[0]
	for i := 1; i < len(flags); i++ {
		result += "|" + flags[i]
	}
	return result
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
