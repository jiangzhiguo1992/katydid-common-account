package snowflake

import (
	"fmt"
	"log"
	"sync"
	"time"

	"katydid-common-account/pkg/idgen/core"
)

// Generator Snowflake算法的ID生成器实现
type Generator struct {
	// ========== 核心状态 ==========
	lastTimestamp int64 // 上次生成ID的时间戳（毫秒）
	datacenterID  int64 // 数据中心ID（0-31）
	workerID      int64 // 工作机器ID（0-31）
	sequence      int64 // 当前毫秒内的序列号（0-4095）

	// ========== 配置和策略 ==========
	config *Config // 生成器配置（依赖倒置：依赖配置抽象）

	// ========== 性能优化 ==========
	precomputedPart int64 // 预计算的ID部分（datacenterID和workerID），避免重复计算

	// ========== 监控和工具 ==========
	metrics   *Metrics         // 性能监控指标（可选，nil时不收集）
	validator core.IDValidator // ID验证器
	parser    core.IDParser    // ID解析器

	// ========== 并发控制 ==========
	mu sync.Mutex // 互斥锁，保护生成器状态
}

// New 创建一个新的Snowflake ID生成器
// 说明：使用最简配置创建生成器，默认关闭监控
func New(datacenterID, workerID int64) (core.Generator, error) {
	return NewWithConfig(&Config{
		DatacenterID:  datacenterID,
		WorkerID:      workerID,
		EnableMetrics: false, // 默认关闭监控以保持性能
	})
}

// NewWithConfig 使用配置创建Snowflake ID生成器
// 说明：完整配置方式，支持自定义时钟回拨策略和监控开关
func NewWithConfig(config *Config) (core.Generator, error) {
	if config == nil {
		return nil, core.ErrNilConfig
	}

	// 步骤1：验证配置
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// 步骤2：设置默认值
	config.SetDefaults()

	// 步骤3：预先计算datacenterID和workerID部分（性能优化）
	// 说明：这两部分在生成器生命周期内不变，预先计算避免每次生成ID时重复计算
	precomputedPart := (config.DatacenterID << DatacenterIDShift) | (config.WorkerID << WorkerIDShift)

	// 步骤4：初始化监控（如果启用）
	var metrics *Metrics
	if config.EnableMetrics {
		metrics = NewMetrics()
	}

	// 步骤5：创建生成器实例
	generator := &Generator{
		datacenterID:    config.DatacenterID,
		workerID:        config.WorkerID,
		lastTimestamp:   -1,             // 初始化为-1，表示尚未生成过ID
		sequence:        -1,             // 初始化为-1，首次生成时会递增为0
		config:          config.Clone(), // 使用配置副本（不可变性原则）
		precomputedPart: precomputedPart,
		metrics:         metrics,
		validator:       NewValidator(),
		parser:          NewParser(),
	}

	log.Println("Snowflake生成器创建成功",
		"datacenter_id", config.DatacenterID,
		"worker_id", config.WorkerID,
		"metrics_enabled", config.EnableMetrics)

	return generator, nil
}

// NextID 生成下一个唯一ID（线程安全）
// 实现core.IDGenerator接口
func (g *Generator) NextID() (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.nextIDUnsafe()
}

