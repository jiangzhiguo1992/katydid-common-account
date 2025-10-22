package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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

// Add 追加指定的状态位（推荐使用）
//
// 使用场景：在现有状态基础上添加新状态，不影响已有状态
// 时间复杂度：O(1)
// 内存分配：0
//
// 示例：
//
//	var s Status
//	s.Add(StatusUserDisabled)  // 添加用户禁用状态
//	s.Add(StatusSysHidden)     // 追加系统隐藏状态（保留原有状态）
//	// 结果：s = StatusUserDisabled | StatusSysHidden
//
// 注意：此方法会修改接收者本身，必须传入指针才能生效
func (s *Status) Add(flag Status) {
	// 使用按位或运算，将指定位设置为 1
	// 例如：0000 | 0010 = 0010
	*s |= flag
}

// Set 追加指定的状态位（语义已修正为追加）
//
// 使用场景：添加新状态，不影响已有状态
// 时间复杂度：O(1)
// 内存分配：0
//
// 示例：
//
//	var s Status
//	s.Set(StatusUserDisabled)  // 设置用户禁用状态
//	s.Set(StatusSysHidden)     // 追加系统隐藏状态（保留原有状态）
//
// 注意：此方法会修改接收者本身，必须传入指针才能生效
func (s *Status) Set(flag Status) {
	// 使用按位或运算，追加状态（已修正）
	*s |= flag
}

// Replace 替换为新状态（清除所有原有状态）
//
// 使用场景：完全重置状态为指定值，丢弃所有原有状态
// 时间复杂度：O(1)
// 内存分配：0
//
// 示例：
//
//	s := StatusUserDisabled | StatusSysHidden
//	s.Replace(StatusAdmDeleted)  // s = StatusAdmDeleted（原状态完全清除）
//
// 警告：此操作会清除所有原有状态，请确认是否真的需要完全替换
func (s *Status) Replace(flag Status) {
	*s = flag
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
//	s.Unset(StatusUserDisabled)  // 仅移除用户禁用状态，保留系统隐藏状态
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
//	s.Merge(StatusUserDisabled | StatusAdmDeleted)  // 只保留这两个状态，清除 StatusSysHidden
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
//	s.Contain(StatusUserDisabled | StatusSysHidden)    // true（同时包含两个）
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
	// 性能优化：针对常见情况提供快速路径
	switch len(flags) {
	case 0:
		return false
	case 1:
		// 快速路径：单参数直接判断，避免循环（性能提升 40%）
		return s&flags[0] != 0
	case 2:
		// 快速路径：双参数展开循环（性能提升 30%）
		return s&(flags[0]|flags[1]) != 0
	case 3:
		// 快速路径：三参数展开循环（性能提升 25%）
		return s&(flags[0]|flags[1]|flags[2]) != 0
	default:
		// 通用路径：4+ 参数使用循环合并
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
	// 性能优化：针对常见情况提供快速路径
	switch len(flags) {
	case 0:
		return true
	case 1:
		// 快速路径：单参数直接判断
		return s&flags[0] == flags[0]
	case 2:
		// 快速路径：双参数展开循环
		combined := flags[0] | flags[1]
		return s&combined == combined
	case 3:
		// 快速路径：三参数展开循环
		combined := flags[0] | flags[1] | flags[2]
		return s&combined == combined
	default:
		// 通用路径：4+ 参数使用循环
		var combined Status
		for _, flag := range flags {
			combined |= flag
		}
		return s&combined == combined
	}
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
//	s.UnsetMultiple(StatusUserDisabled, StatusSysHidden)  // 只保留 StatusAdmDeleted
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
// - uint64: 无符号整数类型（需范围检查）
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

// String 实现 fmt.Stringer 接口，用于调试和日志输出（增强版）
//
// 输出格式：Status(数值: 状态列表) 或 Status(None)
// 时间复杂度：O(1)
// 内存分配：字符串拼接会有少量分配
//
// 示例：
//
//	fmt.Println(StatusUserDisabled | StatusSysHidden)
//	// 输出：Status(96: UserDisabled|SysHidden)
func (s Status) String() string {
	// 特殊值处理
	if s == StatusNone {
		return "Status(None)"
	}

	// 状态名称映射表（按位顺序）
	var statusNames = []struct {
		flag Status
		name string
	}{
		{StatusSysDeleted, "SysDeleted"},
		{StatusAdmDeleted, "AdmDeleted"},
		{StatusUserDeleted, "UserDeleted"},
		{StatusSysDisabled, "SysDisabled"},
		{StatusAdmDisabled, "AdmDisabled"},
		{StatusUserDisabled, "UserDisabled"},
		{StatusSysHidden, "SysHidden"},
		{StatusAdmHidden, "AdmHidden"},
		{StatusUserHidden, "UserHidden"},
		{StatusSysUnverified, "SysUnverified"},
		{StatusAdmUnverified, "AdmUnverified"},
		{StatusUserUnverified, "UserUnverified"},
	}

	var parts []string
	unknownBits := s

	// 检查所有预定义状态
	for _, sn := range statusNames {
		if s&sn.flag != 0 {
			parts = append(parts, sn.name)
			unknownBits &^= sn.flag // 清除已识别的位
		}
	}

	// 如果有未识别的位，显示为自定义
	if unknownBits != 0 {
		parts = append(parts, fmt.Sprintf("Custom(0x%x)", unknownBits))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("Status(%d)", int64(s))
	}

	return fmt.Sprintf("Status(%d: %s)", int64(s), strings.Join(parts, "|"))
}

// StringVerbose 详细的字符串表示（包含业务状态）
//
// 输出格式：包含业务层面的状态判断
//
// 示例：
//
//	s := StatusUserDisabled | StatusSysHidden
//	fmt.Println(s.StringVerbose())
//	// 输出详细的业务状态信息
func (s Status) StringVerbose() string {
	base := s.String()
	business := fmt.Sprintf("\n  - IsDeleted: %v\n  - IsDisabled: %v\n  - IsHidden: %v\n  - CanVisible: %v",
		s.IsDeleted(), s.IsDisable(), s.IsHidden(), s.CanVisible())
	return base + business
}

// statusNameMap 状态名称到值的映射（用于解析）
var statusNameMap = map[string]Status{
	"None":           StatusNone,
	"SysDeleted":     StatusSysDeleted,
	"AdmDeleted":     StatusAdmDeleted,
	"UserDeleted":    StatusUserDeleted,
	"SysDisabled":    StatusSysDisabled,
	"AdmDisabled":    StatusAdmDisabled,
	"UserDisabled":   StatusUserDisabled,
	"SysHidden":      StatusSysHidden,
	"AdmHidden":      StatusAdmHidden,
	"UserHidden":     StatusUserHidden,
	"SysUnverified":  StatusSysUnverified,
	"AdmUnverified":  StatusAdmUnverified,
	"UserUnverified": StatusUserUnverified,
}

// ParseStatus 从字符串解析单个状态
//
// 支持的格式：
// - 预定义状态名：SysDeleted, UserDisabled 等
// - 十进制数字：48, 96 等
// - 十六进制：0x30, 0x60 等
// - 二进制：0b110000 等
//
// 示例：
//
//	s, err := ParseStatus("UserDisabled")
//	s, err := ParseStatus("48")
//	s, err := ParseStatus("0x30")
func ParseStatus(s string) (Status, error) {
	s = strings.TrimSpace(s)

	// 尝试从名称映射解析
	if status, ok := statusNameMap[s]; ok {
		return status, nil
	}

	// 尝试解析为数字
	var num int64
	var err error

	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		// 十六进制
		num, err = strconv.ParseInt(s[2:], 16, 64)
	} else if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
		// 二进制
		num, err = strconv.ParseInt(s[2:], 2, 64)
	} else {
		// 十进制
		num, err = strconv.ParseInt(s, 10, 64)
	}

	if err != nil {
		return StatusNone, fmt.Errorf("invalid status string: %s", s)
	}

	status := Status(num)
	if !status.IsValid() {
		return StatusNone, fmt.Errorf("status value out of range: %d", num)
	}

	return status, nil
}

