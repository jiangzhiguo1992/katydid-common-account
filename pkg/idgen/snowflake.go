package idgen

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Epoch 起始时间戳 (2024-01-01 00:00:00 UTC)
	//Epoch int64 = 1767196800000 // 毫秒时间戳 (2026-01-01，预留未来使用)
	Epoch int64 = 1672502400000 // 毫秒时间戳 (2024-01-01，当前使用)

	// 位数分配
	WorkerIDBits     = 5  // 工作机器ID位数
	DatacenterIDBits = 5  // 数据中心ID位数
	SequenceBits     = 12 // 序列号位数

	// 最大值计算
	MaxWorkerID     = -1 ^ (-1 << WorkerIDBits)     // 31 (2^5 - 1)
	MaxDatacenterID = -1 ^ (-1 << DatacenterIDBits) // 31 (2^5 - 1)
	MaxSequence     = -1 ^ (-1 << SequenceBits)     // 4095 (2^12 - 1)

	// 位移量
	WorkerIDShift     = SequenceBits                                   // 12
	DatacenterIDShift = SequenceBits + WorkerIDBits                    // 17
	TimestampShift    = SequenceBits + WorkerIDBits + DatacenterIDBits // 22

	// 最大时间戳差值 (41位)
	maxTimestampDiff int64 = 1<<41 - 1

	// 等待下一毫秒时的休眠时间（微秒）
	sleepDuration = 100 * time.Microsecond

	// 时钟回拨最大容忍时间（毫秒）
	maxClockBackwardTolerance = 5

	// 批量生成最大数量
	maxBatchSize = 4096

	// 等待策略最大重试次数
	maxWaitRetries = 10
)

var (
	// ErrInvalidWorkerID 工作机器ID超出有效范围
	ErrInvalidWorkerID = errors.New("invalid worker id: must be between 0 and 31")

	// ErrInvalidDatacenterID 数据中心ID超出有效范围
	ErrInvalidDatacenterID = errors.New("invalid datacenter id: must be between 0 and 31")

	// ErrClockMovedBackwards 检测到时钟回拨
	ErrClockMovedBackwards = errors.New("clock moved backwards: refusing to generate id")

	// ErrInvalidSnowflakeID 无效的Snowflake ID
	ErrInvalidSnowflakeID = errors.New("invalid snowflake id: id must be positive")

	// ErrTimestampOverflow 时间戳溢出
	ErrTimestampOverflow = errors.New("timestamp overflow: exceeds maximum allowed value")

	// ErrInvalidBatchSize 批量生成数量无效
	ErrInvalidBatchSize = errors.New("invalid batch size")
)

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

// Metrics 性能监控指标
type Metrics struct {
	IDCount          atomic.Uint64 // 已生成ID总数
	SequenceOverflow atomic.Uint64 // 序列号溢出次数
	ClockBackward    atomic.Uint64 // 时钟回拨次数
	WaitCount        atomic.Uint64 // 等待下一毫秒次数
	TotalWaitTimeNs  atomic.Uint64 // 总等待时间（纳秒）
}

// IDGenerator 定义ID生成器接口（接口隔离原则）
// 该接口只包含ID生成的核心功能，遵循最小接口原则
type IDGenerator interface {
	// NextID 生成下一个唯一ID
	// 返回生成的ID和可能的错误
	NextID() (int64, error)
	// NextIDBatch 批量生成ID
	NextIDBatch(n int) ([]int64, error)
}

// Snowflake Snowflake算法的ID生成器实现
// 实现了IDGenerator和IDParser接口（里氏替换原则）
type Snowflake struct {
	// 使用互斥锁保护并发访问（线程安全）
	mu sync.Mutex

	// 上次生成ID的时间戳
	lastTimestamp int64

	// 数据中心ID（0-31）
	datacenterID int64

	// 工作机器ID（0-31）
	workerID int64

	// 当前毫秒内的序列号（0-4095）
	sequence int64

	// 时钟回拨处理策略
	clockBackwardStrategy ClockBackwardStrategy

	// 时钟回拨容忍时间（毫秒）
	clockBackwardTolerance int64

	// 监控开关
	enableMetrics bool

	// 性能监控指标
	metrics *Metrics

	// 预计算的ID部分
	// datacenterID和workerID部分可以预先计算，减少每次生成时的位运算
	precomputedPart int64
}

