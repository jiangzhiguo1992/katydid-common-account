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
// - 自定义状态位应从 StatusExpand51 开始
// - 数据库索引：int64 类型支持高效索引查询
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
// 4. Unverified: 未验证标记（需要验证）
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

	// 未验证状态组（位 9-11）
	StatusSysUnverified  Status = 1 << 9  // 系统未验证：等待系统自动验证
	StatusAdmUnverified  Status = 1 << 10 // 管理员未验证：等待管理员审核
	StatusUserUnverified Status = 1 << 11 // 用户未验证：等待用户完成验证（如邮箱验证）

	// 扩展位（位 12 开始）
	// 预留 51 位可用于业务自定义状态（63 - 12 = 51）
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

	// StatusAllUnverified 所有未验证状态的组合（系统未验证 | 管理员未验证 | 用户未验证）
	StatusAllUnverified Status = StatusSysUnverified | StatusAdmUnverified | StatusUserUnverified

	// StatusAllSystem 所有系统级状态的组合
	StatusAllSystem Status = StatusSysDeleted | StatusSysDisabled | StatusSysHidden | StatusSysUnverified

	// StatusAllAdmin 所有管理员级状态的组合
	StatusAllAdmin Status = StatusAdmDeleted | StatusAdmDisabled | StatusAdmHidden | StatusAdmUnverified

	// StatusAllUser 所有用户级状态的组合
	StatusAllUser Status = StatusUserDeleted | StatusUserDisabled | StatusUserHidden | StatusUserUnverified
)

// 状态值边界常量（用于运行时检查）
const (
	// maxValidBit 最大有效位数（int64 有 63 位可用，第 63 位为符号位）
	maxValidBit = 62

	// MaxStatus 最大合法状态值（所有 63 位都为 1，但排除符号位）
	MaxStatus Status = (1 << maxValidBit) - 1
)

// IsValid 检查状态值是否合法（运行时安全检查）
//
// 检查规则：
// - 不能为负数（符号位不能为 1）
// - 不能超过最大值（避免溢出）
//
// 使用场景：
// - 从外部输入创建 Status 时进行验证
// - 在自定义状态时检查是否超出范围
//
// 时间复杂度：O(1)
// 内存分配：0
//
// 示例：
//
//	s := Status(100)
//	if s.IsValid() {
//	    // 安全使用
//	}
func (s Status) IsValid() bool {
	// 负数检查：int64 的负数最高位为 1
	// 溢出检查：不应超过所有有效位的组合
	return s >= 0 && s <= MaxStatus
}

// Set 设置指定的状态位（追加状态）
//
// 使用场景：添加新状态，不影响已有状态
// 时间复杂度：O(1)
// 内存分配：0
//
// 示例：
//
//	var s Status
//	s.Set(StatusUserDisabled)  // 设置用户禁用状态
//	s.Set(StatusSysHidden)     // 追加系统隐藏状态
//
// 注意：此方法会修改接收者本身，传入指针才能生效
func (s *Status) Set(flag Status) {
	// 使用按位或运算，将指定位设置为 1
	// 例如：0000 | 0010 = 0010
	*s |= flag
}

// Unset 取消指定的状态位（移除状态）
//
// 使用场景：移除特定状态，保留其他状态
// 时间复杂度：O(1)
// 内存分配：0
//
// 示例：
//
//	s := StatusUserDisabled | StatusSysHidden
//	s.Unset(StatusUserDisabled)  // 仅移除用户禁用状态
//
// 注意：使用按位清除运算（AND NOT），精确移除指定位
func (s *Status) Unset(flag Status) {
	// &^ 是按位清除运算符（AND NOT）
	// 将 flag 中为 1 的位在 s 中清零
	// 例如：0011 &^ 0010 = 0001
	*s &^= flag
}

// Toggle 切换指定的状态位（翻转状态）
//
// 使用场景：开关式状态切换，有则删除，无则添加
// 时间复杂度：O(1)
// 内存分配：0
//
// 示例：
//
//	var s Status
//	s.Toggle(StatusUserDisabled)  // 首次切换：添加状态
//	s.Toggle(StatusUserDisabled)  // 再次切换：移除状态
//
// 注意：适用于布尔型状态的快速切换
func (s *Status) Toggle(flag Status) {
	// 使用异或运算，相同为 0，不同为 1
	// 例如：0011 ^ 0010 = 0001
	*s ^= flag
}