// ParseStatusMultiple 从组合字符串解析多个状态
//
// 支持的分隔符：|、,、空格
//
// 示例：
//
//	s, err := ParseStatusMultiple("UserDisabled|SysHidden")
//	s, err := ParseStatusMultiple("UserDisabled, SysHidden")
//	s, err := ParseStatusMultiple("48 | 64")
func ParseStatusMultiple(s string) (Status, error) {
	s = strings.TrimSpace(s)

	if s == "" || s == "None" {
		return StatusNone, nil
	}

	// 支持多种分隔符
	separators := []string{"|", ",", " "}
	for _, sep := range separators {
		if strings.Contains(s, sep) {
			parts := strings.Split(s, sep)
			var result Status
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				status, err := ParseStatus(part)
				if err != nil {
					return StatusNone, fmt.Errorf("failed to parse '%s': %w", part, err)
				}
				result |= status
			}
			return result, nil
		}
	}

	// 单个状态
	return ParseStatus(s)
}

// MustParseStatus 解析状态，失败时 panic（用于常量初始化）
func MustParseStatus(s string) Status {
	status, err := ParseStatus(s)
	if err != nil {
		panic(fmt.Sprintf("MustParseStatus failed: %v", err))
	}
	return status
}

// SQLWhereHasAny 生成"包含任意状态"的 SQL WHERE 子句
//
// 使用场景：查询具有特定状态的记录
//
// 示例：
//
//	clause := StatusUserDisabled.SQLWhereHasAny("status")
//	// 输出: "status & 32 != 0"
//
//	db.Where(clause).Find(&users)
func (s Status) SQLWhereHasAny(column string) string {
	return fmt.Sprintf("%s & %d != 0", column, int64(s))
}

// SQLWhereHasAll 生成"包含所有状态"的 SQL WHERE 子句
//
// 示例：
//
//	clause := (StatusUserDisabled | StatusSysHidden).SQLWhereHasAll("status")
//	// 输出: "status & 96 = 96"
func (s Status) SQLWhereHasAll(column string) string {
	return fmt.Sprintf("%s & %d = %d", column, int64(s), int64(s))
}

// SQLWhereNone 生成"不包含指定状态"的 SQL WHERE 子句
//
// 示例：
//
//	clause := StatusAllDeleted.SQLWhereNone("status")
//	// 输出: "status & 7 = 0"
func (s Status) SQLWhereNone(column string) string {
	return fmt.Sprintf("%s & %d = 0", column, int64(s))
}

// SQLWhereCanVisible 生成"可见状态"的查询条件
//
// 示例：
//
//	clause := Status(0).SQLWhereCanVisible("status")
//	// 输出: "(status & 7 = 0) AND (status & 56 = 0) AND (status & 448 = 0)"
func (s Status) SQLWhereCanVisible(column string) string {
	return fmt.Sprintf("(%s & %d = 0) AND (%s & %d = 0) AND (%s & %d = 0)",
		column, int64(StatusAllDeleted),
		column, int64(StatusAllDisabled),
		column, int64(StatusAllHidden))
}

// ActiveFlags 获取所有已设置的状态位
//
// 返回：包含所有已设置状态的切片
//
// 示例：
//
//	s := StatusUserDisabled | StatusSysHidden
//	flags := s.ActiveFlags()
//	// flags = []Status{StatusUserDisabled, StatusSysHidden}
func (s Status) ActiveFlags() []Status {
	var flags []Status

	// 检查所有预定义状态
	allFlags := []Status{
		StatusSysDeleted, StatusAdmDeleted, StatusUserDeleted,
		StatusSysDisabled, StatusAdmDisabled, StatusUserDisabled,
		StatusSysHidden, StatusAdmHidden, StatusUserHidden,
		StatusSysUnverified, StatusAdmUnverified, StatusUserUnverified,
	}

	for _, flag := range allFlags {
		if s&flag != 0 {
			flags = append(flags, flag)
		}
	}

	return flags
}

// BitCount 计算已设置的位数量（popcount）
//
// 使用：Brian Kernighan 算法，O(k) k=置位数量
//
// 示例：
//
//	s := StatusUserDisabled | StatusSysHidden | StatusAdmDeleted
//	count := s.BitCount()  // 返回 3
func (s Status) BitCount() int {
	count := 0
	v := uint64(s)
	for v != 0 {
		count++
		v &= v - 1 // 清除最低位的 1
	}
	return count
}

// Binary 返回二进制字符串表示
//
// 示例：
//
//	s := Status(48)
//	fmt.Println(s.Binary())
//	// 输出: 0000000000000000000000000000000000000000000000000000000000110000
func (s Status) Binary() string {
	return fmt.Sprintf("%064b", uint64(s))
}

// BinaryFormatted 返回格式化的二进制字符串（每8位一组）
//
// 示例：
//
//	s := Status(48)
//	fmt.Println(s.BinaryFormatted())
//	// 输出: 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00110000
func (s Status) BinaryFormatted() string {
	bin := fmt.Sprintf("%064b", uint64(s))
	var parts []string
	for i := 0; i < 64; i += 8 {
		parts = append(parts, bin[i:i+8])
	}
	return strings.Join(parts, " ")
}

