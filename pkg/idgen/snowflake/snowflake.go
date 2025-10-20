package snowflake

import (
	"fmt"
	"sync"
	"time"

	"katydid-common-account/pkg/idgen/core"
)

// Generator Snowflake算法的ID生成器实现
// 实现了core.FullFeaturedGenerator接口（里氏替换原则）
type Generator struct {
	// 生成器核心状态
	lastTimestamp int64 // 上次生成ID的时间戳
	datacenterID  int64 // 数据中心ID（0-31）
	workerID      int64 // 工作机器ID（0-31）
	sequence      int64 // 当前毫秒内的序列号（0-4095）

	// 配置和策略
	config *Config // 配置（依赖倒置：依赖配置抽象）

	// 性能优化
	precomputedPart int64 // 预计算的ID部分（datacenterID和workerID）

	// 监控和工具
	metrics   *Metrics   // 性能监控指标（可选）
	validator *Validator // ID验证器
	parser    *Parser    // ID解析器

	// 并发控制
	mu sync.Mutex
}

// New 创建一个新的Snowflake ID生成器
func New(datacenterID, workerID int64) (*Generator, error) {
	return NewWithConfig(&Config{
		DatacenterID:  datacenterID,
		WorkerID:      workerID,
		EnableMetrics: false, // 默认关闭监控以保持性能
	})
}

// NewWithConfig 使用配置创建Snowflake ID生成器
func NewWithConfig(config *Config) (*Generator, error) {
	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// 设置默认值
	config.SetDefaults()

	// 预先计算datacenterID和workerID部分（性能优化）
	precomputedPart := (config.DatacenterID << DatacenterIDShift) | (config.WorkerID << WorkerIDShift)

	// 初始化监控（如果启用）
	var metrics *Metrics
	if config.EnableMetrics {
		metrics = NewMetrics()
	}

	return &Generator{
		datacenterID:    config.DatacenterID,
		workerID:        config.WorkerID,
		lastTimestamp:   -1,
		sequence:        -1,             // 初始化为-1，首次生成时会递增为0
		config:          config.Clone(), // 使用配置副本（不可变性）
		precomputedPart: precomputedPart,
		metrics:         metrics,
		validator:       NewValidator(),
		parser:          NewParser(),
	}, nil
}

// NextID 生成下一个唯一ID（实现core.IDGenerator接口）
func (g *Generator) NextID() (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.nextIDUnsafe()
}