// Merge 保留与指定状态位相同的部分，其他位清除（交集运算）
//
// 使用场景：过滤状态，只保留指定的状态位
// 时间复杂度：O(1)
// 内存分配：0
//
// 示例：
//
//	s := StatusUserDisabled | StatusSysHidden | StatusAdmDeleted
//	s.Merge(StatusUserDisabled | StatusAdmDeleted)  // 只保留这两个状态
//
// 警告：此操作会清除所有未在 flag 中指定的状态位
func (s *Status) Merge(flag Status) {
	// 使用按位与运算，只保留两者都为 1 的位
	// 例如：0111 & 0011 = 0011
	*s &= flag
}

// Contain 检查是否包含指定的状态位（精确匹配）
//
// 使用场景：检查是否同时包含所有指定的状态位
// 时间复杂度：O(1)
// 内存分配：0
//
// 示例：
//
//	s := StatusUserDisabled | StatusSysHidden
//	s.Contain(StatusUserDisabled)                      // true
//	s.Contain(StatusUserDisabled | StatusSysHidden)    // true
//	s.Contain(StatusUserDisabled | StatusAdmDeleted)   // false（缺少 StatusAdmDeleted）
//
// 注意：与 HasAll 功能相同，但参数为单个 Status 值
func (s Status) Contain(flag Status) bool {
	// 检查 flag 的所有位是否都在 s 中
	// s & flag 会保留 s 和 flag 共有的位
	// 如果结果等于 flag，说明 flag 的所有位都在 s 中
	return s&flag == flag
}

// HasAny 检查是否包含任意一个指定的状态位（或运算）
//
// 使用场景：检查是否包含多个候选状态中的至少一个
// 时间复杂度：O(1) - 优化为单次位运算
// 内存分配：0
//
// 示例：
//
//	s := StatusUserDisabled
//	s.HasAny(StatusUserDisabled, StatusAdmDisabled)  // true（包含第一个）
//	s.HasAny(StatusSysDeleted, StatusAdmDeleted)     // false（都不包含）
//
// 性能优化：使用预定义的状态组合常量效率更高
func (s Status) HasAny(flags ...Status) bool {
	// 性能优化：如果没有传入任何标志，直接返回 false
	if len(flags) == 0 {
		return false
	}

	// 优化：先将所有 flags 合并为一个，然后进行单次位运算
	var combined Status
	for _, flag := range flags {
		combined |= flag
	}

	// 单次位运算检查是否有任何交集
	return s&combined != 0
}

// HasAll 检查是否包含所有指定的状态位（与运算）
//
// 使用场景：检查是否同时满足多个状态条件
// 时间复杂度：O(1) - 优化为单次位运算
// 内存分配：0
//
// 示例：
//
//	s := StatusUserDisabled | StatusSysHidden
//	s.HasAll(StatusUserDisabled, StatusSysHidden)  // true（都包含）
//	s.HasAll(StatusUserDisabled, StatusAdmDeleted) // false（缺少第二个）
//
// 性能优化：使用预定义的状态组合常量效率更高
func (s Status) HasAll(flags ...Status) bool {
	// 性能优化：如果没有传入任何标志，直接返回 true（逻辑上正确）
	if len(flags) == 0 {
		return true
	}

	// 优化：先将所有 flags 合并为一个，然后进行单次位运算
	var combined Status
	for _, flag := range flags {
		combined |= flag
	}

	// 单次位运算检查是否包含所有位
	return s&combined == combined
}

// Clear 清除所有状态位（重置为零值）
//
// 使用场景：重置状态，移除所有标记
// 时间复杂度：O(1)
// 内存分配：0
//
// 示例：
//
//	s := StatusUserDisabled | StatusSysHidden
//	s.Clear()  // s 变为 StatusNone
func (s *Status) Clear() {
	*s = StatusNone
}

// Equal 检查状态是否完全匹配（精确相等）
//
// 使用场景：判断两个状态是否完全一致
// 时间复杂度：O(1)
// 内存分配：0
//
// 示例：
//
//	s1 := StatusUserDisabled | StatusSysHidden
//	s2 := StatusUserDisabled | StatusSysHidden
//	s1.Equal(s2)  // true
//
// 注意：与 == 运算符效果相同，但语义更清晰
func (s Status) Equal(status Status) bool {
	return s == status
}