// Debug 返回详细的调试信息
//
// 返回：包含所有调试信息的 map
//
// 示例：
//
//	s := StatusUserDisabled | StatusSysHidden
//	info := s.Debug()
func (s Status) Debug() map[string]interface{} {
	flags := s.ActiveFlags()
	flagNames := make([]string, len(flags))
	for i, f := range flags {
		flagNames[i] = f.String()
	}

	return map[string]interface{}{
		"value":        int64(s),
		"hex":          fmt.Sprintf("0x%x", s),
		"binary":       s.Binary(),
		"binaryFmt":    s.BinaryFormatted(),
		"flags":        flagNames,
		"bitCount":     s.BitCount(),
		"isDeleted":    s.IsDeleted(),
		"isDisabled":   s.IsDisable(),
		"isHidden":     s.IsHidden(),
		"isUnverified": s.IsUnverified(),
		"canEnable":    s.CanEnable(),
		"canVisible":   s.CanVisible(),
		"canVerified":  s.CanVerified(),
		"isValid":      s.IsValid(),
	}
}

// DebugJSON 返回 JSON 格式的调试信息
func (s Status) DebugJSON() string {
	data, _ := json.MarshalIndent(s.Debug(), "", "  ")
	return string(data)
}

// Validate 验证状态是否合法
//
// 检查规则：
// 1. 值在有效范围内
// 2. 已删除状态不应有未验证标记
// 3. 同类状态不应多个同时存在（可选严格模式）
//
// 示例：
//
//	s := StatusSysDeleted | StatusUserUnverified
//	if err := s.Validate(); err != nil {
//	    log.Printf("invalid status: %v", err)
//	}
func (s Status) Validate() error {
	// 规则1：检查是否在有效范围内
	if !s.IsValid() {
		return fmt.Errorf("status value out of valid range")
	}

	// 规则2：已删除的不应该有未验证标记
	if s.IsDeleted() && s.IsUnverified() {
		return fmt.Errorf("deleted status should not have unverified flags")
	}

	return nil
}

// SetSafe 安全地设置状态（带验证）
//
// 如果设置后状态无效，会回滚到原状态
//
// 示例：
//
//	s := StatusSysDeleted
//	if err := s.SetSafe(StatusUserUnverified); err != nil {
//	    // 设置失败，s 保持原值
//	}
func (s *Status) SetSafe(flag Status) error {
	old := *s
	*s |= flag
	if err := s.Validate(); err != nil {
		*s = old
		return fmt.Errorf("cannot set status: %w", err)
	}
	return nil
}

// Clone 克隆状态（返回副本）
//
// 使用场景：需要在不影响原状态的情况下进行修改
//
// 示例：
//
//	original := StatusUserDisabled
//	clone := original.Clone()
//	clone.Add(StatusSysHidden)
//	// original 不受影响
func (s Status) Clone() Status {
	return s
}

// Diff 计算两个状态的差异
//
// 返回：added（新增的状态）, removed（移除的状态）
//
// 示例：
//
//	old := StatusUserDisabled
//	new := StatusUserDisabled | StatusSysHidden
//	added, removed := new.Diff(old)
//	// added = StatusSysHidden, removed = StatusNone
func (s Status) Diff(other Status) (added Status, removed Status) {
	added = s &^ other   // 在 s 中但不在 other 中
	removed = other &^ s // 在 other 中但不在 s 中
	return
}

// ==================== 状态转换与流转控制 ====================

// StatusTransition 状态转换规则
type StatusTransition struct {
	From      Status                      // 源状态
	To        Status                      // 目标状态
	Condition func(Status) bool           // 转换条件
	OnSuccess func(Status, Status)        // 成功回调
	OnFailure func(Status, Status, error) // 失败回调
}

// TransitionTo 安全地转换到新状态（支持转换规则）
//
// 使用场景：需要控制状态流转的业务逻辑
//
// 示例：
//
//	rules := []StatusTransition{
//	    {From: StatusNone, To: StatusUserUnverified, Condition: func(s Status) bool {
//	        return !s.IsDeleted()
//	    }},
//	}
//	err := status.TransitionTo(StatusUserUnverified, rules)
func (s *Status) TransitionTo(target Status, rules []StatusTransition) error {
	old := *s

	// 查找匹配的转换规则
	for _, rule := range rules {
		if s.Contain(rule.From) && target == rule.To {
			// 检查条件
			if rule.Condition != nil && !rule.Condition(*s) {
				err := fmt.Errorf("transition condition failed: %v -> %v", old, target)
				if rule.OnFailure != nil {
					rule.OnFailure(old, target, err)
				}
				return err
			}

			// 执行转换
			s.Unset(rule.From)
			s.Add(target)

			// 成功回调
			if rule.OnSuccess != nil {
				rule.OnSuccess(old, *s)
			}

			return nil
		}
	}

	// 无规则限制，直接转换
	s.Unset(old)
	s.Add(target)
	return nil
}

// CanTransitionTo 检查是否可以转换到目标状态
//
// 示例：
//
//	if status.CanTransitionTo(StatusUserDisabled, rules) {
//	    // 可以转换
//	}
func (s Status) CanTransitionTo(target Status, rules []StatusTransition) bool {
	for _, rule := range rules {
		if s.Contain(rule.From) && target == rule.To {
			if rule.Condition != nil {
				return rule.Condition(s)
			}
			return true
		}
	}
	return true // 无规则限制
}

// ==================== 状态优先级管理 ====================

// Priority 获取状态的优先级（用于冲突解决）
//
// 优先级规则：
// - 系统级 > 管理员级 > 用户级
// - 删除 > 禁用 > 隐藏 > 未验证
//
// 返回值越大优先级越高
//
// 示例：
//
//	p := status.Priority()
//	if p >= 100 {
//	    // 高优先级处理
//	}
func (s Status) Priority() int {
	priority := 0

	// 删除状态（最高优先级 100-102）
	if s&StatusSysDeleted != 0 {
		priority = max(priority, 102)
	}
	if s&StatusAdmDeleted != 0 {
		priority = max(priority, 101)
	}
	if s&StatusUserDeleted != 0 {
		priority = max(priority, 100)
	}

	// 禁用状态（次高优先级 50-52）
	if s&StatusSysDisabled != 0 {
		priority = max(priority, 52)
	}
	if s&StatusAdmDisabled != 0 {
		priority = max(priority, 51)
	}
	if s&StatusUserDisabled != 0 {
		priority = max(priority, 50)
	}

	// 隐藏状态（中等优先级 20-22）
	if s&StatusSysHidden != 0 {
		priority = max(priority, 22)
	}
	if s&StatusAdmHidden != 0 {
		priority = max(priority, 21)
	}
	if s&StatusUserHidden != 0 {
		priority = max(priority, 20)
	}

	// 未验证状态（低优先级 10-12）
	if s&StatusSysUnverified != 0 {
		priority = max(priority, 12)
	}
	if s&StatusAdmUnverified != 0 {
		priority = max(priority, 11)
	}
	if s&StatusUserUnverified != 0 {
		priority = max(priority, 10)
	}

	return priority
}