// NextIDBatch 批量生成ID（线程安全）
// 实现core.BatchGenerator接口
func (g *Generator) NextIDBatch(n int) ([]int64, error) {
	// 参数验证
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

// GetWorkerID 获取工作机器ID
// 实现core.ConfigurableGenerator接口
func (g *Generator) GetWorkerID() int64 {
	return g.workerID
}

// GetDatacenterID 获取数据中心ID
// 实现core.ConfigurableGenerator接口
func (g *Generator) GetDatacenterID() int64 {
	return g.datacenterID
}

// GetMetrics 获取性能监控指标
// 实现core.MonitorableGenerator接口
func (g *Generator) GetMetrics() map[string]uint64 {
	if g.metrics == nil {
		return map[string]uint64{"metrics_enabled": 0}
	}
	return g.metrics.ToMap()
}

// ResetMetrics 重置性能监控指标
// 实现core.MonitorableGenerator接口
func (g *Generator) ResetMetrics() {
	if g.metrics != nil {
		g.metrics.Reset()
	}
}

// GetIDCount 获取已生成的ID总数
// 实现core.MonitorableGenerator接口
func (g *Generator) GetIDCount() uint64 {
	if g.metrics == nil {
		return 0
	}
	return g.metrics.IDCount.Load()
}

// ParseID 解析ID
// 实现core.ParseableGenerator接口
func (g *Generator) ParseID(id int64) (*core.IDInfo, error) {
	return g.parser.Parse(id)
}

// ValidateID 验证ID
// 实现core.ParseableGenerator接口
func (g *Generator) ValidateID(id int64) error {
	return g.validator.Validate(id)
}

// nextIDUnsafe 内部使用的不加锁版本的ID生成方法
// 说明：调用者必须已持有锁
func (g *Generator) nextIDUnsafe() (int64, error) {
	// 步骤1：获取当前时间戳（毫秒）
	timestamp := time.Now().UnixNano() / 1e6

	// 步骤2：时钟回拨检测与处理
	if timestamp < g.lastTimestamp {
		if err := g.handleClockBackward(timestamp); err != nil {
			log.Println("时钟回拨，ID生成失败",
				"current_timestamp", timestamp,
				"last_timestamp", g.lastTimestamp,
				"error", err)
			return 0, err
		}
		// 重新获取时间戳（可能已经等待或使用了上次时间戳）
		timestamp = time.Now().UnixNano() / 1e6
	}

	// 步骤3：序列号管理
	if timestamp == g.lastTimestamp {
		// 同一毫秒内，检查序列号是否溢出
		if g.sequence >= MaxSequence {
			// 序列号已达上限（4095），需要等待下一毫秒
			if g.metrics != nil {
				g.metrics.SequenceOverflow.Add(1)
				g.metrics.WaitCount.Add(1)
			}
			startTime := time.Now()
			timestamp = g.waitNextMillis(g.lastTimestamp)
			if g.metrics != nil {
				g.metrics.TotalWaitTimeNs.Add(uint64(time.Since(startTime).Nanoseconds()))
			}
			// 重置序列号为-1，后面会递增为0
			g.sequence = -1
			g.lastTimestamp = timestamp
		}
		// 序列号递增（溢出后也会执行到这里）
		g.sequence++
	} else {
		// 新的毫秒，序列号重置为0
		g.sequence = 0
		g.lastTimestamp = timestamp
	}

	// 步骤4：组装ID
	// ID结构：时间戳(41位) | 数据中心ID(5位) | 工作机器ID(5位) | 序列号(12位)
	timeDiff := timestamp - Epoch
	id := (timeDiff << TimestampShift) | g.precomputedPart | g.sequence

	// 步骤5：更新监控指标
	if g.metrics != nil {
		g.metrics.IDCount.Add(1)
	}

	return id, nil
}

// nextIDBatchUnsafe 内部使用的不加锁版本的批量生成方法
// 说明：调用者必须已持有锁
func (g *Generator) nextIDBatchUnsafe(n int) ([]int64, error) {
	ids := make([]int64, 0, n)
	remainingIDs := n

	for remainingIDs > 0 {
		// 步骤1：获取当前时间戳
		timestamp := time.Now().UnixNano() / 1e6

		// 步骤2：时钟回拨检测
		if timestamp < g.lastTimestamp {
			if err := g.handleClockBackward(timestamp); err != nil {
				// 返回已生成的ID和错误
				log.Println("批量生成ID时遇到时钟回拨",
					"generated", len(ids),
					"requested", n,
					"error", err)
				return ids, fmt.Errorf("%w (generated %d/%d IDs)", err, len(ids), n)
			}
			timestamp = time.Now().UnixNano() / 1e6
		}

		// 步骤3：计算当前毫秒可用的ID数量
		var availableInCurrentMs int
		if timestamp == g.lastTimestamp {
			// 同一毫秒内，计算剩余可用序列号数量
			// g.sequence 是当前已使用的最后一个序列号（范围0-4095）
			// 剩余可用数量 = MaxSequence - g.sequence
			// 例如：g.sequence=9，则剩余4095-9=4086个
			availableInCurrentMs = int(MaxSequence - g.sequence)
			if availableInCurrentMs <= 0 {
				// 序列号已耗尽，等待下一毫秒
				if g.metrics != nil {
					g.metrics.SequenceOverflow.Add(1)
					g.metrics.WaitCount.Add(1)
				}
				startTime := time.Now()
				timestamp = g.waitNextMillis(g.lastTimestamp)
				if g.metrics != nil {
					g.metrics.TotalWaitTimeNs.Add(uint64(time.Since(startTime).Nanoseconds()))
				}
				// 新毫秒，重置为-1，后续会从0开始
				g.sequence = -1
				g.lastTimestamp = timestamp
				availableInCurrentMs = MaxSequence + 1 // 0-4095，共4096个
			}
		} else {
			// 新的毫秒，有完整的4096个序列号（0-4095）
			g.sequence = -1
			g.lastTimestamp = timestamp
			availableInCurrentMs = MaxSequence + 1 // 0-4095，共4096个
		}

		// 步骤4：确定本轮生成数量
		batchSize := remainingIDs
		if batchSize > availableInCurrentMs {
			batchSize = availableInCurrentMs
		}

		// 步骤5：批量生成ID
		timeDiff := timestamp - Epoch
		baseID := (timeDiff << TimestampShift) | g.precomputedPart

		// 批量生成：每次递增序列号并组装ID
		for i := 0; i < batchSize; i++ {
			g.sequence++
			id := baseID | g.sequence
			ids = append(ids, id)
		}

		// 步骤6：更新剩余数量
		remainingIDs -= batchSize
	}

	// 更新监控指标
	if g.metrics != nil {
		g.metrics.IDCount.Add(uint64(n))
	}

	return ids, nil
}

// handleClockBackward 处理时钟回拨
func (g *Generator) handleClockBackward(currentTimestamp int64) error {
	// 计算回拨偏移量
	offset := g.lastTimestamp - currentTimestamp

	// 更新监控指标
	if g.metrics != nil {
		g.metrics.ClockBackward.Add(1)
	}

	// 根据策略处理
	switch g.config.ClockBackwardStrategy {
	case core.StrategyError:
		// 策略1：直接返回错误（默认）
		return fmt.Errorf("%w: detected backward drift of %d ms",
			core.ErrClockMovedBackwards, offset)

	case core.StrategyWait:
		// 策略2：等待时钟追上
		if offset <= g.config.ClockBackwardTolerance {
			// 回拨在容忍范围内，尝试等待
			for retries := 0; retries < maxWaitRetries; retries++ {
				time.Sleep(time.Duration(offset+1) * time.Millisecond)
				newTimestamp := time.Now().UnixNano() / 1e6
				if newTimestamp >= g.lastTimestamp {
					// 时钟已追上
					return nil
				}
				// 重新计算偏移量
				offset = g.lastTimestamp - newTimestamp
			}
			// 超过最大重试次数
			return fmt.Errorf("%w: backward drift persisted after %d retries",
				core.ErrClockMovedBackwards, maxWaitRetries)
		}
		// 回拨超过容忍范围
		return fmt.Errorf("%w: backward drift %d ms exceeds tolerance %d ms",
			core.ErrClockMovedBackwards, offset, g.config.ClockBackwardTolerance)

	case core.StrategyUseLastTimestamp:
		// 策略3：使用上次时间戳（风险较高，仅特殊场景）
		// 警告：此策略可能导致ID重复，生产环境慎用！
		log.Println("使用上次时间戳策略处理时钟回拨",
			"offset", offset,
			"warning", "可能存在ID重复风险")
		return nil

	default:
		// 未知策略
		return fmt.Errorf("%w: unknown clock backward strategy",
			core.ErrClockMovedBackwards)
	}
}

// waitNextMillis 等待直到获取到比lastTimestamp更大的时间戳
// 说明：当序列号耗尽时，需要等待下一毫秒
func (g *Generator) waitNextMillis(lastTimestamp int64) int64 {
	timestamp := time.Now().UnixNano() / 1e6
	for timestamp <= lastTimestamp {
		time.Sleep(sleepDuration) // 休眠100微秒，避免CPU空转
		timestamp = time.Now().UnixNano() / 1e6
	}
	return timestamp
}