// SnowflakeConfig Snowflake配置选项
type SnowflakeConfig struct {
	DatacenterID           int64                 // 数据中心ID
	WorkerID               int64                 // 工作机器ID
	ClockBackwardStrategy  ClockBackwardStrategy // 时钟回拨处理策略（可选，默认StrategyError）
	ClockBackwardTolerance int64                 // 时钟回拨容忍时间（毫秒，可选，默认5ms）
	EnableMetrics          bool                  // 是否启用监控
}

// NewSnowflake 创建一个新的Snowflake ID生成器
//
// 参数:
//
//	datacenterID: 数据中心ID，取值范围 [0, 31]
//	workerID: 工作机器ID，取值范围 [0, 31]
//
// 返回:
//
//	*Snowflake: Snowflake ID生成器实例
//	error: 参数验证失败时返回错误
//
// 注意: 建议使用NewSnowflakeWithConfig以获得更好的可扩展性
func NewSnowflake(datacenterID, workerID int64) (*Snowflake, error) {
	return NewSnowflakeWithConfig(&SnowflakeConfig{
		DatacenterID:  datacenterID,
		WorkerID:      workerID,
		EnableMetrics: false, // 默认关闭监控以保持性能
	})
}

// NewSnowflakeWithConfig 使用配置创建Snowflake ID生成器（开放封闭原则）
// 通过配置对象扩展功能，无需修改原有代码
//
// 参数:
//
//	config: Snowflake配置选项
//
// 返回:
//
//	*Snowflake: Snowflake ID生成器实例
//	error: 参数验证失败时返回错误
func NewSnowflakeWithConfig(config *SnowflakeConfig) (*Snowflake, error) {
	// 参数验证（健壮性）
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	if config.DatacenterID < 0 || config.DatacenterID > MaxDatacenterID {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidDatacenterID, config.DatacenterID)
	}

	if config.WorkerID < 0 || config.WorkerID > MaxWorkerID {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidWorkerID, config.WorkerID)
	}

	// 默认时钟回拨策略
	clockBackwardStrategy := config.ClockBackwardStrategy
	// 如果未设置策略，默认使用 StrategyError（更安全）
	// 注意：0 值就是 StrategyError，所以这个检查是正确的

	// 默认时钟回拨容忍时间
	clockBackwardTolerance := config.ClockBackwardTolerance
	if clockBackwardTolerance <= 0 {
		clockBackwardTolerance = maxClockBackwardTolerance
	}

	// metrics 初始化
	var metrics *Metrics
	if config.EnableMetrics {
		metrics = &Metrics{}
	}

	// 预先计算datacenterID和workerID部分
	precomputedPart := (config.DatacenterID << DatacenterIDShift) | (config.WorkerID << WorkerIDShift)

	return &Snowflake{
		datacenterID:           config.DatacenterID,
		workerID:               config.WorkerID,
		lastTimestamp:          -1,
		sequence:               0,
		clockBackwardStrategy:  clockBackwardStrategy,
		clockBackwardTolerance: clockBackwardTolerance,
		enableMetrics:          config.EnableMetrics,
		metrics:                metrics,
		precomputedPart:        precomputedPart,
	}, nil
}

// NextID 生成下一个唯一ID
// 该方法是线程安全的，可以在多个goroutine中并发调用
//
// 返回:
//
//	int64: 生成的唯一ID（63位正整数）
//	error: 当检测到时钟回拨或时间戳溢出时返回错误
//
// 性能特性:
//   - 单个实例每毫秒最多生成4096个ID
//   - 使用互斥锁保证线程安全
//   - 序列号耗尽时自动等待下一毫秒
func (s *Snowflake) NextID() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.nextIDUnsafe()
}