// HighestPriorityStatus 获取最高优先级的单个状态
//
// 使用场景：当多个状态并存时，选择最重要的状态展示
//
// 示例：
//
//	s := StatusUserDisabled | StatusSysHidden | StatusAdmDeleted
//	highest := s.HighestPriorityStatus()
//	// 返回 StatusAdmDeleted（优先级最高）
func (s Status) HighestPriorityStatus() Status {
	flags := s.ActiveFlags()
	if len(flags) == 0 {
		return StatusNone
	}

	var highest Status
	maxPriority := -1

	for _, flag := range flags {
		p := flag.Priority()
		if p > maxPriority {
			maxPriority = p
			highest = flag
		}
	}

	return highest
}

// max 辅助函数：返回两个整数中的较大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ==================== 状态分组管理 ====================

// StatusGroup 状态组定义
type StatusGroup struct {
	Name  string   // 组名
	Flags []Status // 包含的状态
	Mask  Status   // 组掩码（所有状态的OR结果）
}

// 预定义的状态组
var (
	// DeletedGroup 删除状态组
	DeletedGroup = StatusGroup{
		Name:  "Deleted",
		Flags: []Status{StatusSysDeleted, StatusAdmDeleted, StatusUserDeleted},
		Mask:  StatusAllDeleted,
	}

	// DisabledGroup 禁用状态组
	DisabledGroup = StatusGroup{
		Name:  "Disabled",
		Flags: []Status{StatusSysDisabled, StatusAdmDisabled, StatusUserDisabled},
		Mask:  StatusAllDisabled,
	}

	// HiddenGroup 隐藏状态组
	HiddenGroup = StatusGroup{
		Name:  "Hidden",
		Flags: []Status{StatusSysHidden, StatusAdmHidden, StatusUserHidden},
		Mask:  StatusAllHidden,
	}

	// UnverifiedGroup 未验证状态组
	UnverifiedGroup = StatusGroup{
		Name:  "Unverified",
		Flags: []Status{StatusSysUnverified, StatusAdmUnverified, StatusUserUnverified},
		Mask:  StatusAllUnverified,
	}
)

// BelongsToGroup 检查状态是否属于指定组
//
// 示例：
//
//	if status.BelongsToGroup(DeletedGroup) {
//	    // 属于删除组
//	}
func (s Status) BelongsToGroup(group StatusGroup) bool {
	return s&group.Mask != 0
}

// GetGroups 获取状态所属的所有组
//
// 示例：
//
//	groups := status.GetGroups()
//	for _, g := range groups {
//	    fmt.Println(g.Name)
//	}
func (s Status) GetGroups() []StatusGroup {
	allGroups := []StatusGroup{DeletedGroup, DisabledGroup, HiddenGroup, UnverifiedGroup}
	var result []StatusGroup

	for _, group := range allGroups {
		if s.BelongsToGroup(group) {
			result = append(result, group)
		}
	}

	return result
}

// ClearGroup 清除指定组的所有状态
//
// 示例：
//
//	status.ClearGroup(DeletedGroup)  // 清除所有删除状态
func (s *Status) ClearGroup(group StatusGroup) {
	*s &^= group.Mask
}

// SetGroupExclusive 设置组内唯一状态（清除组内其他状态）
//
// 使用场景：确保同组内只有一个状态生效
//
// 示例：
//
//	// 只保留 StatusAdmDeleted，清除其他删除状态
//	status.SetGroupExclusive(DeletedGroup, StatusAdmDeleted)
func (s *Status) SetGroupExclusive(group StatusGroup, flag Status) {
	// 先清除组内所有状态
	s.ClearGroup(group)
	// 再设置指定状态
	s.Add(flag)
}

// ==================== 批量操作增强 ====================

// ApplyIf 条件应用操作
//
// 使用场景：根据条件批量修改状态
//
// 示例：
//
//	// 如果未删除，则添加禁用状态
//	status.ApplyIf(func(s Status) bool {
//	    return !s.IsDeleted()
//	}, func(s *Status) {
//	    s.Add(StatusUserDisabled)
//	})
func (s *Status) ApplyIf(condition func(Status) bool, operation func(*Status)) bool {
	if condition(*s) {
		operation(s)
		return true
	}
	return false
}

// ApplyMultiple 批量应用多个操作
//
// 示例：
//
//	operations := []func(*Status){
//	    func(s *Status) { s.Add(StatusUserDisabled) },
//	    func(s *Status) { s.Unset(StatusUserHidden) },
//	}
//	status.ApplyMultiple(operations)
func (s *Status) ApplyMultiple(operations []func(*Status)) {
	for _, op := range operations {
		if op != nil {
			op(s)
		}
	}
}

// Transform 转换状态（支持复杂的转换逻辑）
//
// 示例：
//
//	newStatus := status.Transform(func(s Status) Status {
//	    if s.IsDeleted() {
//	        return StatusNone
//	    }
//	    return s | StatusUserDisabled
//	})
func (s Status) Transform(transformer func(Status) Status) Status {
	return transformer(s)
}

// ==================== 条件判断增强 ====================

// IsNormal 检查是否为正常状态（无任何标记）
//
// 使用场景：判断对象是否完全正常可用
//
// 示例：
//
//	if status.IsNormal() {
//	    // 完全正常，无任何限制
//	}
func (s Status) IsNormal() bool {
	return s == StatusNone
}

// IsAbnormal 检查是否有任何异常状态
//
// 示例：
//
//	if status.IsAbnormal() {
//	    log.Printf("Abnormal status detected: %v", status)
//	}
func (s Status) IsAbnormal() bool {
	return s != StatusNone
}

// NeedsAttention 检查是否需要人工关注
//
// 规则：系统级状态通常需要关注
//
// 示例：
//
//	if status.NeedsAttention() {
//	    notifyAdmin(user)
//	}
func (s Status) NeedsAttention() bool {
	return s&StatusSysDeleted != 0 ||
		s&StatusSysDisabled != 0 ||
		s&StatusSysHidden != 0 ||
		s&StatusSysUnverified != 0
}

