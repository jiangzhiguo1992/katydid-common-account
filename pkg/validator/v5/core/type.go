package core

import "reflect"

const (
	// ErrorMessageEstimatedLength 预估的错误消息平均长度，用于优化字符串构建时的内存分配
	ErrorMessageEstimatedLength = 80
)

// 验证相关的元数据键
const (
	MetadataKeyValidateFields = "validate_fields" // 指定字段验证的元数据键
	MetadataKeyExcludeFields  = "exclude_fields"  // 排除字段验证的元数据键
)

// StrategyType 验证策略类型枚举
type StrategyType int8

// 验证策略类型枚举值
const (
	StrategyTypeRule StrategyType = iota + 1
	StrategyTypeNested
	StrategyTypeBusiness
)

// FieldAccessor 字段访问器函数类型
// 通过索引访问字段，避免重复的 FieldByName 查找，性能提升 20-30%
type FieldAccessor func(v reflect.Value) reflect.Value

// TypeInfo 类型信息，缓存类型的验证能力信息
type TypeInfo struct {
	// Type 类型
	Type reflect.Type
	// IsRuleValidator 是否实现了 IRuleValidation
	IsRuleValidator bool
	// IsBusinessValidator 是否实现了 IBusinessValidation
	IsBusinessValidator bool
	// IsLifecycleHooks 是否实现了 ILifecycleHooks
	IsLifecycleHooks bool
	// Rules 缓存的规则（如果实现了 IRuleValidation）
	Rules map[Scene]map[string]string
	// Accessors 字段访问器缓存（优化：避免重复的 FieldByName 查找）
	Accessors map[string]FieldAccessor
}