// SetMultiple 批量设置多个状态位（批量追加）
//
// 使用场景：一次性添加多个状态
// 时间复杂度：O(1) - 优化为单次位运算
// 内存分配：0
//
// 示例：
//
//	var s Status
//	s.SetMultiple(StatusUserDisabled, StatusSysHidden, StatusAdmUnverified)
//
// 性能优化：预先合并所有标志，进行单次 OR 运算
func (s *Status) SetMultiple(flags ...Status) {
	// 优化：将所有 flags 先合并，然后一次性设置
	var combined Status
	for _, flag := range flags {
		combined |= flag
	}
	*s |= combined
}

// UnsetMultiple 批量取消多个状态位（批量移除）
//
// 使用场景：一次性移除多个状态
// 时间复杂度：O(1) - 优化为单次位运算
// 内存分配：0
//
// 示例：
//
//	s := StatusUserDisabled | StatusSysHidden | StatusAdmDeleted
//	s.UnsetMultiple(StatusUserDisabled, StatusSysHidden)
//
// 性能优化：预先合并所有标志，进行单次 AND NOT 运算
func (s *Status) UnsetMultiple(flags ...Status) {
	// 优化：将所有 flags 先合并，然后一次性清除
	var combined Status
	for _, flag := range flags {
		combined |= flag
	}
	*s &^= combined
}

// IsDeleted 检查是否被标记为删除（任意级别）
//
// 业务语义：被删除的内容通常不应该被访问或展示
// 时间复杂度：O(1)
// 内存分配：0
//
// 返回值：包含任意删除状态时返回 true
//
// 性能优化：使用预定义的状态组合常量
func (s Status) IsDeleted() bool {
	return s&StatusAllDeleted != 0
}

// IsDisable 检查是否被禁用（任意级别）
//
// 业务语义：被禁用的内容暂时不可用，可能需要管理员或用户操作恢复
// 时间复杂度：O(1)
// 内存分配：0
//
// 返回值：包含任意禁用状态时返回 true
//
// 性能优化：使用预定义的状态组合常量
func (s Status) IsDisable() bool {
	return s&StatusAllDisabled != 0
}

// IsHidden 检查是否被隐藏（任意级别）
//
// 业务语义：被隐藏的内容不对外展示，但功能可能正常
// 时间复杂度：O(1)
// 内存分配：0
//
// 返回值：包含任意隐藏状态时返回 true
//
// 性能优化：使用预定义的状态组合常量
func (s Status) IsHidden() bool {
	return s&StatusAllHidden != 0
}

// IsUnverified 检查是否未验证（任意级别）
//
// 业务语义：未验证的内容可能需要审核或用户完成验证流程
// 时间复杂度：O(1)
// 内存分配：0
//
// 返回值：包含任意未验证状态时返回 true
//
// 性能优化：使用预定义的状态组合常量
func (s Status) IsUnverified() bool {
	return s&StatusAllUnverified != 0
}

// CanEnable 检查是否为可启用状态（业务可用性检查）
//
// 业务规则：未被删除且未被禁用的内容才可以启用
// 时间复杂度：O(1)
// 内存分配：0
//
// 返回值：可以启用返回 true，否则返回 false
//
// 使用场景：在启用某个功能前检查状态是否允许
func (s Status) CanEnable() bool {
	return !s.IsDeleted() && !s.IsDisable()
}

// CanVisible 检查是否为可见状态（业务可见性检查）
//
// 业务规则：可启用且未被隐藏的内容才可见
// 时间复杂度：O(1)
// 内存分配：0
//
// 返回值：可以对外展示返回 true，否则返回 false
//
// 使用场景：在列表查询中过滤不可见的内容
func (s Status) CanVisible() bool {
	return s.CanEnable() && !s.IsHidden()
}

// CanVerified 检查是否为已验证状态（业务验证检查）
//
// 业务规则：可见且已通过验证的内容才算完全可用
// 时间复杂度：O(1)
// 内存分配：0
//
// 返回值：已验证返回 true，否则返回 false
//
// 使用场景：在需要验证的业务流程中检查状态
func (s Status) CanVerified() bool {
	return s.CanVisible() && !s.IsUnverified()
}

// Value 实现 driver.Valuer 接口，用于数据库写入
//
// 数据库存储：将 Status 转换为 int64 存储
// 索引支持：int64 类型支持高效的 B-tree 索引
// 时间复杂度：O(1)
// 内存分配：0
//
// 错误处理：此方法不会返回错误（int64 转换总是成功）
func (s Status) Value() (driver.Value, error) {
	return int64(s), nil
}