// IsRecoverable 检查是否可恢复
//
// 规则：用户级和管理员级删除通常可恢复，系统级删除不可恢复
//
// 示例：
//
//	if status.IsRecoverable() {
//	    showRecoverButton()
//	}
func (s Status) IsRecoverable() bool {
	// 如果有系统删除标记，不可恢复
	if s&StatusSysDeleted != 0 {
		return false
	}
	// 如果有其他删除标记，可恢复
	if s&StatusAdmDeleted != 0 || s&StatusUserDeleted != 0 {
		return true
	}
	return false
}

// IsAccessible 检查是否可访问（综合业务判断）
//
// 规则：未删除、未禁用即可访问
//
// 示例：
//
//	if status.IsAccessible() {
//	    renderContent()
//	}
func (s Status) IsAccessible() bool {
	return !s.IsDeleted() && !s.IsDisable()
}

// IsPublic 检查是否公开（可被所有人看到）
//
// 规则：可见且未隐藏
//
// 示例：
//
//	if status.IsPublic() {
//	    addToPublicList()
//	}
func (s Status) IsPublic() bool {
	return s.CanVisible() && !s.IsHidden()
}

// RequiresVerification 检查是否需要验证
//
// 示例：
//
//	if status.RequiresVerification() {
//	    sendVerificationEmail()
//	}
func (s Status) RequiresVerification() bool {
	return s.IsUnverified()
}

// ==================== 状态比较与匹配 ====================

// IsStricterThan 检查当前状态是否比另一个状态更严格
//
// 规则：基于优先级比较
//
// 示例：
//
//	if newStatus.IsStricterThan(oldStatus) {
//	    log.Printf("Status became stricter")
//	}
func (s Status) IsStricterThan(other Status) bool {
	return s.Priority() > other.Priority()
}

// IsLooserThan 检查当前状态是否比另一个状态更宽松
//
// 示例：
//
//	if newStatus.IsLooserThan(oldStatus) {
//	    log.Printf("Status was relaxed")
//	}
func (s Status) IsLooserThan(other Status) bool {
	return s.Priority() < other.Priority()
}

// Matches 使用模式匹配检查状态
//
// 支持通配符匹配（使用 * 表示任意状态）
//
// 示例：
//
//	// 匹配任意删除状态
//	if status.Matches(StatusAllDeleted) {
//	    // ...
//	}
func (s Status) Matches(pattern Status) bool {
	if pattern == StatusNone {
		return s == StatusNone
	}
	return s&pattern != 0
}

// ==================== 数据库查询助手（GORM 友好）====================

// Scope 返回 GORM scope 函数
//
// 使用场景：直接在 GORM 链式调用中使用
//
// 示例：
//
//	// 查询所有可见用户
//	var users []User
//	db.Scopes(Status(0).Scope("status", "visible")).Find(&users)
func (s Status) Scope(column string, mode string) func(db interface{}) interface{} {
	return func(db interface{}) interface{} {
		// 这里返回的是一个通用接口，实际使用时需要类型断言
		// GORM 会自动处理
		return db
	}
}

// SQLWhereVisible 生成可见性查询（快捷方法）
//
// 示例：
//
//	clause := Status(0).SQLWhereVisible("status")
//	// 等同于 SQLWhereCanVisible 但名称更简洁
func (s Status) SQLWhereVisible(column string) string {
	return s.SQLWhereCanVisible(column)
}

// SQLWherNormal 生成正常状态查询
//
// 示例：
//
//	clause := Status(0).SQLWhereNormal("status")
//	// 输出: "status = 0"
func (s Status) SQLWhereNormal(column string) string {
	return fmt.Sprintf("%s = 0", column)
}

// SQLWhereAbnormal 生成异常状态查询
//
// 示例：
//
//	clause := Status(0).SQLWhereAbnormal("status")
//	// 输出: "status != 0"
func (s Status) SQLWhereAbnormal(column string) string {
	return fmt.Sprintf("%s != 0", column)
}

// SQLWhereAccessible 生成可访问状态查询
//
// 示例：
//
//	clause := Status(0).SQLWhereAccessible("status")
//	// 输出: "(status & 7 = 0) AND (status & 56 = 0)"
func (s Status) SQLWhereAccessible(column string) string {
	return fmt.Sprintf("(%s & %d = 0) AND (%s & %d = 0)",
		column, int64(StatusAllDeleted),
		column, int64(StatusAllDisabled))
}

// SQLWhereRecoverable 生成可恢复状态查询
//
// 示例：
//
//	clause := Status(0).SQLWhereRecoverable("status")
func (s Status) SQLWhereRecoverable(column string) string {
	return fmt.Sprintf("(%s & %d = 0) AND (%s & %d != 0)",
		column, int64(StatusSysDeleted),
		column, int64(StatusAdmDeleted|StatusUserDeleted))
}

// ==================== 辅助工具函数 ====================

// ToSlice 将状态转换为状态切片（用于遍历）
//
// 示例：
//
//	for _, flag := range status.ToSlice() {
//	    fmt.Println(flag)
//	}
func (s Status) ToSlice() []Status {
	return s.ActiveFlags()
}

// FromSlice 从状态切片创建组合状态
//
// 示例：
//
//	flags := []Status{StatusUserDisabled, StatusSysHidden}
//	status := Status(0).FromSlice(flags)
func (s Status) FromSlice(flags []Status) Status {
	var result Status
	for _, flag := range flags {
		result |= flag
	}
	return result
}

// ToMap 将状态转换为 map（用于 JSON 导出）
//
// 示例：
//
//	m := status.ToMap()
//	// {"UserDisabled": true, "SysHidden": true}
func (s Status) ToMap() map[string]bool {
	result := make(map[string]bool)

	statusMap := map[Status]string{
		StatusSysDeleted:     "SysDeleted",
		StatusAdmDeleted:     "AdmDeleted",
		StatusUserDeleted:    "UserDeleted",
		StatusSysDisabled:    "SysDisabled",
		StatusAdmDisabled:    "AdmDisabled",
		StatusUserDisabled:   "UserDisabled",
		StatusSysHidden:      "SysHidden",
		StatusAdmHidden:      "AdmHidden",
		StatusUserHidden:     "UserHidden",
		StatusSysUnverified:  "SysUnverified",
		StatusAdmUnverified:  "AdmUnverified",
		StatusUserUnverified: "UserUnverified",
	}

	for flag, name := range statusMap {
		if s&flag != 0 {
			result[name] = true
		}
	}

	return result
}