// waitNextMillis 等待直到获取到比lastTimestamp更大的时间戳
// 使用短暂休眠代替忙等待，减少CPU占用
//
// 参数:
//
//	lastTimestamp: 上一次的时间戳
//
// 返回:
//
//	int64: 新的时间戳（保证大于lastTimestamp）
func (s *Snowflake) waitNextMillis(lastTimestamp int64) int64 {
	// 内联时间函数调用
	timestamp := time.Now().UnixNano() / 1e6
	for timestamp <= lastTimestamp {
		// 短暂休眠，避免CPU空转（只阻塞当前goroutine）
		time.Sleep(sleepDuration)
		timestamp = time.Now().UnixNano() / 1e6
	}
	return timestamp
}

// GetIDCount 获取已生成的ID总数（用于监控）
//
// 返回:
//
//	uint64: 已生成的ID数量，如果未启用监控则返回0
func (s *Snowflake) GetIDCount() uint64 {
	if !s.enableMetrics || s.metrics == nil {
		return 0
	}
	return s.metrics.IDCount.Load()
}

// GetWorkerID 获取工作机器ID
func (s *Snowflake) GetWorkerID() int64 {
	return s.workerID
}

// GetDatacenterID 获取数据中心ID
func (s *Snowflake) GetDatacenterID() int64 {
	return s.datacenterID
}

// ParseSnowflakeID 解析Snowflake ID，提取其中的时间戳、数据中心ID、工作机器ID和序列号
// 这是一个便捷的全局函数，无需创建Snowflake实例
//
// 参数:
//
//	id: 要解析的Snowflake ID
//
// 返回:
//
//	timestamp: 生成ID时的时间戳（毫秒，Unix时间）
//	datacenterID: 数据中心ID (0-31)
//	workerID: 工作机器ID (0-31)
//	sequence: 序列号 (0-4095)
func ParseSnowflakeID(id int64) (timestamp int64, datacenterID int64, workerID int64, sequence int64) {
	timestamp = (id >> TimestampShift) + Epoch
	datacenterID = (id >> DatacenterIDShift) & MaxDatacenterID
	workerID = (id >> WorkerIDShift) & MaxWorkerID
	sequence = id & MaxSequence
	return
}

// GetTimestamp 从Snowflake ID中提取时间戳并转换为time.Time类型
// 这是一个便捷的全局函数
//
// 参数:
//
//	id: Snowflake ID
//
// 返回:
//
//	time.Time: ID生成时的时间
func GetTimestamp(id int64) time.Time {
	timestamp := (id >> TimestampShift) + Epoch
	return time.UnixMilli(timestamp)
}

// ValidateSnowflakeID 验证Snowflake ID的有效性
// 检查ID是否符合格式要求和时间戳是否合理
//
// 参数:
//
//	id: 要验证的Snowflake ID
//
// 返回:
//
//	error: 验证失败时返回错误，成功返回nil
func ValidateSnowflakeID(id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidSnowflakeID)
	}

	// 提取时间戳
	timestamp := (id >> TimestampShift) + Epoch

	// 检查时间戳是否在合理范围内
	now := time.Now().UnixMilli()
	if timestamp < Epoch {
		return fmt.Errorf("%w: timestamp %d is before epoch %d",
			ErrInvalidSnowflakeID, timestamp, Epoch)
	}

	// 允许一定的时钟误差（例如5分钟）
	if timestamp > now+300000 {
		return fmt.Errorf("%w: timestamp %d is too far in the future",
			ErrInvalidSnowflakeID, timestamp)
	}

	return nil
}

