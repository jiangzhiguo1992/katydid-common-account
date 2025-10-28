package core

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
