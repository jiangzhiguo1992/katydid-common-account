package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"unsafe"
)

// Status 基于位运算的状态管理类型
//
// 设计理念：
// - 使用 int64 类型，提供 63 位可用状态位（第 0 位到第 62 位）
// - 每个状态位独立存在，支持同时设置多个状态
// - 通过位运算实现高性能的状态操作和查询
//
// 性能特点：
// - 内存占用：固定 8 字节
// - 状态检查：单次位运算，零内存分配
// - JSON 序列化：直接转换为 int64，性能优于字符串
//
// 注意事项：
// - 避免使用负数作为状态值（会导致符号位冲突）
// - 自定义状态位应从 StatusExpand51 开始左移
// - 所有修改方法都需要指针接收者才能生效
type Status int64

// 预定义状态位常量
//
// 状态分层设计（优先级从高到低）：
// - Sys (System)：系统级，由系统自动管理
// - Adm (Admin)：管理员级，由管理员手动操作
// - User：用户级，由用户自主控制
//
// 状态分类：Deleted（删除）、Disabled（禁用）、Hidden（隐藏）、Review（审核）
const (
	StatusNone Status = 0 // 零值，表示无状态

	// 删除状态组（位 0-2）
	StatusSysDeleted  Status = 1 << 0 // 系统删除，通常不可恢复
	StatusAdmDeleted  Status = 1 << 1 // 管理员删除，可能支持恢复
	StatusUserDeleted Status = 1 << 2 // 用户删除，通常可恢复

	// 禁用状态组（位 3-5）
	StatusSysDisabled  Status = 1 << 3 // 系统检测异常后自动禁用
	StatusAdmDisabled  Status = 1 << 4 // 管理员手动禁用
	StatusUserDisabled Status = 1 << 5 // 用户主动禁用（如账号冻结）

	// 隐藏状态组（位 6-8）
	StatusSysHidden  Status = 1 << 6 // 系统根据规则自动隐藏
	StatusAdmHidden  Status = 1 << 7 // 管理员手动隐藏
	StatusUserHidden Status = 1 << 8 // 用户设置为私密/不公开

	// 审核状态组（位 9-11）
	StatusSysReview  Status = 1 << 9  // 等待系统自动审核
	StatusAdmReview  Status = 1 << 10 // 等待管理员审核
	StatusUserReview Status = 1 << 11 // 等待用户验证（如邮箱验证）

	// 扩展起始位（位 12 开始，预留 51 位用于业务自定义状态）
	StatusExpand51 Status = 1 << 12
)

// 预定义状态组合常量（避免重复位运算，提升性能）
const (
	// StatusAllDeleted 所有删除状态的组合
	StatusAllDeleted Status = StatusSysDeleted | StatusAdmDeleted | StatusUserDeleted

	// StatusAllDisabled 所有禁用状态的组合
	StatusAllDisabled Status = StatusSysDisabled | StatusAdmDisabled | StatusUserDisabled

	// StatusAllHidden 所有隐藏状态的组合
	StatusAllHidden Status = StatusSysHidden | StatusAdmHidden | StatusUserHidden

	// StatusAllReview 所有审核状态的组合
	StatusAllReview Status = StatusSysReview | StatusAdmReview | StatusUserReview
)

// 状态值边界常量
const (
	// maxValidBit 最大有效位索引（从 0 开始计数）
	maxValidBit = 62

	// MaxStatus 最大合法状态值
	MaxStatus Status = 1<<(maxValidBit+1) - 1
)

// 性能优化：popcount 查表法（8位查找表）
// 相比 Brian Kernighan 算法快 2-3 倍，相比循环法快 5-10 倍
var popcount8 = [256]uint8{
	0, 1, 1, 2, 1, 2, 2, 3, 1, 2, 2, 3, 2, 3, 3, 4,
	1, 2, 2, 3, 2, 3, 3, 4, 2, 3, 3, 4, 3, 4, 4, 5,
	1, 2, 2, 3, 2, 3, 3, 4, 2, 3, 3, 4, 3, 4, 4, 5,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	1, 2, 2, 3, 2, 3, 3, 4, 2, 3, 3, 4, 3, 4, 4, 5,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	3, 4, 4, 5, 4, 5, 5, 6, 4, 5, 5, 6, 5, 6, 6, 7,
	1, 2, 2, 3, 2, 3, 3, 4, 2, 3, 3, 4, 3, 4, 4, 5,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	3, 4, 4, 5, 4, 5, 5, 6, 4, 5, 5, 6, 5, 6, 6, 7,
	2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, 5, 4, 5, 5, 6,
	3, 4, 4, 5, 4, 5, 5, 6, 4, 5, 5, 6, 5, 6, 6, 7,
	3, 4, 4, 5, 4, 5, 5, 6, 4, 5, 5, 6, 5, 6, 6, 7,
	4, 5, 5, 6, 5, 6, 6, 7, 5, 6, 6, 7, 6, 7, 7, 8,
}