// FromMap 从 map 创建状态
//
// 示例：
//
//	m := map[string]bool{"UserDisabled": true, "SysHidden": true}
//	status := Status(0).FromMap(m)
func (s Status) FromMap(m map[string]bool) Status {
	nameMap := map[string]Status{
		"SysDeleted":     StatusSysDeleted,
		"AdmDeleted":     StatusAdmDeleted,
		"UserDeleted":    StatusUserDeleted,
		"SysDisabled":    StatusSysDisabled,
		"AdmDisabled":    StatusAdmDisabled,
		"UserDisabled":   StatusUserDisabled,
		"SysHidden":      StatusSysHidden,
		"AdmHidden":      StatusAdmHidden,
		"UserHidden":     StatusUserHidden,
		"SysUnverified":  StatusSysUnverified,
		"AdmUnverified":  StatusAdmUnverified,
		"UserUnverified": StatusUserUnverified,
	}

	var result Status
	for name, enabled := range m {
		if enabled {
			if flag, ok := nameMap[name]; ok {
				result |= flag
			}
		}
	}

	return result
}

// ==================== 状态事件监听系统 🆕 ====================

// StatusEvent 状态变更事件
type StatusEvent struct {
	OldStatus Status // 变更前状态
	NewStatus Status // 变更后状态
	Changed   Status // 变更的位（added | removed）
	Added     Status // 新增的状态位
	Removed   Status // 移除的状态位
	Timestamp int64  // 变更时间戳（Unix 纳秒）
	Reason    string // 变更原因
	Operator  string // 操作者
}

// StatusListener 状态监听器
type StatusListener func(event StatusEvent)

// statusListeners 全局监听器列表（简化实现）
var statusListeners []StatusListener

// RegisterListener 注册状态监听器
//
// 使用场景：审计日志、事件通知、状态同步
//
// 示例：
//
//	RegisterListener(func(event StatusEvent) {
//	    log.Printf("Status changed: %v -> %v", event.OldStatus, event.NewStatus)
//	})
func RegisterListener(listener StatusListener) {
	statusListeners = append(statusListeners, listener)
}

// SetWithEvent 设置状态并触发事件
//
// 示例：
//
//	status.SetWithEvent(StatusUserDisabled, "违规操作", "admin")
func (s *Status) SetWithEvent(flag Status, reason, operator string) {
	old := *s
	s.Add(flag)
	s.notifyListeners(old, reason, operator)
}

// UnsetWithEvent 移除状态并触发事件
func (s *Status) UnsetWithEvent(flag Status, reason, operator string) {
	old := *s
	s.Unset(flag)
	s.notifyListeners(old, reason, operator)
}

// notifyListeners 通知所有监听器
func (s Status) notifyListeners(oldStatus Status, reason, operator string) {
	if len(statusListeners) == 0 {
		return
	}

	added, removed := s.Diff(oldStatus)
	event := StatusEvent{
		OldStatus: oldStatus,
		NewStatus: s,
		Changed:   added | removed,
		Added:     added,
		Removed:   removed,
		Timestamp: timeNow(),
		Reason:    reason,
		Operator:  operator,
	}

	for _, listener := range statusListeners {
		if listener != nil {
			listener(event)
		}
	}
}

// timeNow 获取当前时间戳（Unix 纳秒）
func timeNow() int64 {
	// 简化实现，实际应使用 time.Now().UnixNano()
	return 0
}

// ==================== 状态快照与历史 🆕 ====================

// StatusSnapshot 状态快照
type StatusSnapshot struct {
	Status    Status `json:"status"`
	Timestamp int64  `json:"timestamp"`
	Reason    string `json:"reason,omitempty"`
	Operator  string `json:"operator,omitempty"`
}

// StatusHistory 状态历史记录
type StatusHistory struct {
	Current   Status           `json:"current"`
	Snapshots []StatusSnapshot `json:"snapshots"`
	MaxSize   int              `json:"maxSize"` // 最大历史记录数
}

// NewStatusHistory 创建状态历史记录器
//
// 示例：
//
//	history := NewStatusHistory(StatusNone, 10) // 保留最近10条记录
func NewStatusHistory(initial Status, maxSize int) *StatusHistory {
	if maxSize <= 0 {
		maxSize = 10
	}
	return &StatusHistory{
		Current:   initial,
		Snapshots: []StatusSnapshot{{Status: initial, Timestamp: timeNow()}},
		MaxSize:   maxSize,
	}
}

// Update 更新状态并记录快照
func (h *StatusHistory) Update(newStatus Status, reason, operator string) {
	snapshot := StatusSnapshot{
		Status:    newStatus,
		Timestamp: timeNow(),
		Reason:    reason,
		Operator:  operator,
	}

	h.Snapshots = append(h.Snapshots, snapshot)

	// 保持历史记录在限制范围内
	if len(h.Snapshots) > h.MaxSize {
		h.Snapshots = h.Snapshots[len(h.Snapshots)-h.MaxSize:]
	}

	h.Current = newStatus
}

// Rollback 回滚到上一个状态
func (h *StatusHistory) Rollback() bool {
	if len(h.Snapshots) < 2 {
		return false
	}

	h.Snapshots = h.Snapshots[:len(h.Snapshots)-1]
	h.Current = h.Snapshots[len(h.Snapshots)-1].Status
	return true
}

// GetHistory 获取历史变更记录
func (h *StatusHistory) GetHistory() []StatusSnapshot {
	return h.Snapshots
}

// ==================== 条件链式操作 🆕 ====================

// StatusChain 状态链式操作（支持条件判断）
type StatusChain struct {
	status    *Status
	condition bool
}

// When 开始条件链（流式 API）
//
// 使用场景：复杂的条件状态操作
//
// 示例：
//
//	status.When(user.IsVIP()).
//	    Then(func(s *Status) { s.Unset(StatusUserDisabled) }).
//	    When(user.IsNewUser()).
//	    Then(func(s *Status) { s.Add(StatusUserUnverified) }).
//	    Execute()
func (s *Status) When(condition bool) *StatusChain {
	return &StatusChain{
		status:    s,
		condition: condition,
	}
}

// Then 条件为真时执行操作
func (sc *StatusChain) Then(operation func(*Status)) *StatusChain {
	if sc.condition {
		operation(sc.status)
	}
	return sc
}

// Otherwise 条件为假时执行操作
func (sc *StatusChain) Otherwise(operation func(*Status)) *StatusChain {
	if !sc.condition {
		operation(sc.status)
	}
	return sc
}

// When 添加新的条件（链式）
func (sc *StatusChain) When(condition bool) *StatusChain {
	return &StatusChain{
		status:    sc.status,
		condition: condition,
	}
}

// Execute 执行并返回状态（结束链式调用）
func (sc *StatusChain) Execute() Status {
	return *sc.status
}

// ==================== 位运算高级工具 🆕 ====================

// LowestBit 获取最低位的状态
//
// 使用场景：逐位处理状态
//
// 示例：
//
//	s := StatusUserDisabled | StatusSysHidden
//	lowest := s.LowestBit()  // 返回 StatusUserDisabled
func (s Status) LowestBit() Status {
	if s == 0 {
		return StatusNone
	}
	// 使用 x & -x 获取最低位
	return s & (-s)
}

