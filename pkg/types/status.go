package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Status 状态类型，使用位运算支持多状态叠加
//
// 设计说明：
// - 基于 int64，支持最多 63 种状态位（第 63 位用于符号位）
// - 使用位运算实现高效的状态管理，时间复杂度 O(1)
// - 支持多状态组合，适用于需要同时表达多种状态的场景
// - 值类型设计，天然线程安全（每次操作都在副本上进行）
//
// 性能特点：
// - 内存占用：固定 8 字节
// - 状态检查：单次位运算，无内存分配
// - JSON 序列化：直接转换为 int64，性能优于字符串
//
// 注意事项：
// - 避免使用负数作为状态值（会导致符号位冲突）
// - 自定义状态位应从 StatusExpand51 开始左移
// - 数据库索引：int64 类型支持高效索引查询
// - 所有修改方法都需要指针接收者才能生效
type Status int64

// 预定义的常用状态位
//
// 状态分层设计：
// - Sys (System): 系统级别，最高优先级，通常由系统自动管理
// - Adm (Admin): 管理员级别，中等优先级，由管理员手动操作
// - User: 用户级别，最低优先级，由用户自主控制
//
// 四类状态：
// 1. Deleted: 删除标记（软删除）
// 2. Disabled: 禁用标记（暂时不可用）
// 3. Hidden: 隐藏标记（不对外展示）
// 4. Review: 审核标记（需要审核）
const (
	StatusNone Status = 0 // 无状态（零值，表示所有状态位都未设置）

	// 删除状态组（位 0-2）
	StatusSysDeleted  Status = 1 << 0 // 系统删除：由系统自动标记删除，通常不可恢复
	StatusAdmDeleted  Status = 1 << 1 // 管理员删除：由管理员操作删除，可能支持恢复
	StatusUserDeleted Status = 1 << 2 // 用户删除：由用户主动删除，通常可恢复(回收箱)

	// 禁用状态组（位 3-5）
	StatusSysDisabled  Status = 1 << 3 // 系统禁用：系统检测到异常后自动禁用
	StatusAdmDisabled  Status = 1 << 4 // 管理员禁用：管理员手动禁用
	StatusUserDisabled Status = 1 << 5 // 用户禁用：用户主动禁用（如账号冻结）

	// 隐藏状态组（位 6-8）
	StatusSysHidden  Status = 1 << 6 // 系统隐藏：系统根据规则自动隐藏
	StatusAdmHidden  Status = 1 << 7 // 管理员隐藏：管理员手动隐藏内容
	StatusUserHidden Status = 1 << 8 // 用户隐藏：用户设置为私密/不公开

	// 审核/验证状态组（位 9-11）
	StatusSysReview  Status = 1 << 9  // 系统审核：等待系统自动审核
	StatusAdmReview  Status = 1 << 10 // 管理员审核：等待管理员审核
	StatusUserReview Status = 1 << 11 // 用户审核：等待用户完成验证（如邮箱验证）

	// 扩展位（位 12 开始），预留 51 位可用于业务自定义状态（63 - 12 = 51）
	StatusExpand51 Status = 1 << 12 // 扩展起始位，自定义状态应基于此值左移
)

// 预定义的状态组合常量（性能优化：避免重复位运算）
const (
	// StatusAllDeleted 所有删除状态的组合（系统删除 | 管理员删除 | 用户删除）
	StatusAllDeleted Status = StatusSysDeleted | StatusAdmDeleted | StatusUserDeleted

	// StatusAllDisabled 所有禁用状态的组合（系统禁用 | 管理员禁用 | 用户禁用）
	StatusAllDisabled Status = StatusSysDisabled | StatusAdmDisabled | StatusUserDisabled

	// StatusAllHidden 所有隐藏状态的组合（系统隐藏 | 管理员隐藏 | 用户隐藏）
	StatusAllHidden Status = StatusSysHidden | StatusAdmHidden | StatusUserHidden

	// StatusAllReview 所有审核状态的组合（系统审核 | 管理员审核 | 用户审核）
	StatusAllReview Status = StatusSysReview | StatusAdmReview | StatusUserReview
)

// 状态值边界常量（用于运行时检查）
const (
	// maxValidBit 最大有效位数（int64 有 63 位可用，第 63 位为符号位）
	maxValidBit = 62

	// MaxStatus 最大合法状态值（所有 63 位都为 1，但排除符号位）
	MaxStatus Status = (1 << maxValidBit) - 1
)

// ============================================================================
// 状态修改方法
// ============================================================================

// Set 替换为新状态
//
// 使用场景：完全重置状态为指定值，丢弃所有原有状态
// 警告：此操作会清除所有原有状态，请确认是否真的需要完全替换
func (s *Status) Set(flag Status) {
	*s = flag
}