// Scan 实现 sql.Scanner 接口，用于从数据库读取
//
// 支持的数据库类型：
// - int64: 标准整数类型
// - int: Go 原生整数类型
// - []byte: JSON 格式的数字
//
// 时间复杂度：O(1)，除 []byte 需要 JSON 解析
// 内存分配：仅 []byte 类型需要分配
//
// 错误处理：
// - nil 值会被设置为 StatusNone
// - 不支持的类型会返回明确的错误信息
// - JSON 解析失败会返回原始错误
// - 添加溢出检查，防止数据库中的异常值
func (s *Status) Scan(value interface{}) error {
	// 处理 NULL 值：数据库中的 NULL 映射为零值
	if value == nil {
		*s = StatusNone
		return nil
	}

	// 类型断言：支持常见的数据库驱动返回类型
	switch v := value.(type) {
	case int64:
		// 最常见的数据库整数类型
		// 添加边界检查，防止数据库中存储了异常值
		if v < 0 {
			return fmt.Errorf("invalid Status value: negative number %d is not allowed (sign bit conflict)", v)
		}
		*s = Status(v)

	case int:
		// Go 原生整数类型（某些驱动可能返回）
		if v < 0 {
			return fmt.Errorf("invalid Status value: negative number %d is not allowed (sign bit conflict)", v)
		}
		*s = Status(v)

	case uint64:
		// 无符号整数类型
		// 检查是否超过 int64 的最大值
		if v > uint64(MaxStatus) {
			return fmt.Errorf("invalid Status value: %d exceeds maximum allowed value %d (overflow)", v, MaxStatus)
		}
		*s = Status(v)

	case []byte:
		// JSON 或文本格式（某些驱动如 SQLite）
		var num int64
		if err := json.Unmarshal(v, &num); err != nil {
			return fmt.Errorf("failed to unmarshal Status from bytes: %w", err)
		}
		if num < 0 {
			return fmt.Errorf("invalid Status value: negative number %d is not allowed (sign bit conflict)", num)
		}
		*s = Status(num)

	default:
		// 不支持的类型：返回详细的错误信息
		return fmt.Errorf("cannot scan type %T into Status: unsupported database type, expected int64, int, uint64, or []byte", value)
	}

	return nil
}

// MarshalJSON 实现 json.Marshaler 接口，用于 JSON 序列化
//
// JSON 格式：直接序列化为数字（非字符串）
// 优势：
// - 体积小：数字比字符串紧凑
// - 性能好：无需字符串转换
// - 类型安全：客户端可以直接用数字类型接收
//
// 示例输出：{"status": 5} 而非 {"status": "5"}
//
// 时间复杂度：O(1)
// 内存分配：仅 JSON 编码器分配
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(s))
}

// UnmarshalJSON 实现 json.Unmarshaler 接口，用于 JSON 反序列化
//
// 支持的 JSON 格式：数字类型
// 示例：{"status": 5} 或 {"status": 0}
//
// 时间复杂度：O(1)
// 内存分配：仅 JSON 解码器分配
//
// 错误处理：
// - JSON 格式错误会返回解析错误
// - 非数字类型会返回类型错误
// - 添加边界检查，防止恶意输入
func (s *Status) UnmarshalJSON(data []byte) error {
	var num int64
	if err := json.Unmarshal(data, &num); err != nil {
		return fmt.Errorf("failed to unmarshal Status from JSON: invalid format, expected integer number: %w", err)
	}

	// 边界检查：防止反序列化时的异常值
	if num < 0 {
		return fmt.Errorf("failed to unmarshal Status from JSON: negative value %d is not allowed (sign bit conflict)", num)
	}

	*s = Status(num)
	return nil
}

// String 实现 fmt.Stringer 接口，用于调试和日志输出
//
// 输出格式：Status(数值) 或具体的状态描述
// 时间复杂度：O(1)
// 内存分配：字符串拼接会有少量分配
//
// 示例：
//
//	fmt.Println(StatusUserDisabled)  // 输出：Status(32)
func (s Status) String() string {
	// 特殊值处理
	if s == StatusNone {
		return "Status(None)"
	}

	// 返回数值表示，便于调试
	return fmt.Sprintf("Status(%d)", int64(s))
}