// NextIDBatch 批量生成ID
// 通过一次加锁生成多个ID，减少锁竞争，提高批量生成场景的性能
//
// 参数:
//
//	n: 要生成的ID数量，必须在 [1, 4096] 范围内
//
// 返回:
//
//	[]int64: 生成的ID列表
//	error: 生成失败时返回错误
//
// 使用场景:
//   - 批量初始化数据
//   - 预先生成ID池
//   - 减少高并发场景下的锁竞争
func (s *Snowflake) NextIDBatch(n int) ([]int64, error) {
	if n <= 0 {
		return nil, fmt.Errorf("%w: batch size must be positive, got %d", ErrInvalidBatchSize, n)
	}
	if n > maxBatchSize {
		return nil, fmt.Errorf("%w: batch size too large (max %d), got %d", ErrInvalidBatchSize, maxBatchSize, n)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]int64, 0, n)

	// 批量生成优化 - 预先计算可能需要的毫秒数
	// 如果需要的ID数量超过单毫秒最大序列号，提前知道需要跨越多个毫秒
	remainingIDs := n

	for remainingIDs > 0 {
		// 内联时间获取
		timestamp := time.Now().UnixNano() / 1e6

		// 时钟回拨检测
		if timestamp < s.lastTimestamp {
			offset := s.lastTimestamp - timestamp
			if s.enableMetrics && s.metrics != nil {
				s.metrics.ClockBackward.Add(1)
			}

			switch s.clockBackwardStrategy {
			case StrategyError:
				// 返回已生成的ID和错误
				return ids, fmt.Errorf("%w: backward %dms (generated %d IDs)", ErrClockMovedBackwards, offset, len(ids))
			case StrategyWait:
				if offset <= s.clockBackwardTolerance {
					// 使用重试机制，类似 nextIDUnsafe
					retries := 0
					for retries < maxWaitRetries {
						time.Sleep(time.Duration(offset+1) * time.Millisecond)
						timestamp = time.Now().UnixNano() / 1e6
						if timestamp >= s.lastTimestamp {
							break
						}
						offset = s.lastTimestamp - timestamp
						retries++
					}
					if timestamp < s.lastTimestamp {
						return ids, fmt.Errorf("%w: backward %dms after %d retries (generated %d IDs)",
							ErrClockMovedBackwards, s.lastTimestamp-timestamp, retries, len(ids))
					}
				} else {
					return ids, fmt.Errorf("%w: backward %dms exceeds tolerance %dms (generated %d IDs)",
						ErrClockMovedBackwards, offset, s.clockBackwardTolerance, len(ids))
				}
			case StrategyUseLastTimestamp:
				timestamp = s.lastTimestamp
			}
		}

		// 批量生成ID前先检查时间戳是否有效
		timeDiff := timestamp - Epoch
		if timeDiff < 0 {
			return ids, fmt.Errorf("%w: timestamp %d is before epoch %d (generated %d IDs)",
				ErrTimestampOverflow, timestamp, Epoch, len(ids))
		}

		if timeDiff > maxTimestampDiff {
			return ids, fmt.Errorf("%w: timestamp difference %d exceeds maximum %d (generated %d IDs)",
				ErrTimestampOverflow, timeDiff, maxTimestampDiff, len(ids))
		}

		// 计算当前毫秒可以生成的ID数量
		var availableInCurrentMs int
		if timestamp == s.lastTimestamp {
			// 同一毫秒内，计算剩余可用序列号数量
			// s.sequence 是上次使用的序列号（0-4095）
			// 剩余可用：MaxSequence - s.sequence（例如：s.sequence=10，则剩余4095-10=4085个）
			availableInCurrentMs = int(MaxSequence - s.sequence)

			// 如果序列号已经用完（s.sequence == MaxSequence），需要等待下一毫秒
			if availableInCurrentMs <= 0 {
				if s.enableMetrics && s.metrics != nil {
					s.metrics.SequenceOverflow.Add(1)
					s.metrics.WaitCount.Add(1)
				}
				startTime := time.Now()
				timestamp = s.waitNextMillis(s.lastTimestamp)
				if s.enableMetrics && s.metrics != nil {
					s.metrics.TotalWaitTimeNs.Add(uint64(time.Since(startTime).Nanoseconds()))
				}

				// 等待下一毫秒后，重新检查时间戳是否有效
				timeDiff = timestamp - Epoch
				if timeDiff < 0 || timeDiff > maxTimestampDiff {
					return ids, fmt.Errorf("%w: timestamp after wait is invalid (generated %d IDs)",
						ErrTimestampOverflow, len(ids))
				}

				// 新毫秒，序列号重置，全部4096个序列号可用
				s.sequence = 0
				s.lastTimestamp = timestamp
				availableInCurrentMs = MaxSequence + 1 // 0-4095，共4096个
			}
		} else {
			// 新的毫秒，序列号重置，全部4096个序列号可用
			s.sequence = 0
			s.lastTimestamp = timestamp
			availableInCurrentMs = MaxSequence + 1 // 0-4095，共4096个
		}

		// 本轮生成数量
		batchSize := remainingIDs
		if batchSize > availableInCurrentMs {
			batchSize = availableInCurrentMs
		}

		// 使用预计算的部分（在循环外计算以提高性能）
		baseID := (timeDiff << TimestampShift) | s.precomputedPart

		// 从当前序列号开始生成
		for i := 0; i < batchSize; i++ {
			id := baseID | s.sequence

			// 最终安全检查：确保生成的ID为正数
			if id <= 0 {
				return ids, fmt.Errorf("%w: generated id is not positive: %d (after generating %d IDs)",
					ErrTimestampOverflow, id, len(ids))
			}
			ids = append(ids, id)

			// 递增序列号
			s.sequence++
		}

		// s.sequence 现在指向下一个待使用的序列号，需要减1得到最后使用的序列号
		s.sequence--
		remainingIDs -= batchSize
	}

	if s.enableMetrics && s.metrics != nil {
		s.metrics.IDCount.Add(uint64(n))
	}

	return ids, nil
}

