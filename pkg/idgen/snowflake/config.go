package snowflake

import (
	"fmt"

	"katydid-common-account/pkg/idgen/core"
)

// ============================================================================
// Snowflake 配置定义
// ============================================================================

// Config Snowflake生成器配置
type Config struct {
	// DatacenterID 数据中心ID
	// 范围：0-31（5位二进制）
	// 用途：标识不同的数据中心，避免跨数据中心ID冲突
	DatacenterID int64

	// WorkerID 工作机器ID
	// 范围：0-31（5位二进制）
	// 用途：标识同一数据中心内的不同机器，避免同数据中心内ID冲突
	WorkerID int64

	// ClockBackwardStrategy 时钟回拨处理策略
	// 可选值：
	//   - StrategyError: 直接返回错误（默认，最安全）
	//   - StrategyWait: 等待时钟追上（容忍短暂回拨）
	//   - StrategyUseLastTimestamp: 使用上次时间戳（最激进，慎用）
	//
	// 默认值：StrategyError
	ClockBackwardStrategy core.ClockBackwardStrategy

	// ClockBackwardTolerance 时钟回拨容忍时间（毫秒）
	// 说明：
	//   - 仅在策略为 StrategyWait 时生效
	//   - 回拨时间在容忍范围内时，生成器会等待时钟追上
	//   - 超过容忍范围仍会返回错误
	//
	// 范围：0-1000ms（防止无限等待）
	// 默认值：5ms
	ClockBackwardTolerance int64

	// EnableMetrics 是否启用性能监控
	// 说明：
	//   - true: 收集ID生成统计信息（如：生成数量、序列号溢出次数等）
	//   - false: 不收集监控数据，性能更优
	//
	// 默认值：false
	// 建议：生产环境根据需要开启，测试环境可关闭以提升性能
	EnableMetrics bool
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	// 验证数据中心ID
	if c.DatacenterID < 0 || c.DatacenterID > MaxDatacenterID {
		return fmt.Errorf("%w: got %d, valid range [0, %d]",
			core.ErrInvalidDatacenterID, c.DatacenterID, MaxDatacenterID)
	}

	// 验证工作机器ID
	if c.WorkerID < 0 || c.WorkerID > MaxWorkerID {
		return fmt.Errorf("%w: got %d, valid range [0, %d]",
			core.ErrInvalidWorkerID, c.WorkerID, MaxWorkerID)
	}

	// 验证时钟回拨容忍时间（不能为负数）
	if c.ClockBackwardTolerance < 0 {
		return fmt.Errorf("clock backward tolerance must be non-negative, got %d ms",
			c.ClockBackwardTolerance)
	}

	// 验证时钟回拨容忍时间（防止无限等待）
	if c.ClockBackwardTolerance > maxClockBackwardToleranceLimit {
		return fmt.Errorf("clock backward tolerance too large: max %d ms, got %d ms",
			maxClockBackwardToleranceLimit, c.ClockBackwardTolerance)
	}

	return nil
}

// SetDefaults 设置配置的默认值
func (c *Config) SetDefaults() {
	// 设置时钟回拨容忍时间的默认值
	// 条件：未设置（<0）或超出合法范围
	if c.ClockBackwardTolerance < 0 || c.ClockBackwardTolerance > maxClockBackwardToleranceLimit {
		c.ClockBackwardTolerance = maxClockBackwardTolerance
	}

	// 注意：ClockBackwardStrategy的零值是StrategyError，这是合理的默认值
	// 因此无需显式设置
}

// Clone 克隆配置对象
func (c *Config) Clone() *Config {
	// 创建新的配置对象，复制所有字段
	return &Config{
		DatacenterID:           c.DatacenterID,
		WorkerID:               c.WorkerID,
		ClockBackwardStrategy:  c.ClockBackwardStrategy,
		ClockBackwardTolerance: c.ClockBackwardTolerance,
		EnableMetrics:          c.EnableMetrics,
	}
}
