package core

// GeneratorType 生成器类型枚举
type GeneratorType string

const (
	// GeneratorTypeSnowflake Snowflake算法生成器
	GeneratorTypeSnowflake GeneratorType = "snowflake"
	// GeneratorTypeUUID UUID生成器（预留，便于扩展）
	GeneratorTypeUUID GeneratorType = "uuid"
	// GeneratorTypeCustom 自定义生成器（预留，便于扩展）
	GeneratorTypeCustom GeneratorType = "custom"
)

// String 实现Stringer接口
func (t GeneratorType) String() string {
	return string(t)
}

// IsValid 验证生成器类型是否有效
func (t GeneratorType) IsValid() bool {
	switch t {
	case GeneratorTypeSnowflake, GeneratorTypeUUID, GeneratorTypeCustom:
		return true
	default:
		return false
	}
}

// ClockBackwardStrategy 时钟回拨处理策略
type ClockBackwardStrategy int

const (
	// StrategyError 直接返回错误（默认，最安全）
	StrategyError ClockBackwardStrategy = iota
	// StrategyWait 等待追上（容忍短暂回拨）
	StrategyWait
	// StrategyUseLastTimestamp 使用上次时间戳（最激进，仅用于特殊场景）
	StrategyUseLastTimestamp
)

// String 实现Stringer接口
func (s ClockBackwardStrategy) String() string {
	switch s {
	case StrategyError:
		return "Error"
	case StrategyWait:
		return "Wait"
	case StrategyUseLastTimestamp:
		return "UseLastTimestamp"
	default:
		return "Unknown"
	}
}