// NextIDBatch 批量生成ID（实现core.BatchGenerator接口）
func (g *Generator) NextIDBatch(n int) ([]int64, error) {
	if n <= 0 {
		return nil, fmt.Errorf("%w: batch size must be positive, got %d",
			core.ErrInvalidBatchSize, n)
	}
	if n > maxBatchSize {
		return nil, fmt.Errorf("%w: batch size too large (max %d), got %d",
			core.ErrInvalidBatchSize, maxBatchSize, n)
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	return g.nextIDBatchUnsafe(n)
}

// GetWorkerID 获取工作机器ID（实现core.ConfigurableGenerator接口）
func (g *Generator) GetWorkerID() int64 {
	return g.workerID
}

// GetDatacenterID 获取数据中心ID（实现core.ConfigurableGenerator接口）
func (g *Generator) GetDatacenterID() int64 {
	return g.datacenterID
}

// GetMetrics 获取性能监控指标（实现core.MonitorableGenerator接口）
func (g *Generator) GetMetrics() map[string]uint64 {
	if g.metrics == nil {
		return map[string]uint64{"metrics_enabled": 0}
	}
	return g.metrics.ToMap()
}

// ResetMetrics 重置性能监控指标（实现core.MonitorableGenerator接口）
func (g *Generator) ResetMetrics() {
	if g.metrics != nil {
		g.metrics.Reset()
	}
}

// GetIDCount 获取已生成的ID总数（实现core.MonitorableGenerator接口）
func (g *Generator) GetIDCount() uint64 {
	if g.metrics == nil {
		return 0
	}
	return g.metrics.IDCount.Load()
}

// ParseID 解析ID（实现core.ParseableGenerator接口）
func (g *Generator) ParseID(id int64) (*core.IDInfo, error) {
	return g.parser.Parse(id)
}

// ValidateID 验证ID（实现core.ParseableGenerator接口）
func (g *Generator) ValidateID(id int64) error {
	return g.validator.Validate(id)
}

// nextIDUnsafe 内部使用的不加锁版本的ID生成方法
func (g *Generator) nextIDUnsafe() (int64, error) {
	timestamp := time.Now().UnixNano() / 1e6

	// 时钟回拨检测与处理
	if timestamp < g.lastTimestamp {
		if err := g.handleClockBackward(timestamp); err != nil {
			return 0, err
		}
		// 重新获取时间戳（可能已经等待或使用了上次时间戳）
		timestamp = time.Now().UnixNano() / 1e6
	}

	// 序列号管理
	if timestamp == g.lastTimestamp {
		// 同一毫秒内，检查序列号是否溢出
		if g.sequence >= MaxSequence {
			// 等待下一毫秒
			if g.metrics != nil {
				g.metrics.SequenceOverflow.Add(1)
				g.metrics.WaitCount.Add(1)
			}
			startTime := time.Now()
			timestamp = g.waitNextMillis(g.lastTimestamp)
			if g.metrics != nil {
				g.metrics.TotalWaitTimeNs.Add(uint64(time.Since(startTime).Nanoseconds()))
			}
			g.sequence = -1
			g.lastTimestamp = timestamp
		}
		g.sequence++
	} else {
		// 新的毫秒，序列号重置
		g.sequence = 0
		g.lastTimestamp = timestamp
	}

	// 组装ID
	timeDiff := timestamp - Epoch
	id := (timeDiff << TimestampShift) | g.precomputedPart | g.sequence

	// 更新监控指标
	if g.metrics != nil {
		g.metrics.IDCount.Add(1)
	}

	return id, nil
}

// nextIDBatchUnsafe 内部使用的不加锁版本的批量生成方法
func (g *Generator) nextIDBatchUnsafe(n int) ([]int64, error) {
	ids := make([]int64, 0, n)
	remainingIDs := n

	for remainingIDs > 0 {
		timestamp := time.Now().UnixNano() / 1e6

		// 时钟回拨检测
		if timestamp < g.lastTimestamp {
			if err := g.handleClockBackward(timestamp); err != nil {
				// 返回已生成的ID和错误
				return ids, fmt.Errorf("%w (generated %d/%d IDs)", err, len(ids), n)
			}
			timestamp = time.Now().UnixNano() / 1e6
		}

		// 计算当前毫秒可用的ID数量
		var availableInCurrentMs int
		if timestamp == g.lastTimestamp {
			availableInCurrentMs = int(MaxSequence - g.sequence)
			if availableInCurrentMs <= 0 {
				// 等待下一毫秒
				if g.metrics != nil {
					g.metrics.SequenceOverflow.Add(1)
					g.metrics.WaitCount.Add(1)
				}
				startTime := time.Now()
				timestamp = g.waitNextMillis(g.lastTimestamp)
				if g.metrics != nil {
					g.metrics.TotalWaitTimeNs.Add(uint64(time.Since(startTime).Nanoseconds()))
				}
				g.sequence = -1
				g.lastTimestamp = timestamp
				availableInCurrentMs = MaxSequence + 1
			}
		} else {
			g.sequence = -1
			g.lastTimestamp = timestamp
			availableInCurrentMs = MaxSequence + 1
		}

		// 本轮生成数量
		batchSize := remainingIDs
		if batchSize > availableInCurrentMs {
			batchSize = availableInCurrentMs
		}

		// 生成ID
		timeDiff := timestamp - Epoch
		baseID := (timeDiff << TimestampShift) | g.precomputedPart

		for i := 0; i < batchSize; i++ {
			g.sequence++
			id := baseID | g.sequence
			ids = append(ids, id)
		}

		remainingIDs -= batchSize
	}

	if g.metrics != nil {
		g.metrics.IDCount.Add(uint64(n))
	}

	return ids, nil
}

// handleClockBackward 处理时钟回拨（单一职责：时钟回拨处理逻辑）
func (g *Generator) handleClockBackward(currentTimestamp int64) error {
	offset := g.lastTimestamp - currentTimestamp

	if g.metrics != nil {
		g.metrics.ClockBackward.Add(1)
	}

	switch g.config.ClockBackwardStrategy {
	case core.StrategyError:
		return fmt.Errorf("%w: detected backward drift of %dms",
			core.ErrClockMovedBackwards, offset)

	case core.StrategyWait:
		if offset <= g.config.ClockBackwardTolerance {
			// 尝试等待时钟追赶
			for retries := 0; retries < maxWaitRetries; retries++ {
				time.Sleep(time.Duration(offset+1) * time.Millisecond)
				newTimestamp := time.Now().UnixNano() / 1e6
				if newTimestamp >= g.lastTimestamp {
					return nil
				}
				offset = g.lastTimestamp - newTimestamp
			}
			return fmt.Errorf("%w: backward drift persisted after %d retries",
				core.ErrClockMovedBackwards, maxWaitRetries)
		}
		return fmt.Errorf("%w: backward drift %dms exceeds tolerance %dms",
			core.ErrClockMovedBackwards, offset, g.config.ClockBackwardTolerance)

	case core.StrategyUseLastTimestamp:
		// 使用上次时间戳（风险较高，仅特殊场景）
		return nil

	default:
		return fmt.Errorf("%w: unknown strategy", core.ErrClockMovedBackwards)
	}
}

// waitNextMillis 等待直到获取到比lastTimestamp更大的时间戳
func (g *Generator) waitNextMillis(lastTimestamp int64) int64 {
	timestamp := time.Now().UnixNano() / 1e6
	for timestamp <= lastTimestamp {
		time.Sleep(sleepDuration)
		timestamp = time.Now().UnixNano() / 1e6
	}
	return timestamp
}