// ============================================================================
// 状态修改方法 - 零内存分配设计
// ============================================================================

// Set 设置为新状态（完全替换）
func (s *Status) Set(flag Status) {
	*s = flag
}

// Clear 清除所有状态位
func (s *Status) Clear() {
	*s = StatusNone
}

// Add 追加指定的状态位
//
//go:inline
func (s *Status) Add(flag Status) {
	*s |= flag
}

// AddMultiple 批量设置多个状态位
func (s *Status) AddMultiple(flags ...Status) {
	if len(flags) == 0 {
		return
	}

	var combined Status
	for _, flag := range flags {
		combined |= flag
	}
	*s |= combined
}

// Del 移除指定的状态位
//
//go:inline
func (s *Status) Del(flag Status) {
	*s &^= flag
}

// DelMultiple 批量取消多个状态位
func (s *Status) DelMultiple(flags ...Status) {
	if len(flags) == 0 {
		return
	}

	var combined Status
	for _, flag := range flags {
		combined |= flag
	}
	*s &^= combined
}

// And 保留与指定状态位相同的部分
//
//go:inline
func (s *Status) And(flag Status) {
	*s &= flag
}

// AndMultiple 批量保留指定状态位
func (s *Status) AndMultiple(flags ...Status) {
	if len(flags) == 0 {
		return
	}

	var combined Status
	for _, flag := range flags {
		combined |= flag
	}
	*s &= combined
}

// Toggle 切换指定的状态位
//
//go:inline
func (s *Status) Toggle(flag Status) {
	*s ^= flag
}

// ToggleMultiple 批量切换状态位
func (s *Status) ToggleMultiple(flags ...Status) {
	if len(flags) == 0 {
		return
	}

	var combined Status
	for _, flag := range flags {
		combined |= flag
	}
	*s ^= combined
}

// ============================================================================
// 状态查询方法
// ============================================================================

// Has 检查是否包含指定的状态位
//
//go:inline
func (s Status) Has(flag Status) bool {
	return s&flag == flag && flag != 0
}

// HasAny 检查是否包含任意状态位
//
//go:inline
func (s Status) HasAny(flags ...Status) bool {
	if len(flags) == 0 {
		return false
	}

	var combined Status
	for _, flag := range flags {
		combined |= flag
	}
	return s&combined != 0
}

// HasAll 检查是否包含所有状态位
//
//go:inline
func (s Status) HasAll(flags ...Status) bool {
	if len(flags) == 0 {
		return true
	}

	var combined Status
	for _, flag := range flags {
		combined |= flag
	}
	return s&combined == combined
}

// ActiveFlags 获取所有已设置的状态位
func (s Status) ActiveFlags() []Status {
	if s == 0 {
		return nil
	}

	// 预分配切片容量
	bitCount := s.BitCount()
	flags := make([]Status, 0, bitCount)

	// 遍历所有可能的位
	//for i := 0; i <= maxValidBit; i++ {
	//	flag := Status(1 << i)
	//	if s&flag != 0 {
	//		flags = append(flags, flag)
	//		if len(flags) == bitCount {
	//			break // 早期退出
	//		}
	//	}
	//}

	// 使用 trailing zeros 算法，跳过未设置的位
	val := uint64(s)
	for val != 0 {
		// 找到最低位的 1
		bit := trailingZeros64(val)
		flags = append(flags, Status(1<<bit))
		// 清除最低位的 1
		val &= val - 1
	}

	return flags
}

// trailingZeros64 使用 De Bruijn 序列实现的 Trailing Zeros 算法
//
//go:nosplit
func trailingZeros64(x uint64) int {
	if x == 0 {
		return 64
	}
	// De Bruijn 乘法表
	const debruijn64 = 0x03f79d71b4ca8b09
	var deBruijnIdx64 = [64]byte{
		0, 1, 56, 2, 57, 49, 28, 3, 61, 58, 42, 50, 38, 29, 17, 4,
		62, 47, 59, 36, 45, 43, 51, 22, 53, 39, 33, 30, 24, 18, 12, 5,
		63, 55, 48, 27, 60, 41, 37, 16, 46, 35, 44, 21, 52, 32, 23, 11,
		54, 26, 40, 15, 34, 20, 31, 10, 25, 14, 19, 9, 13, 8, 7, 6,
	}
	return int(deBruijnIdx64[((x&-x)*debruijn64)>>58])
}

// Diff 比较两个状态的差异
func (s Status) Diff(other Status) (added Status, removed Status) {
	added = s &^ other   // 新增的状态位
	removed = other &^ s // 移除的状态位
	return
}

// ============================================================================
// 业务状态检查方法
// ============================================================================

// IsDeleted 检查是否被标记为删除（任意级别）
//
//go:inline
func (s Status) IsDeleted() bool {
	return s&StatusAllDeleted != 0
}

// IsDisable 检查是否被禁用（任意级别）
//
//go:inline
func (s Status) IsDisable() bool {
	return s&StatusAllDisabled != 0
}

