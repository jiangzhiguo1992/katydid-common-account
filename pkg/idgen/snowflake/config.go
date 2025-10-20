package snowflake

import (
	"fmt"

	"katydid-common-account/pkg/idgen/core"
)

// Config Snowflake配置选项（单一职责：只负责配置数据）
type Config struct {
	DatacenterID           int64                      // 数据中心ID (0-31)
	WorkerID               int64                      // 工作机器ID (0-31)
	ClockBackwardStrategy  core.ClockBackwardStrategy // 时钟回拨处理策略（可选，默认StrategyError）
	ClockBackwardTolerance int64                      // 时钟回拨容忍时间（毫秒，可选，默认5ms）
	EnableMetrics          bool                       // 是否启用监控
}

// Validate 验证配置的有效性（单一职责：只负责配置验证）
func (c *Config) Validate() error {
	if c == nil {
		return core.ErrNilConfig
	}

	if c.DatacenterID < 0 || c.DatacenterID > MaxDatacenterID {
		return fmt.Errorf("%w: got %d", core.ErrInvalidDatacenterID, c.DatacenterID)
	}

	if c.WorkerID < 0 || c.WorkerID > MaxWorkerID {
		return fmt.Errorf("%w: got %d", core.ErrInvalidWorkerID, c.WorkerID)
	}

	return nil
}

// SetDefaults 设置默认值（单一职责：配置默认化）
func (c *Config) SetDefaults() {
	if c.ClockBackwardTolerance <= 0 {
		c.ClockBackwardTolerance = maxClockBackwardTolerance
	}
}

// Clone 克隆配置（不可变性：返回新对象而非修改原对象）
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	return &Config{
		DatacenterID:           c.DatacenterID,
		WorkerID:               c.WorkerID,
		ClockBackwardStrategy:  c.ClockBackwardStrategy,
		ClockBackwardTolerance: c.ClockBackwardTolerance,
		EnableMetrics:          c.EnableMetrics,
	}
}