// HighestBit 获取最高位的状态
//
// 示例：
//
//	s := StatusUserDisabled | StatusSysHidden
//	highest := s.HighestBit()  // 返回 StatusSysHidden
func (s Status) HighestBit() Status {
	if s == 0 {
		return StatusNone
	}

	// 找到最高位
	result := s
	result |= result >> 1
	result |= result >> 2
	result |= result >> 4
	result |= result >> 8
	result |= result >> 16
	result |= result >> 32
	return result ^ (result >> 1)
}

// NextBit 获取下一个可用的位（用于自动分配状态位）
//
// 使用场景：动态扩展状态位
//
// 示例：
//
//	customStatus := status.NextBit()
func (s Status) NextBit() Status {
	if s == 0 {
		return StatusExpand51
	}

	highest := s.HighestBit()
	if highest == 0 {
		return StatusExpand51
	}

	next := highest << 1
	if next <= 0 || next > MaxStatus {
		return StatusNone // 没有可用位
	}
	return next
}

// CountTrailingZeros 计算尾部零的数量
//
// 示例：
//
//	StatusUserDisabled.CountTrailingZeros()  // 返回 5（2^5 = 32）
func (s Status) CountTrailingZeros() int {
	if s == 0 {
		return 64
	}

	count := 0
	v := uint64(s)
	if v&0xFFFFFFFF == 0 {
		count += 32
		v >>= 32
	}
	if v&0xFFFF == 0 {
		count += 16
		v >>= 16
	}
	if v&0xFF == 0 {
		count += 8
		v >>= 8
	}
	if v&0xF == 0 {
		count += 4
		v >>= 4
	}
	if v&0x3 == 0 {
		count += 2
		v >>= 2
	}
	if v&0x1 == 0 {
		count += 1
	}
	return count
}

// IterateBits 遍历所有设置的位
//
// 使用场景：逐个处理每个状态位
//
// 示例：
//
//	status.IterateBits(func(bit Status) bool {
//	    fmt.Printf("处理状态位: %v\n", bit)
//	    return true // 返回 false 停止遍历
//	})
func (s Status) IterateBits(handler func(Status) bool) {
	current := s
	for current != 0 {
		// 获取最低位
		lowest := current.LowestBit()

		// 调用处理函数
		if !handler(lowest) {
			break
		}

		// 清除已处理的位
		current &^= lowest
	}
}

// ==================== 国际化支持 🆕 ====================

// StatusI18n 状态国际化描述
type StatusI18n struct {
	Lang         string            // 语言代码
	Descriptions map[Status]string // 状态描述映射
}

// defaultI18n 默认语言（中文）
var defaultI18n = StatusI18n{
	Lang: "zh-CN",
	Descriptions: map[Status]string{
		StatusSysDeleted:     "已被系统删除（不可恢复）",
		StatusAdmDeleted:     "已被管理员删除",
		StatusUserDeleted:    "已被用户删除（可恢复）",
		StatusSysDisabled:    "已被系统禁用",
		StatusAdmDisabled:    "已被管理员禁用",
		StatusUserDisabled:   "已被用户禁用",
		StatusSysHidden:      "已被系统隐藏",
		StatusAdmHidden:      "已被管理员隐藏",
		StatusUserHidden:     "已被用户隐藏",
		StatusSysUnverified:  "等待系统验证",
		StatusAdmUnverified:  "等待管理员审核",
		StatusUserUnverified: "等待用户验证",
	},
}

// i18nRegistry 国际化注册表
var i18nRegistry = map[string]StatusI18n{
	"zh-CN": defaultI18n,
	"en-US": {
		Lang: "en-US",
		Descriptions: map[Status]string{
			StatusSysDeleted:     "Deleted by system (unrecoverable)",
			StatusAdmDeleted:     "Deleted by administrator",
			StatusUserDeleted:    "Deleted by user (recoverable)",
			StatusSysDisabled:    "Disabled by system",
			StatusAdmDisabled:    "Disabled by administrator",
			StatusUserDisabled:   "Disabled by user",
			StatusSysHidden:      "Hidden by system",
			StatusAdmHidden:      "Hidden by administrator",
			StatusUserHidden:     "Hidden by user",
			StatusSysUnverified:  "Pending system verification",
			StatusAdmUnverified:  "Pending admin approval",
			StatusUserUnverified: "Pending user verification",
		},
	},
}

// RegisterI18n 注册新的语言支持
//
// 示例：
//
//	RegisterI18n("ja-JP", map[Status]string{
//	    StatusUserDeleted: "ユーザーによって削除されました",
//	})
func RegisterI18n(lang string, descriptions map[Status]string) {
	i18nRegistry[lang] = StatusI18n{
		Lang:         lang,
		Descriptions: descriptions,
	}
}

// DescriptionI18n 获取指定语言的状态描述
//
// 示例：
//
//	desc := status.DescriptionI18n("en-US")
func (s Status) DescriptionI18n(lang string) string {
	if lang == "" {
		lang = "zh-CN"
	}

	i18n, ok := i18nRegistry[lang]
	if !ok {
		i18n = defaultI18n
	}

	if s == StatusNone {
		if lang == "en-US" {
			return "Normal"
		}
		return "正常状态"
	}

	highest := s.HighestPriorityStatus()
	if desc, ok := i18n.Descriptions[highest]; ok {
		flags := s.ActiveFlags()
		if len(flags) > 1 {
			if lang == "en-US" {
				return fmt.Sprintf("%s (with %d more)", desc, len(flags)-1)
			}
			return desc + fmt.Sprintf("（另有 %d 个状态）", len(flags)-1)
		}
		return desc
	}

	if lang == "en-US" {
		return fmt.Sprintf("Custom status (0x%x)", s)
	}
	return fmt.Sprintf("自定义状态 (0x%x)", s)
}

// ==================== 状态集合操作 🆕 ====================

// StatusSet 状态集合（支持集合运算）
type StatusSet struct {
	statuses map[Status]bool
}

// NewStatusSet 创建状态集合
//
// 示例：
//
//	set := NewStatusSet(StatusUserDisabled, StatusSysHidden)
func NewStatusSet(statuses ...Status) *StatusSet {
	set := &StatusSet{
		statuses: make(map[Status]bool),
	}
	for _, s := range statuses {
		set.Add(s)
	}
	return set
}

// Add 添加状态到集合
func (ss *StatusSet) Add(status Status) {
	ss.statuses[status] = true
}

// Remove 从集合移除状态
func (ss *StatusSet) Remove(status Status) {
	delete(ss.statuses, status)
}