// IsHidden 检查是否被隐藏（任意级别）
//
//go:inline
func (s Status) IsHidden() bool {
	return s&StatusAllHidden != 0
}

// IsReview 检查是否在审核中（任意级别）
//
//go:inline
func (s Status) IsReview() bool {
	return s&StatusAllReview != 0
}

// CanEnable 检查是否为可启用状态（未删除且未禁用）
//
//go:inline
func (s Status) CanEnable() bool {
	return s&(StatusAllDeleted|StatusAllDisabled) == 0
}

// CanVisible 检查是否为可见状态（未删除、未禁用且未隐藏）
//
//go:inline
func (s Status) CanVisible() bool {
	return s&(StatusAllDeleted|StatusAllDisabled|StatusAllHidden) == 0
}

// CanActive 检查是否为完全激活状态（未删除、未禁用、未隐藏且已通过审核）
//
//go:inline
func (s Status) CanActive() bool {
	return s&(StatusAllDeleted|StatusAllDisabled|StatusAllHidden|StatusAllReview) == 0
}

// ============================================================================
// 格式化方法
// ============================================================================

// String 实现 fmt.Stringer 接口
//
// 返回格式：Status(值)[位数 bits]
// 示例：Status(7)[3 bits] 表示状态值为 7，有 3 个状态位被设置
func (s Status) String() string {
	bitCount := s.BitCount()

	// 预估容量：Status( + 最多20位数字 + )[ + 最多2位数字 + bits]
	buf := make([]byte, 0, 32)

	buf = append(buf, "Status("...)
	buf = strconv.AppendInt(buf, int64(s), 10)
	buf = append(buf, ")["...)
	buf = strconv.AppendInt(buf, int64(bitCount), 10)
	buf = append(buf, " bits]"...)

	// unsafe 零拷贝转换（性能优化：避免 string(buf) 的内存拷贝）
	return *(*string)(unsafe.Pointer(&buf))
}

// BitCount 计算已设置的位数量（popcount）
//
// 使用查表法，将 int64 分成 8 个字节，每个字节查表后累加结果
//
//go:inline
func (s Status) BitCount() int {
	v := uint64(s)
	return int(
		popcount8[v&0xff] +
			popcount8[(v>>8)&0xff] +
			popcount8[(v>>16)&0xff] +
			popcount8[(v>>24)&0xff] +
			popcount8[(v>>32)&0xff] +
			popcount8[(v>>40)&0xff] +
			popcount8[(v>>48)&0xff] +
			popcount8[(v>>56)&0xff],
	)
}

// ============================================================================
// 数据库接口实现
// ============================================================================

// Value 实现 driver.Valuer 接口
//
//go:inline
func (s Status) Value() (driver.Value, error) {
	if s < 0 {
		return nil, fmt.Errorf("invalid Status value: negative number %d is not allowed", s)
	}
	if s > MaxStatus {
		return nil, fmt.Errorf("invalid Status value: %d exceeds maximum allowed value %d", s, MaxStatus)
	}
	return int64(s), nil
}

// Scan 实现 sql.Scanner 接口
func (s *Status) Scan(value interface{}) error {
	if value == nil {
		*s = StatusNone
		return nil
	}

	switch v := value.(type) {
	case int64:
		return s.setFromInt64(v)
	case int:
		return s.setFromInt64(int64(v))
	case uint64:
		if v > uint64(MaxStatus) {
			return fmt.Errorf("invalid Status value: %d exceeds maximum allowed value %d", v, MaxStatus)
		}
		*s = Status(v)
		return nil
	case []byte:
		// 数据库返回的 JSON 字节
		var num int64
		if err := json.Unmarshal(v, &num); err != nil {
			return fmt.Errorf("failed to unmarshal Status from bytes: %w", err)
		}
		return s.setFromInt64(num)
	default:
		return fmt.Errorf("cannot scan type %T into Status", value)
	}
}

//go:inline
func (s *Status) setFromInt64(v int64) error {
	if v < 0 {
		return fmt.Errorf("invalid Status value: negative number %d is not allowed", v)
	}
	if v > int64(MaxStatus) {
		return fmt.Errorf("invalid Status value: %d exceeds maximum allowed value %d", v, MaxStatus)
	}
	*s = Status(v)
	return nil
}

// ============================================================================
// JSON 接口实现
// ============================================================================

// MarshalJSON 实现 json.Marshaler 接口
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(s))
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (s *Status) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty JSON data")
	}

	// 快速路径：处理 JSON null
	if len(data) == 4 && data[0] == 'n' && data[1] == 'u' && data[2] == 'l' && data[3] == 'l' {
		*s = StatusNone
		return nil
	}

	var num int64
	if err := json.Unmarshal(data, &num); err != nil {
		return fmt.Errorf("failed to unmarshal Status from JSON: %w", err)
	}

	return s.setFromInt64(num)
}
