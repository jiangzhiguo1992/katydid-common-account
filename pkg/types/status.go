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

	// MaxStatus 最大合法状态值（int64 最大正数：9223372036854775807）
	// 使用 1<<63 - 1 避免 1<<62 的溢出问题
	MaxStatus Status = 1<<63 - 1
)

// ============================================================================
// 状态修改方法
// ============================================================================

// Set 设置为新状态
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
	if flag != 0 {
		*s |= flag
	}
}

// AddMultiple 批量设置多个状态位
//
// 使用场景：一次性添加多个状态
// 性能优化：预先合并所有标志，进行单次 OR 运算
func (s *Status) AddMultiple(flags ...Status) {
	if len(flags) == 0 {
		return
	}

	var combined Status
	for _, flag := range flags {
		if flag != 0 {
			combined |= flag
		}
	}

	if combined != 0 {
		*s |= combined
	}
}

// Del 移除指定的状态位
//
// 使用场景：移除特定状态，保留其他状态
// 时间复杂度：O(1)
// 内存分配：0
func (s *Status) Del(flag Status) {
	if flag != 0 {
		*s &^= flag
	}
}

// DelMultiple 批量取消多个状态位
//
// 使用场景：一次性移除多个状态
// 性能优化：预先合并所有标志，进行单次 AND NOT 运算
func (s *Status) DelMultiple(flags ...Status) {
	if len(flags) == 0 {
		return
	}

	var combined Status
	for _, flag := range flags {
		if flag != 0 {
			combined |= flag
		}
	}

	if combined != 0 {
		*s &^= combined
	}
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
	if len(flags) == 0 {
		*s = StatusNone
		return
	}

	var combined Status
	for _, flag := range flags {
		if flag != 0 {
			combined |= flag
		}
	}
	*s &= combined
}

// Toggle 切换指定的状态位
//
// 使用场景：开关式状态切换，有则删除，无则添加
// 时间复杂度：O(1)
func (s *Status) Toggle(flag Status) {
	if flag != 0 {
		*s ^= flag
	}
}

// ToggleMultiple 批量切换指定的状态位
func (s *Status) ToggleMultiple(flags ...Status) {
	if len(flags) == 0 {
		return
	}

	var combined Status
	for _, flag := range flags {
		if flag != 0 {
			combined |= flag
		}
	}

	if combined != 0 {
		*s ^= combined
	}
}

// ============================================================================
// 状态查询方法
// ============================================================================

// Has 检查是否包含指定的状态位（精确匹配）
//
// 使用场景：检查是否同时包含所有指定的状态位
// 时间复杂度：O(1)
// 注意：Has(0) 永远返回 false，因为 StatusNone 表示无状态
func (s Status) Has(flag Status) bool {
	if flag == 0 {
		return false
	}
	return s&flag == flag
}

// HasAny 检查是否包含任意一个指定的状态位（或运算）
//
// 使用场景：检查是否包含多个候选状态中的至少一个
// 性能优化：使用预定义的状态组合常量效率更高
// 注意：零值标志会被自动过滤
func (s Status) HasAny(flags ...Status) bool {
	if len(flags) == 0 {
		return false
	}

	// 合并所有非零标志
	var combined Status
	for _, flag := range flags {
		if flag != 0 {
			combined |= flag
		}
	}

	// 如果所有标志都是零，返回 false
	if combined == 0 {
		return false
	}

	return s&combined != 0
}

// HasAll 检查是否包含所有指定的状态位（与运算）
//
// 使用场景：检查是否同时满足多个状态条件
// 性能优化：使用预定义的状态组合常量效率更高
// 注意：零值标志会被自动过滤，空参数列表返回 true
func (s Status) HasAll(flags ...Status) bool {
	if len(flags) == 0 {
		return true
	}

	// 合并所有非零标志
	var combined Status
	for _, flag := range flags {
		if flag != 0 {
			combined |= flag
		}
	}

	// 如果所有标志都是零，返回 true（空集是任意集合的子集）
	if combined == 0 {
		return true
	}

	return s&combined == combined
}

