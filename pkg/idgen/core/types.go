package core

// GeneratorType 生成器类型枚举
// 用途：标识不同的ID生成算法类型
type GeneratorType string

const (
	// GeneratorTypeSnowflake Snowflake算法生成器（Twitter开源的分布式ID生成算法）
	// 特点：
	//   - 64位整数ID，包含时间戳、机器ID、序列号
	//   - 单机每毫秒可生成4096个唯一ID
	//   - 适用于分布式系统
	GeneratorTypeSnowflake GeneratorType = "snowflake"

	// GeneratorTypeUUID UUID生成器（预留，便于扩展）
	// 特点：
	//   - 128位全球唯一标识符
	//   - 无需中心化协调
	//   - 适用于对ID长度不敏感的场景
	GeneratorTypeUUID GeneratorType = "uuid"

	// GeneratorTypeCustom 自定义生成器（预留，便于扩展）
	// 用途：支持业务自定义的ID生成算法
	GeneratorTypeCustom GeneratorType = "custom"
)

// String 实现Stringer接口，便于日志打印和调试
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
// 背景：在分布式系统中，时钟可能因NTP同步、手动调整等原因向后回拨
// 影响：时钟回拨可能导致ID重复，需要特殊处理
type ClockBackwardStrategy int

const (
	// StrategyError 直接返回错误（默认策略，最安全）
	// 适用场景：
	//   - 对ID唯一性要求极高的场景
	//   - 可以接受短暂服务不可用的场景
	//
	// 优点：绝对保证ID不重复
	// 缺点：时钟回拨时会导致ID生成失败
	StrategyError ClockBackwardStrategy = iota

	// StrategyWait 等待时钟追上（容忍短暂回拨）
	// 适用场景：
	//   - 时钟回拨幅度较小（如：几毫秒到几十毫秒）
	//   - 可以接受短暂阻塞的场景
	//
	// 优点：自动恢复，无需人工干预
	// 缺点：等待期间会阻塞ID生成请求
	//
	// 注意事项：
	//   - 只容忍配置范围内的回拨（默认5ms）
	//   - 超过容忍范围仍会返回错误
	StrategyWait

	// StrategyUseLastTimestamp 使用上次时间戳（最激进，仅用于特殊场景）
	// 适用场景：
	//   - 对可用性要求极高，可接受极小概率ID重复的场景
	//   - 有其他机制保证唯一性的场景
	//
	// 优点：完全不影响ID生成
	// 缺点：可能导致ID重复（如果序列号也耗尽）
	//
	// 警告：此策略存在ID重复风险，生产环境慎用！
	StrategyUseLastTimestamp
)

// String 实现Stringer接口，便于日志打印和调试
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

// IsValid 验证策略是否有效
func (s ClockBackwardStrategy) IsValid() bool {
	return s >= StrategyError && s <= StrategyUseLastTimestamp
}