// nextIDUnsafe 内部使用的不加锁版本的ID生成方法
// 注意: 调用此方法前必须先获取锁
//
// 返回:
//
//	int64: 生成的唯一ID
//	error: 生成失败时返回错误
func (s *Snowflake) nextIDUnsafe() (int64, error) {
	// 内联时间函数，减少函数调用开销
	timestamp := time.Now().UnixNano() / 1e6

	// 时钟回拨检测
	if timestamp < s.lastTimestamp {
		offset := s.lastTimestamp - timestamp
		if s.enableMetrics && s.metrics != nil {
			s.metrics.ClockBackward.Add(1)
		}

		// 根据策略处理时钟回拨
		switch s.clockBackwardStrategy {
		case StrategyError:
			return 0, fmt.Errorf("%w: backward %dms", ErrClockMovedBackwards, offset)

		case StrategyWait:
			if offset <= s.clockBackwardTolerance {
				// 使用重试机制等待时钟追赶
				retries := 0
				for retries < maxWaitRetries {
					// 使用 offset+1 确保时钟真正前进
					time.Sleep(time.Duration(offset+1) * time.Millisecond)
					timestamp = time.Now().UnixNano() / 1e6
					if timestamp >= s.lastTimestamp {
						break
					}
					// 重新计算偏移量
					offset = s.lastTimestamp - timestamp
					retries++
				}
				// 等待后仍然回拨，返回错误
				if timestamp < s.lastTimestamp {
					return 0, fmt.Errorf("%w: backward %dms after %d retries",
						ErrClockMovedBackwards, s.lastTimestamp-timestamp, retries)
				}
			} else {
				return 0, fmt.Errorf("%w: backward %dms exceeds tolerance %dms",
					ErrClockMovedBackwards, offset, s.clockBackwardTolerance)
			}

		case StrategyUseLastTimestamp:
			timestamp = s.lastTimestamp
		}
	}

	// 检查时间戳溢出（提前检查避免生成负数ID）
	timeDiff := timestamp - Epoch
	if timeDiff < 0 {
		return 0, fmt.Errorf("%w: current time %d is before epoch %d",
			ErrTimestampOverflow, timestamp, Epoch)
	}

	// 检查时间戳是否超过41位能表示的最大值
	if timeDiff > maxTimestampDiff {
		return 0, fmt.Errorf("%w: timestamp difference %d exceeds maximum %d",
			ErrTimestampOverflow, timeDiff, maxTimestampDiff)
	}

	// 同一毫秒内，序列号递增
	if timestamp == s.lastTimestamp {
		s.sequence++
		// 序列号溢出，等待下一毫秒
		if s.sequence > MaxSequence {
			if s.enableMetrics && s.metrics != nil {
				s.metrics.SequenceOverflow.Add(1)
			}
			startTime := time.Now()
			timestamp = s.waitNextMillis(s.lastTimestamp)
			if s.enableMetrics && s.metrics != nil {
				s.metrics.WaitCount.Add(1)
				s.metrics.TotalWaitTimeNs.Add(uint64(time.Since(startTime).Nanoseconds()))
			}

			// 等待后重新检查时间戳是否仍然有效
			newTimeDiff := timestamp - Epoch
			if newTimeDiff < 0 || newTimeDiff > maxTimestampDiff {
				return 0, fmt.Errorf("%w: timestamp after wait is invalid", ErrTimestampOverflow)
			}
			// 重置序列号
			s.sequence = 0
		}
	} else {
		// 不同毫秒，序列号重置为0
		s.sequence = 0
	}

	s.lastTimestamp = timestamp

	// 使用预计算的部分组装ID，减少位运算
	id := ((timeDiff) << TimestampShift) | s.precomputedPart | s.sequence

	// 最终安全检查：确保生成的ID为正数
	if id <= 0 {
		return 0, fmt.Errorf("%w: generated id is not positive: %d", ErrTimestampOverflow, id)
	}

	// 只在启用监控时才更新指标
	if s.enableMetrics && s.metrics != nil {
		s.metrics.IDCount.Add(1)
	}

	return id, nil
}