// Clear 清除所有状态位
//
// 使用场景：重置状态，移除所有标记
func (s *Status) Clear() {
	*s = StatusNone
}

// Add 追加指定的状态位
//
// 使用场景：在现有状态基础上添加新状态，不影响已有状态
// 时间复杂度：O(1)
// 内存分配：0
//
// 注意：此方法会修改接收者本身，必须传入指针才能生效
func (s *Status) Add(flag Status) {
	*s |= flag
}

// AddMultiple 批量设置多个状态位
//
// 使用场景：一次性添加多个状态
// 性能优化：预先合并所有标志，进行单次 OR 运算
func (s *Status) AddMultiple(flags ...Status) {
	var combined Status
	for _, flag := range flags {
		combined |= flag
	}
	*s |= combined
}

// Del 移除指定的状态位
//
// 使用场景：移除特定状态，保留其他状态
// 时间复杂度：O(1)
// 内存分配：0
func (s *Status) Del(flag Status) {
	*s &^= flag
}

// DelMultiple 批量取消多个状态位
//
// 使用场景：一次性移除多个状态
// 性能优化：预先合并所有标志，进行单次 AND NOT 运算
func (s *Status) DelMultiple(flags ...Status) {
	var combined Status
	for _, flag := range flags {
		combined |= flag
	}
	*s &^= combined
}

// And 保留与指定状态位相同的部分，其他位清除
//
// 使用场景：过滤状态，只保留指定的状态位
// 警告：此操作会清除所有未在 flag 中指定的状态位
func (s *Status) And(flag Status) {
	*s &= flag
}

// AndMultiple 批量保留与指定状态位相同的部分，其他位清除
func (s *Status) AndMultiple(flags ...Status) {
	var combined Status
	for _, flag := range flags {
		combined |= flag
	}
	*s &= combined
}

// Toggle 切换指定的状态位
//
// 使用场景：开关式状态切换，有则删除，无则添加
// 时间复杂度：O(1)
func (s *Status) Toggle(flag Status) {
	*s ^= flag
}

// ToggleMultiple 批量切换指定的状态位
func (s *Status) ToggleMultiple(flags ...Status) {
	var combined Status
	for _, flag := range flags {
		combined |= flag
	}
	*s ^= combined
}

// ============================================================================
// 状态查询方法
// ============================================================================

// Equal 检查状态是否完全匹配（精确相等）
//
// 使用场景：判断两个状态是否完全一致
// 注意：与 == 运算符效果相同，但语义更清晰
func (s Status) Equal(status Status) bool {
	return s == status
}

// Has 检查是否包含指定的状态位（精确匹配）
//
// 使用场景：检查是否同时包含所有指定的状态位
// 时间复杂度：O(1)
func (s Status) Has(flag Status) bool {
	return s&flag == flag
}

// HasAny 检查是否包含任意一个指定的状态位（或运算）
//
// 使用场景：检查是否包含多个候选状态中的至少一个
// 性能优化：使用预定义的状态组合常量效率更高
func (s Status) HasAny(flags ...Status) bool {
	switch len(flags) {
	case 0:
		return false
	case 1:
		return s&flags[0] != 0
	case 2:
		return s&(flags[0]|flags[1]) != 0
	case 3:
		return s&(flags[0]|flags[1]|flags[2]) != 0
	default:
		var combined Status
		for _, flag := range flags {
			combined |= flag
		}
		return s&combined != 0
	}
}

// HasAll 检查是否包含所有指定的状态位（与运算）
//
// 使用场景：检查是否同时满足多个状态条件
// 性能优化：使用预定义的状态组合常量效率更高
func (s Status) HasAll(flags ...Status) bool {
	switch len(flags) {
	case 0:
		return true
	case 1:
		return s&flags[0] == flags[0]
	case 2:
		combined := flags[0] | flags[1]
		return s&combined == combined
	case 3:
		combined := flags[0] | flags[1] | flags[2]
		return s&combined == combined
	default:
		var combined Status
		for _, flag := range flags {
			combined |= flag
		}
		return s&combined == combined
	}
}

// ActiveFlags 获取所有已设置的状态位
func (s Status) ActiveFlags() []Status {
	var flags []Status

	for i := 0; i <= maxValidBit; i++ {
		flag := Status(1 << i)
		if s&flag != 0 {
			flags = append(flags, flag)
		}
	}

	return flags
}

// Diff 比较两个状态的差异
//
// 参数 other 是旧状态，s 是新状态
// 返回：新增的状态位和移除的状态位
func (s Status) Diff(other Status) (added Status, removed Status) {
	added = s &^ other   // s 中有但 other 中没有的（新增）
	removed = other &^ s // other 中有但 s 中没有的（移除）
	return
}