// ActiveFlags 获取所有已设置的状态位
//
// 使用场景：获取当前所有激活的状态标志
// 性能优化：预分配切片容量，避免多次扩容
func (s Status) ActiveFlags() []Status {
	if s == 0 {
		return nil
	}

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
// 辅助方法
// ============================================================================

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

// String 实现 fmt.Stringer 接口，用于调试和日志输出
//
// 返回格式：Status(数值) 或详细的状态位信息
// 使用场景：日志记录、错误消息、调试输出
func (s Status) String() string {
	if s == StatusNone {
		return "Status(0)"
	}

	// 检查是否为预定义的常量
	switch s {
	case StatusSysDeleted:
		return "StatusSysDeleted(1)"
	case StatusAdmDeleted:
		return "StatusAdmDeleted(2)"
	case StatusUserDeleted:
		return "StatusUserDeleted(4)"
	case StatusSysDisabled:
		return "StatusSysDisabled(8)"
	case StatusAdmDisabled:
		return "StatusAdmDisabled(16)"
	case StatusUserDisabled:
		return "StatusUserDisabled(32)"
	case StatusSysHidden:
		return "StatusSysHidden(64)"
	case StatusAdmHidden:
		return "StatusAdmHidden(128)"
	case StatusUserHidden:
		return "StatusUserHidden(256)"
	case StatusSysReview:
		return "StatusSysReview(512)"
	case StatusAdmReview:
		return "StatusAdmReview(1024)"
	case StatusUserReview:
		return "StatusUserReview(2048)"
	case StatusAllDeleted:
		return "StatusAllDeleted(7)"
	case StatusAllDisabled:
		return "StatusAllDisabled(56)"
	case StatusAllHidden:
		return "StatusAllHidden(448)"
	case StatusAllReview:
		return "StatusAllReview(3584)"
	}

	// 复合状态，返回数值和位计数
	return fmt.Sprintf("Status(%d)[%d bits]", s, s.BitCount())
}

// ============================================================================
// 数据库接口实现
// ============================================================================

// Value 实现 driver.Valuer 接口，用于数据库写入
//
// 数据库存储：将 Status 转换为 int64 存储
// 索引支持：int64 类型支持高效的 B-tree 索引
// 安全检查：写入前验证状态值的合法性
func (s Status) Value() (driver.Value, error) {
	// 验证状态值的合法性
	if s < 0 {
		return nil, fmt.Errorf("invalid Status value: negative number %d is not allowed", s)
	}
	if s > MaxStatus {
		return nil, fmt.Errorf("invalid Status value: %d exceeds maximum allowed value %d", s, MaxStatus)
	}
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
		if v > int64(MaxStatus) {
			return fmt.Errorf("invalid Status value: %d exceeds maximum allowed value %d (overflow)", v, MaxStatus)
		}
		*s = Status(v)

	case int:
		if v < 0 {
			return fmt.Errorf("invalid Status value: negative number %d is not allowed (sign bit conflict)", v)
		}
		if int64(v) > int64(MaxStatus) {
			return fmt.Errorf("invalid Status value: %d exceeds maximum allowed value %d (overflow)", v, MaxStatus)
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
		if num > int64(MaxStatus) {
			return fmt.Errorf("invalid Status value: %d exceeds maximum allowed value %d (overflow)", num, MaxStatus)
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
// 支持的 JSON 格式：数字类型或 null
// 示例：{"status": 5} 或 {"status": 0} 或 {"status": null}
func (s *Status) UnmarshalJSON(data []byte) error {
	if data == nil || len(data) == 0 {
		return fmt.Errorf("empty JSON data")
	}

	// 处理 JSON null（使用字节直接比较，避免内存分配）
	if len(data) == 4 && data[0] == 'n' && data[1] == 'u' && data[2] == 'l' && data[3] == 'l' {
		*s = StatusNone
		return nil
	}

	var num int64
	if err := json.Unmarshal(data, &num); err != nil {
		return fmt.Errorf("failed to unmarshal Status from JSON: invalid format, expected integer number: %w", err)
	}

	if num < 0 {
		return fmt.Errorf("failed to unmarshal Status from JSON: negative value %d is not allowed (sign bit conflict)", num)
	}

	if num > int64(MaxStatus) {
		return fmt.Errorf("failed to unmarshal Status from JSON: value %d exceeds maximum allowed value %d (overflow)", num, MaxStatus)
	}

	*s = Status(num)
	return nil
}