// GetMetrics 获取性能监控指标
//
// 返回:
//
//	map[string]uint64: 包含各项性能指标的映射
//
// 指标说明:
//   - id_count: 已生成的ID总数
//   - sequence_overflow: 序列号溢出次数（需要等待下一毫秒的次数）
//   - clock_backward: 检测到时钟回拨的次数
//   - wait_count: 等待下一毫秒的总次数
//   - avg_wait_time_ns: 平均等待时间（纳秒）
func (s *Snowflake) GetMetrics() map[string]uint64 {
	if !s.enableMetrics {
		return map[string]uint64{
			"metrics_enabled": 0,
		}
	}

	waitCount := s.metrics.WaitCount.Load()
	var avgWaitTime uint64
	if waitCount > 0 {
		avgWaitTime = s.metrics.TotalWaitTimeNs.Load() / waitCount
	}

	return map[string]uint64{
		"metrics_enabled":   1,
		"id_count":          s.metrics.IDCount.Load(),
		"sequence_overflow": s.metrics.SequenceOverflow.Load(),
		"clock_backward":    s.metrics.ClockBackward.Load(),
		"wait_count":        waitCount,
		"avg_wait_time_ns":  avgWaitTime,
	}
}

// GetMetricsSnapshot 获取性能指标的快照（结构体形式）
//
// 返回:
//
//	*Metrics: 性能指标快照（指针类型，避免复制锁）
func (s *Snowflake) GetMetricsSnapshot() *Metrics {
	if !s.enableMetrics || s.metrics == nil {
		return &Metrics{}
	}

	// 创建快照并复制当前值
	snapshot := &Metrics{}
	snapshot.IDCount.Store(s.metrics.IDCount.Load())
	snapshot.SequenceOverflow.Store(s.metrics.SequenceOverflow.Load())
	snapshot.ClockBackward.Store(s.metrics.ClockBackward.Load())
	snapshot.WaitCount.Store(s.metrics.WaitCount.Load())
	snapshot.TotalWaitTimeNs.Store(s.metrics.TotalWaitTimeNs.Load())
	return snapshot
}

// ResetMetrics 重置性能监控指标
// 注意: 此方法不是线程安全的，仅用于测试场景
func (s *Snowflake) ResetMetrics() {
	// 添加 nil 检查，防止 panic
	if !s.enableMetrics || s.metrics == nil {
		return
	}
	s.metrics.IDCount.Store(0)
	s.metrics.SequenceOverflow.Store(0)
	s.metrics.ClockBackward.Store(0)
	s.metrics.WaitCount.Store(0)
	s.metrics.TotalWaitTimeNs.Store(0)
}
