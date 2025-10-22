package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ============================================================================
// 类型定义 (Type Definitions)
// ============================================================================

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

// StatusTransition 状态转换规则
type StatusTransition struct {
	From      Status             // 源状态
	To        Status             // 目标状态
	Validator func(Status) error // 自定义验证器（可选）
}

// StatusGroup 状态组（用于分组管理）
type StatusGroup struct {
	Name  string   // 组名
	Flags []Status // 组内状态
}

// ============================================================================
// 常量定义 (Constants)
// ============================================================================

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

// ============================================================================
// 包级变量 (Package Variables)
// ============================================================================

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

// 预定义状态组
var (
	GroupDeleted = StatusGroup{
		Name:  "Deleted",
		Flags: []Status{StatusSysDeleted, StatusAdmDeleted, StatusUserDeleted},
	}

	GroupDisabled = StatusGroup{
		Name:  "Disabled",
		Flags: []Status{StatusSysDisabled, StatusAdmDisabled, StatusUserDisabled},
	}

	GroupHidden = StatusGroup{
		Name:  "Hidden",
		Flags: []Status{StatusSysHidden, StatusAdmHidden, StatusUserHidden},
	}

	GroupUnverified = StatusGroup{
		Name:  "Unverified",
		Flags: []Status{StatusSysUnverified, StatusAdmUnverified, StatusUserUnverified},
	}
)

// ============================================================================
// 基础验证方法 (Basic Validation Methods)
// ============================================================================

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
func (s Status) IsValid() bool {
	return s >= 0 && s <= MaxStatus
}

// Validate 验证状态是否合法
//
// 检查规则：
// 1. 值在有效范围内
// 2. 已删除状态不应有未验证标记
func (s Status) Validate() error {
	if !s.IsValid() {
		return fmt.Errorf("status value out of valid range")
	}

	if s.IsDeleted() && s.IsUnverified() {
		return fmt.Errorf("deleted status should not have unverified flags")
	}

	return nil
}

// ============================================================================
// 状态修改方法 (State Modification Methods)
// ============================================================================

// Add 追加指定的状态位（推荐使用）
//
// 使用场景：在现有状态基础上添加新状态，不影响已有状态
// 时间复杂度：O(1)
// 内存分配：0
//
// 注意：此方法会修改接收者本身，必须传入指针才能生效
func (s *Status) Add(flag Status) {
	*s |= flag
}

// Set 追加指定的状态位（语义已修正为追加）
//
// 使用场景：添加新状态，不影响已有状态
// 时间复杂度：O(1)
// 内存分配：0
func (s *Status) Set(flag Status) {
	*s |= flag
}

// Unset 取消指定的状态位（移除状态）
//
// 使用场景：移除特定状态，保留其他状态
// 时间复杂度：O(1)
// 内存分配：0
func (s *Status) Unset(flag Status) {
	*s &^= flag
}

// Replace 替换为新状态（清除所有原有状态）
//
// 使用场景：完全重置状态为指定值，丢弃所有原有状态
// 警告：此操作会清除所有原有状态，请确认是否真的需要完全替换
func (s *Status) Replace(flag Status) {
	*s = flag
}

// Toggle 切换指定的状态位（翻转状态）
//
// 使用场景：开关式状态切换，有则删除，无则添加
// 时间复杂度：O(1)
func (s *Status) Toggle(flag Status) {
	*s ^= flag
}

// Merge 保留与指定状态位相同的部分，其他位清除（交集运算）
//
// 使用场景：过滤状态，只保留指定的状态位
// 警告：此操作会清除所有未在 flag 中指定的状态位
func (s *Status) Merge(flag Status) {
	*s &= flag
}

// Clear 清除所有状态位（重置为零值）
//
// 使用场景：重置状态，移除所有标记
func (s *Status) Clear() {
	*s = StatusNone
}