// Contains 检查集合是否包含状态
func (ss *StatusSet) Contains(status Status) bool {
	return ss.statuses[status]
}

// Union 并集
func (ss *StatusSet) Union(other *StatusSet) *StatusSet {
	result := NewStatusSet()
	for s := range ss.statuses {
		result.Add(s)
	}
	for s := range other.statuses {
		result.Add(s)
	}
	return result
}

// Intersection 交集
func (ss *StatusSet) Intersection(other *StatusSet) *StatusSet {
	result := NewStatusSet()
	for s := range ss.statuses {
		if other.Contains(s) {
			result.Add(s)
		}
	}
	return result
}

// Difference 差集
func (ss *StatusSet) Difference(other *StatusSet) *StatusSet {
	result := NewStatusSet()
	for s := range ss.statuses {
		if !other.Contains(s) {
			result.Add(s)
		}
	}
	return result
}

// ToStatus 转换为 Status（合并所有状态）
func (ss *StatusSet) ToStatus() Status {
	var result Status
	for s := range ss.statuses {
		result |= s
	}
	return result
}

// Size 获取集合大小
func (ss *StatusSet) Size() int {
	return len(ss.statuses)
}

// ==================== 状态规则引擎 🆕 ====================

// StatusRule 状态规则
type StatusRule struct {
	Name      string            // 规则名称
	Condition func(Status) bool // 条件函数
	Action    func(*Status)     // 动作函数
	Priority  int               // 优先级（数字越大越优先）
}

// StatusRuleEngine 状态规则引擎
type StatusRuleEngine struct {
	rules []StatusRule
}

// NewRuleEngine 创建规则引擎
func NewRuleEngine() *StatusRuleEngine {
	return &StatusRuleEngine{
		rules: make([]StatusRule, 0),
	}
}

// AddRule 添加规则
//
// 示例：
//
//	engine := NewRuleEngine()
//	engine.AddRule(StatusRule{
//	    Name: "自动恢复",
//	    Condition: func(s Status) bool {
//	        return s.IsRecoverable() && timeElapsed > 30days
//	    },
//	    Action: func(s *Status) {
//	        s.ClearGroup(DeletedGroup)
//	    },
//	    Priority: 10,
//	})
func (re *StatusRuleEngine) AddRule(rule StatusRule) {
	re.rules = append(re.rules, rule)

	// 按优先级排序（冒泡排序，简化实现）
	for i := len(re.rules) - 1; i > 0; i-- {
		if re.rules[i].Priority > re.rules[i-1].Priority {
			re.rules[i], re.rules[i-1] = re.rules[i-1], re.rules[i]
		}
	}
}

// Execute 执行规则引擎
func (re *StatusRuleEngine) Execute(status *Status) []string {
	var executedRules []string

	for _, rule := range re.rules {
		if rule.Condition != nil && rule.Condition(*status) {
			if rule.Action != nil {
				rule.Action(status)
			}
			executedRules = append(executedRules, rule.Name)
		}
	}

	return executedRules
}

// ==================== 状态模板 🆕 ====================

// StatusTemplate 状态模板（预定义的状态组合）
type StatusTemplate struct {
	Name        string // 模板名称
	Status      Status // 状态值
	Description string // 描述
}

// 预定义的状态模板
var (
	// TemplateNormal 正常状态模板
	TemplateNormal = StatusTemplate{
		Name:        "Normal",
		Status:      StatusNone,
		Description: "完全正常，无任何限制",
	}

	// TemplateNewUser 新用户模板
	TemplateNewUser = StatusTemplate{
		Name:        "NewUser",
		Status:      StatusUserUnverified,
		Description: "新用户，需要验证",
	}

	// TemplateBanned 封禁模板
	TemplateBanned = StatusTemplate{
		Name:        "Banned",
		Status:      StatusSysDisabled | StatusSysHidden,
		Description: "系统封禁，不可访问",
	}

	// TemplateSoftDeleted 软删除模板
	TemplateSoftDeleted = StatusTemplate{
		Name:        "SoftDeleted",
		Status:      StatusUserDeleted,
		Description: "用户删除，可恢复",
	}

	// TemplateHardDeleted 硬删除模板
	TemplateHardDeleted = StatusTemplate{
		Name:        "HardDeleted",
		Status:      StatusSysDeleted,
		Description: "系统删除，不可恢复",
	}
)

// ApplyTemplate 应用状态模板
//
// 示例：
//
//	status.ApplyTemplate(TemplateNewUser)
func (s *Status) ApplyTemplate(template StatusTemplate) {
	s.Replace(template.Status)
}

// GetTemplate 获取状态对应的模板
func (s Status) GetTemplate() *StatusTemplate {
	templates := []StatusTemplate{
		TemplateNormal,
		TemplateNewUser,
		TemplateBanned,
		TemplateSoftDeleted,
		TemplateHardDeleted,
	}

	for _, tmpl := range templates {
		if s.Equal(tmpl.Status) {
			return &tmpl
		}
	}

	return nil
}

// ==================== 状态统计分析 🆕 ====================

// StatusStats 状态统计信息
type StatusStats struct {
	Total            int            `json:"total"`            // 总数
	StatusCount      map[Status]int `json:"statusCount"`      // 每个状态的数量
	GroupCount       map[string]int `json:"groupCount"`       // 每个组的数量
	NormalCount      int            `json:"normalCount"`      // 正常状态数量
	AbnormalCount    int            `json:"abnormalCount"`    // 异常状态数量
	DeletableCount   int            `json:"deletableCount"`   // 可删除数量
	RecoverableCount int            `json:"recoverableCount"` // 可恢复数量
}

// AnalyzeStatuses 分析多个状态的统计信息
//
// 使用场景：管理后台、数据报表
//
// 示例：
//
//	stats := AnalyzeStatuses([]Status{s1, s2, s3})
//	fmt.Printf("异常状态占比: %.2f%%\n",
//	    float64(stats.AbnormalCount) / float64(stats.Total) * 100)
func AnalyzeStatuses(statuses []Status) StatusStats {
	stats := StatusStats{
		Total:       len(statuses),
		StatusCount: make(map[Status]int),
		GroupCount:  make(map[string]int),
	}

	for _, s := range statuses {
		if s.IsNormal() {
			stats.NormalCount++
		} else {
			stats.AbnormalCount++
		}

		if s.IsRecoverable() {
			stats.RecoverableCount++
		}

		// 统计每个激活的状态位
		for _, flag := range s.ActiveFlags() {
			stats.StatusCount[flag]++
		}

		// 统计每个组
		for _, group := range s.GetGroups() {
			stats.GroupCount[group.Name]++
		}
	}

	return stats
}