// BitCount 计算已设置的位数量（popcount）
//
// 使用：Brian Kernighan 算法，O(k) k=置位数量
func (s Status) BitCount() int {
	count := 0
	v := uint64(s)
	for v != 0 {
		count++
		v &= v - 1
	}
	return count
}

// ============================================================================
// 业务状态检查方法
// ============================================================================

// IsDeleted 检查是否被标记为删除（任意级别）
//
// 业务语义：被删除的内容通常不应该被访问或展示
func (s Status) IsDeleted() bool {
	return s&StatusAllDeleted != 0
}

// IsDisable 检查是否被禁用（任意级别）
//
// 业务语义：被禁用的内容暂时不可用，可能需要管理员或用户操作恢复
func (s Status) IsDisable() bool {
	return s&StatusAllDisabled != 0
}

// IsHidden 检查是否被隐藏（任意级别）
//
// 业务语义：被隐藏的内容不对外展示，但功能可能正常
func (s Status) IsHidden() bool {
	return s&StatusAllHidden != 0
}

// IsReview 检查是否审核（任意级别）
//
// 业务语义：审核的内容可能需要审核或用户完成验证流程
func (s Status) IsReview() bool {
	return s&StatusAllReview != 0
}

// CanEnable 检查是否为可启用状态（业务可用性检查）
//
// 业务规则：未被删除且未被禁用的内容才可以启用
func (s Status) CanEnable() bool {
	return !s.IsDeleted() && !s.IsDisable()
}

// CanVisible 检查是否为可见状态（业务可见性检查）
//
// 业务规则：可启用且未被隐藏的内容才可见
func (s Status) CanVisible() bool {
	return s.CanEnable() && !s.IsHidden()
}

// CanActive 检查是否为已验证状态（业务验证检查）
//
// 业务规则：可见且已通过验证的内容才算完全可用
func (s Status) CanActive() bool {
	return s.CanVisible() && !s.IsReview()
}

// ============================================================================
// 数据库接口实现
// ============================================================================

// Value 实现 driver.Valuer 接口，用于数据库写入
//
// 数据库存储：将 Status 转换为 int64 存储
// 索引支持：int64 类型支持高效的 B-tree 索引
func (s Status) Value() (driver.Value, error) {
	return int64(s), nil
}

// Scan 实现 sql.Scanner 接口，用于从数据库读取
//
// 支持的数据库类型：
// - int64: 标准整数类型
// - int: Go 原生整数类型
// - uint64: 无符号整数类型（需范围检查）
// - []byte: JSON 格式的数字
func (s *Status) Scan(value interface{}) error {
	if value == nil {
		*s = StatusNone
		return nil
	}

	switch v := value.(type) {
	case int64:
		if v < 0 {
			return fmt.Errorf("invalid Status value: negative number %d is not allowed (sign bit conflict)", v)
		}
		*s = Status(v)

	case int:
		if v < 0 {
			return fmt.Errorf("invalid Status value: negative number %d is not allowed (sign bit conflict)", v)
		}
		*s = Status(v)

	case uint64:
		if v > uint64(MaxStatus) {
			return fmt.Errorf("invalid Status value: %d exceeds maximum allowed value %d (overflow)", v, MaxStatus)
		}
		*s = Status(v)

	case []byte:
		var num int64
		if err := json.Unmarshal(v, &num); err != nil {
			return fmt.Errorf("failed to unmarshal Status from bytes: %w", err)
		}
		if num < 0 {
			return fmt.Errorf("invalid Status value: negative number %d is not allowed (sign bit conflict)", num)
		}
		*s = Status(num)

	default:
		return fmt.Errorf("cannot scan type %T into Status: unsupported database type, expected int64, int, uint64, or []byte", value)
	}

	return nil
}

// ============================================================================
// JSON 序列化接口实现 (JSON Serialization Interface Implementation)
// ============================================================================

// MarshalJSON 实现 json.Marshaler 接口，用于 JSON 序列化
//
// JSON 格式：直接序列化为数字（非字符串）
// 示例输出：{"status": 5} 而非 {"status": "5"}
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(s))
}

// UnmarshalJSON 实现 json.Unmarshaler 接口，用于 JSON 反序列化
//
// 支持的 JSON 格式：数字类型
// 示例：{"status": 5} 或 {"status": 0}
func (s *Status) UnmarshalJSON(data []byte) error {
	if data == nil || len(data) == 0 {
		return fmt.Errorf("empty JSON data")
	}
	var num int64
	if err := json.Unmarshal(data, &num); err != nil {
		return fmt.Errorf("failed to unmarshal Status from JSON: invalid format, expected integer number: %w", err)
	}

	if num < 0 {
		return fmt.Errorf("failed to unmarshal Status from JSON: negative value %d is not allowed (sign bit conflict)", num)
	}

	*s = Status(num)
	return nil
}