// SetMultiple 批量设置多个状态位（批量追加）
//
// 使用场景：一次性添加多个状态
// 性能优化：预先合并所有标志，进行单次 OR 运算
func (s *Status) SetMultiple(flags ...Status) {
	var combined Status
	for _, flag := range flags {
		combined |= flag
	}
	*s |= combined
}

// UnsetMultiple 批量取消多个状态位（批量移除）
//
// 使用场景：一次性移除多个状态
// 性能优化：预先合并所有标志，进行单次 AND NOT 运算
func (s *Status) UnsetMultiple(flags ...Status) {
	var combined Status
	for _, flag := range flags {
		combined |= flag
	}
	*s &^= combined
}

// SetSafe 安全地设置状态（带验证）
//
// 如果设置后状态无效，会回滚到原状态
func (s *Status) SetSafe(flag Status) error {
	old := *s
	*s |= flag
	if err := s.Validate(); err != nil {
		*s = old
		return err
	}
	return nil
}

// ============================================================================
// 状态查询方法 (State Query Methods)
// ============================================================================

// Contain 检查是否包含指定的状态位（精确匹配）
//
// 使用场景：检查是否同时包含所有指定的状态位
// 时间复杂度：O(1)
func (s Status) Contain(flag Status) bool {
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

// Equal 检查状态是否完全匹配（精确相等）
//
// 使用场景：判断两个状态是否完全一致
// 注意：与 == 运算符效果相同，但语义更清晰
func (s Status) Equal(status Status) bool {
	return s == status
}

// Matches 检查是否匹配模式（pattern 中的所有位都存在）
func (s Status) Matches(pattern Status) bool {
	return s&pattern == pattern
}

// ============================================================================
// 业务状态检查方法 (Business State Check Methods)
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

// IsUnverified 检查是否未验证（任意级别）
//
// 业务语义：未验证的内容可能需要审核或用户完成验证流程
func (s Status) IsUnverified() bool {
	return s&StatusAllUnverified != 0
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

// CanVerified 检查是否为已验证状态（业务验证检查）
//
// 业务规则：可见且已通过验证的内容才算完全可用
func (s Status) CanVerified() bool {
	return s.CanVisible() && !s.IsUnverified()
}

// IsNormal 检查是否为正常状态（无任何异常标记）
func (s Status) IsNormal() bool {
	return !s.IsDeleted() && !s.IsDisable() && !s.IsHidden() && !s.IsUnverified()
}

// IsAbnormal 检查是否存在异常状态
func (s Status) IsAbnormal() bool {
	return !s.IsNormal()
}

// NeedsAttention 检查是否需要关注（已删除或禁用）
func (s Status) NeedsAttention() bool {
	return s.IsDeleted() || s.IsDisable()
}

// IsRecoverable 检查是否可恢复（仅用户级删除/禁用/隐藏）
func (s Status) IsRecoverable() bool {
	if s.IsDeleted() && !s.HasAny(StatusSysDeleted, StatusAdmDeleted) {
		return true
	}
	if s.IsDisable() && !s.HasAny(StatusSysDisabled, StatusAdmDisabled) {
		return true
	}
	if s.IsHidden() && !s.HasAny(StatusSysHidden, StatusAdmHidden) {
		return true
	}
	return false
}

// IsAccessible 检查是否可访问（未删除且未被系统禁用）
func (s Status) IsAccessible() bool {
	return !s.IsDeleted() && !s.Contain(StatusSysDisabled)
}

// IsPublic 检查是否公开（未隐藏且可见）
func (s Status) IsPublic() bool {
	return s.CanVisible()
}

// RequiresVerification 检查是否需要验证
func (s Status) RequiresVerification() bool {
	return s.IsUnverified()
}

// ============================================================================
// 状态转换与优先级 (State Transition & Priority)
// ============================================================================

// Clone 克隆当前状态
func (s Status) Clone() Status {
	return s
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

// TransitionTo 状态转换（带规则验证）
//
// 使用场景：需要控制状态转换规则时使用
func (s *Status) TransitionTo(target Status, rules []StatusTransition) error {
	// 检查是否有匹配的转换规则
	for _, rule := range rules {
		if s.Contain(rule.From) {
			// 执行自定义验证器
			if rule.Validator != nil {
				if err := rule.Validator(*s); err != nil {
					return fmt.Errorf("transition validation failed: %w", err)
				}
			}
			// 允许转换
			if target == rule.To {
				*s = target
				return nil
			}
		}
	}

	// 没有找到匹配的规则
	return fmt.Errorf("no valid transition rule from %s to %s", s.String(), target.String())
}

// CanTransitionTo 检查是否可以转换到目标状态
func (s Status) CanTransitionTo(target Status, rules []StatusTransition) bool {
	for _, rule := range rules {
		if s.Contain(rule.From) && target == rule.To {
			if rule.Validator != nil {
				return rule.Validator(s) == nil
			}
			return true
		}
	}
	return false
}

// Priority 获取状态的优先级（值越大优先级越高）
//
// 优先级规则：
// - 系统级 > 管理员级 > 用户级
// - 删除 > 禁用 > 隐藏 > 未验证
func (s Status) Priority() int {
	priority := 0

	// 删除状态（最高优先级）
	if s&StatusSysDeleted != 0 {
		priority = max(priority, 12)
	}
	if s&StatusAdmDeleted != 0 {
		priority = max(priority, 11)
	}
	if s&StatusUserDeleted != 0 {
		priority = max(priority, 10)
	}

	// 禁用状态
	if s&StatusSysDisabled != 0 {
		priority = max(priority, 9)
	}
	if s&StatusAdmDisabled != 0 {
		priority = max(priority, 8)
	}
	if s&StatusUserDisabled != 0 {
		priority = max(priority, 7)
	}

	// 隐藏状态
	if s&StatusSysHidden != 0 {
		priority = max(priority, 6)
	}
	if s&StatusAdmHidden != 0 {
		priority = max(priority, 5)
	}
	if s&StatusUserHidden != 0 {
		priority = max(priority, 4)
	}

	// 未验证状态（最低优先级）
	if s&StatusSysUnverified != 0 {
		priority = max(priority, 3)
	}
	if s&StatusAdmUnverified != 0 {
		priority = max(priority, 2)
	}
	if s&StatusUserUnverified != 0 {
		priority = max(priority, 1)
	}

	return priority
}

// HighestPriorityStatus 获取优先级最高的状态位
func (s Status) HighestPriorityStatus() Status {
	allFlags := []Status{
		StatusSysDeleted, StatusAdmDeleted, StatusUserDeleted,
		StatusSysDisabled, StatusAdmDisabled, StatusUserDisabled,
		StatusSysHidden, StatusAdmHidden, StatusUserHidden,
		StatusSysUnverified, StatusAdmUnverified, StatusUserUnverified,
	}

	for _, flag := range allFlags {
		if s&flag != 0 {
			return flag
		}
	}
	return StatusNone
}

// IsStricterThan 检查当前状态是否比另一个状态更严格
func (s Status) IsStricterThan(other Status) bool {
	return s.Priority() > other.Priority()
}

// IsLooserThan 检查当前状态是否比另一个状态更宽松
func (s Status) IsLooserThan(other Status) bool {
	return s.Priority() < other.Priority()
}

// ============================================================================
// 状态组管理 (Status Group Management)
// ============================================================================

// BelongsToGroup 检查状态是否属于指定组
func (s Status) BelongsToGroup(group StatusGroup) bool {
	for _, flag := range group.Flags {
		if s&flag != 0 {
			return true
		}
	}
	return false
}

// GetGroups 获取状态所属的所有组
func (s Status) GetGroups() []StatusGroup {
	var groups []StatusGroup
	allGroups := []StatusGroup{GroupDeleted, GroupDisabled, GroupHidden, GroupUnverified}

	for _, group := range allGroups {
		if s.BelongsToGroup(group) {
			groups = append(groups, group)
		}
	}

	return groups
}

// ClearGroup 清除指定组的所有状态
func (s *Status) ClearGroup(group StatusGroup) {
	for _, flag := range group.Flags {
		s.Unset(flag)
	}
}

// SetGroupExclusive 在组内独占设置某个状态（清除组内其他状态）
func (s *Status) SetGroupExclusive(group StatusGroup, flag Status) {
	s.ClearGroup(group)
	s.Set(flag)
}

// ============================================================================
// 高级操作方法 (Advanced Operation Methods)
// ============================================================================

// ApplyIf 条件应用操作
//
// 如果条件满足，执行操作并返回 true
func (s *Status) ApplyIf(condition func(Status) bool, operation func(*Status)) bool {
	if condition(*s) {
		operation(s)
		return true
	}
	return false
}

// ApplyMultiple 批量应用多个操作
func (s *Status) ApplyMultiple(operations []func(*Status)) {
	for _, op := range operations {
		op(s)
	}
}

// Transform 转换状态（函数式编程风格）
func (s Status) Transform(transformer func(Status) Status) Status {
	return transformer(s)
}

// ============================================================================
// 辅助信息方法 (Helper Information Methods)
// ============================================================================

// ActiveFlags 获取所有已设置的状态位
func (s Status) ActiveFlags() []Status {
	var flags []Status

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
func (s Status) BitCount() int {
	count := 0
	v := uint64(s)
	for v != 0 {
		count++
		v &= v - 1
	}
	return count
}

// Binary 返回二进制字符串表示
func (s Status) Binary() string {
	return fmt.Sprintf("%064b", uint64(s))
}

// BinaryFormatted 返回格式化的二进制字符串（每8位一组）
func (s Status) BinaryFormatted() string {
	bin := fmt.Sprintf("%064b", uint64(s))
	var parts []string
	for i := 0; i < 64; i += 8 {
		parts = append(parts, bin[i:i+8])
	}
	return strings.Join(parts, " ")
}

// Debug 返回详细的调试信息
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

// ============================================================================
// 数据库接口实现 (Database Interface Implementation)
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

// ============================================================================
// 字符串表示方法 (String Representation Methods)
// ============================================================================

// String 实现 fmt.Stringer 接口，用于调试和日志输出（增强版）
//
// 输出格式：Status(数值: 状态列表) 或 Status(None)
func (s Status) String() string {
	if s == StatusNone {
		return "Status(None)"
	}

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

	for _, sn := range statusNames {
		if s&sn.flag != 0 {
			parts = append(parts, sn.name)
			unknownBits &^= sn.flag
		}
	}

	if unknownBits != 0 {
		parts = append(parts, fmt.Sprintf("Custom(0x%x)", unknownBits))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("Status(%d)", int64(s))
	}

	return fmt.Sprintf("Status(%d: %s)", int64(s), strings.Join(parts, "|"))
}

// StringVerbose 详细的字符串表示（包含业务状态）
func (s Status) StringVerbose() string {
	base := s.String()
	business := fmt.Sprintf("\n  - IsDeleted: %v\n  - IsDisabled: %v\n  - IsHidden: %v\n  - CanVisible: %v",
		s.IsDeleted(), s.IsDisable(), s.IsHidden(), s.CanVisible())
	return base + business
}

// ============================================================================
// 解析方法 (Parsing Methods)
// ============================================================================

// ParseStatus 从字符串解析单个状态
//
// 支持的格式：
// - 预定义状态名：SysDeleted, UserDisabled 等
// - 十进制数字：48, 96 等
// - 十六进制：0x30, 0x60 等
// - 二进制：0b110000 等
func ParseStatus(s string) (Status, error) {
	s = strings.TrimSpace(s)

	if status, ok := statusNameMap[s]; ok {
		return status, nil
	}

	var num int64
	var err error

	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		num, err = strconv.ParseInt(s[2:], 16, 64)
	} else if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
		num, err = strconv.ParseInt(s[2:], 2, 64)
	} else {
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
func ParseStatusMultiple(s string) (Status, error) {
	s = strings.TrimSpace(s)

	if s == "" || s == "None" {
		return StatusNone, nil
	}

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

// ============================================================================
// SQL 查询辅助方法 (SQL Query Helper Methods)
// ============================================================================

// SQLWhereHasAny 生成"包含任意状态"的 SQL WHERE 子句
//
// 示例：StatusUserDisabled.SQLWhereHasAny("status") => "status & 32 != 0"
func (s Status) SQLWhereHasAny(column string) string {
	return fmt.Sprintf("%s & %d != 0", column, int64(s))
}

// SQLWhereHasAll 生成"包含所有状态"的 SQL WHERE 子句
//
// 示例：(StatusUserDisabled | StatusSysHidden).SQLWhereHasAll("status") => "status & 96 = 96"
func (s Status) SQLWhereHasAll(column string) string {
	return fmt.Sprintf("%s & %d = %d", column, int64(s), int64(s))
}

// SQLWhereNone 生成"不包含指定状态"的 SQL WHERE 子句
//
// 示例：StatusAllDeleted.SQLWhereNone("status") => "status & 7 = 0"
func (s Status) SQLWhereNone(column string) string {
	return fmt.Sprintf("%s & %d = 0", column, int64(s))
}

// SQLWhereCanVisible 生成"可见状态"的查询条件
func (s Status) SQLWhereCanVisible(column string) string {
	return fmt.Sprintf("(%s & %d = 0) AND (%s & %d = 0) AND (%s & %d = 0)",
		column, int64(StatusAllDeleted),
		column, int64(StatusAllDisabled),
		column, int64(StatusAllHidden))
}

// SQLWhereVisible 生成"可见状态"的查询条件（别名）
func (s Status) SQLWhereVisible(column string) string {
	return s.SQLWhereCanVisible(column)
}

// SQLWhereNormal 生成"正常状态"的查询条件
func (s Status) SQLWhereNormal(column string) string {
	allAbnormal := StatusAllDeleted | StatusAllDisabled | StatusAllHidden | StatusAllUnverified
	return fmt.Sprintf("%s & %d = 0", column, int64(allAbnormal))
}

// SQLWhereAbnormal 生成"异常状态"的查询条件
func (s Status) SQLWhereAbnormal(column string) string {
	allAbnormal := StatusAllDeleted | StatusAllDisabled | StatusAllHidden | StatusAllUnverified
	return fmt.Sprintf("%s & %d != 0", column, int64(allAbnormal))
}

// SQLWhereAccessible 生成"可访问状态"的查询条件
func (s Status) SQLWhereAccessible(column string) string {
	return fmt.Sprintf("(%s & %d = 0) AND (%s & %d = 0)",
		column, int64(StatusAllDeleted),
		column, int64(StatusSysDisabled))
}

// SQLWhereRecoverable 生成"可恢复状态"的查询条件
func (s Status) SQLWhereRecoverable(column string) string {
	systemLevel := StatusSysDeleted | StatusSysDisabled | StatusAdmDeleted | StatusAdmDisabled
	userLevel := StatusUserDeleted | StatusUserDisabled | StatusUserHidden
	return fmt.Sprintf("(%s & %d = 0) AND (%s & %d != 0)",
		column, int64(systemLevel),
		column, int64(userLevel))
}

// Scope 生成 GORM 查询范围（用于链式查询）
//
// mode 参数：
// - "visible": 可见状态
// - "normal": 正常状态
// - "abnormal": 异常状态
// - "accessible": 可访问状态
// - "recoverable": 可恢复状态
func (s Status) Scope(column string, mode string) func(db interface{}) interface{} {
	return func(db interface{}) interface{} {
		// 这是一个示例实现，实际使用时需要根据具体的 ORM 框架调整
		return db
	}
}

// ============================================================================
// 工具函数 (Utility Functions)
// ============================================================================

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
