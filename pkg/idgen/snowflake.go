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

	// 最大值计算(切记不是个数)
	MaxWorkerID     = -1 ^ (-1 << WorkerIDBits)     // 31 (2^5 - 1) [0, 31]
	MaxDatacenterID = -1 ^ (-1 << DatacenterIDBits) // 31 (2^5 - 1) [0, 31]
	MaxSequence     = -1 ^ (-1 << SequenceBits)     // 4095 (2^12 - 1) [0, 4095]

	// 位移量
	WorkerIDShift     = SequenceBits                                   // 12
	DatacenterIDShift = SequenceBits + WorkerIDBits                    // 17
	TimestampShift    = SequenceBits + WorkerIDBits + DatacenterIDBits // 22

	// 最大时间戳差值 (41位)
	//maxTimestampDiff int64 = 1<<41 - 1

	// 等待下一毫秒时的休眠时间（微秒）
	sleepDuration = 100 * time.Microsecond

	// 时钟回拨最大容忍时间（毫秒）
	maxClockBackwardTolerance = 5

	// 批量生成最大数量（支持跨毫秒生成）
	maxBatchSize = 100_000

	// 等待策略最大重试次数
	maxWaitRetries = 10

	// 允许的未来时间容差（毫秒）
	maxFutureTimeTolerance = 5 * 60 * 1000
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
	//ErrTimestampOverflow = errors.New("timestamp overflow: exceeds maximum allowed value")

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

// IDGenerator 定义ID生成器接口
type IDGenerator interface {
	// NextID 生成下一个唯一ID
	NextID() (int64, error)
	// NextIDBatch 批量生成ID
	NextIDBatch(n int) ([]int64, error)
}

// Snowflake Snowflake算法的ID生成器实现
type Snowflake struct {
	lastTimestamp int64 // 上次生成ID的时间戳
	datacenterID  int64 // 数据中心ID（0-31）
	workerID      int64 // 工作机器ID（0-31）
	sequence      int64 // 当前毫秒内的序列号（0-4095）

	clockBackwardStrategy  ClockBackwardStrategy // 时钟回拨处理策略
	clockBackwardTolerance int64                 // 时钟回拨容忍时间（毫秒）

	enableMetrics bool     // 监控开关
	metrics       *Metrics // 性能监控指标

	precomputedPart int64 // 预计算的ID部分，（datacenterI+和workerID）

	mu sync.Mutex
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
func NewSnowflake(datacenterID, workerID int64) (*Snowflake, error) {
	return NewSnowflakeWithConfig(&SnowflakeConfig{
		DatacenterID:  datacenterID,
		WorkerID:      workerID,
		EnableMetrics: false, // 默认关闭监控以保持性能
	})
}

// NewSnowflakeWithConfig 使用配置创建Snowflake ID生成器
func NewSnowflakeWithConfig(config *SnowflakeConfig) (*Snowflake, error) {
	// 参数验证
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	if config.DatacenterID < 0 || config.DatacenterID > MaxDatacenterID {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidDatacenterID, config.DatacenterID)
	}

	if config.WorkerID < 0 || config.WorkerID > MaxWorkerID {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidWorkerID, config.WorkerID)
	}

	// 默认时钟回拨策略，如果未设置策略，默认使用 StrategyError（更安全）
	clockBackwardStrategy := config.ClockBackwardStrategy

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
		sequence:               -1, // 初始化为-1，首次生成时会递增为0
		clockBackwardStrategy:  clockBackwardStrategy,
		clockBackwardTolerance: clockBackwardTolerance,
		enableMetrics:          config.EnableMetrics,
		metrics:                metrics,
		precomputedPart:        precomputedPart,
	}, nil
}

// NextID 生成下一个唯一ID
func (s *Snowflake) NextID() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.nextIDUnsafe()
}

// waitNextMillis 等待直到获取到比lastTimestamp更大的时间戳
func (s *Snowflake) waitNextMillis(lastTimestamp int64) int64 {
	timestamp := time.Now().UnixNano() / 1e6
	for timestamp <= lastTimestamp {
		// 短暂休眠，避免CPU空转（只阻塞当前goroutine）
		time.Sleep(sleepDuration)
		timestamp = time.Now().UnixNano() / 1e6
	}
	return timestamp
}

// GetIDCount 获取已生成的ID总数（用于监控）
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

// ParseSnowflakeID 解析Snowflake ID
func ParseSnowflakeID(id int64) (timestamp int64, datacenterID int64, workerID int64, sequence int64) {
	timestamp = (id >> TimestampShift) + Epoch
	datacenterID = (id >> DatacenterIDShift) & MaxDatacenterID
	workerID = (id >> WorkerIDShift) & MaxWorkerID
	sequence = id & MaxSequence
	return
}

// GetTimestamp 从Snowflake ID中提取时间戳
func GetTimestamp(id int64) time.Time {
	timestamp := (id >> TimestampShift) + Epoch
	return time.UnixMilli(timestamp)
}

// ValidateSnowflakeID 验证Snowflake ID的有效性
func ValidateSnowflakeID(id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidSnowflakeID)
	}

	// 提取时间戳
	timestamp := (id >> TimestampShift) + Epoch

	// 检查时间戳是否在合理范围内
	if timestamp < Epoch {
		return fmt.Errorf("%w: timestamp %d is before epoch %d",
			ErrInvalidSnowflakeID, timestamp, Epoch)
	}

	// 允许一定的时钟误差，防止恶意构造ID
	now := time.Now().UnixMilli()
	if timestamp > now+maxFutureTimeTolerance {
		return fmt.Errorf("%w: timestamp %d is too far in the future (max tolerance %dms)",
			ErrInvalidSnowflakeID, timestamp, maxFutureTimeTolerance)
	}

	return nil
}

// NextIDBatch 批量生成ID
func (s *Snowflake) NextIDBatch(n int) ([]int64, error) {
	if n <= 0 {
		return nil, fmt.Errorf("%w: batch size must be positive, got %d", ErrInvalidBatchSize, n)
	}
	if n > maxBatchSize {
		return nil, fmt.Errorf("%w: batch size too large (max %d), got %d", ErrInvalidBatchSize, maxBatchSize, n)
	}

	s.mu.Lock() // TODO:GG
	defer s.mu.Unlock()

	ids := make([]int64, 0, n)
	remainingIDs := n

	for remainingIDs > 0 {
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
					// 使用重试机制等待时钟追赶
					retries := 0
					for retries < maxWaitRetries {
						// 使用 offset+1 确保时钟真正前进
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

		// 永远不会触发（当前时间总在Epoch之后）
		//if timeDiff < 0 {
		//	return ids, fmt.Errorf("%w: timestamp %d is before epoch %d (generated %d IDs)",
		//		ErrTimestampOverflow, timestamp, Epoch, len(ids))
		//}
		// 即使用满了69年，后续也会改进更好的方案，理论上不用检查
		//if timeDiff > maxTimestampDiff {
		//	return ids, fmt.Errorf("%w: timestamp difference %d exceeds maximum %d (generated %d IDs)",
		//		ErrTimestampOverflow, timeDiff, maxTimestampDiff, len(ids))
		//}

		// 计算当前毫秒可以生成的ID数量
		var availableInCurrentMs int
		if timestamp == s.lastTimestamp {
			// 同一毫秒内，计算剩余可用序列号数量
			// s.sequence 是当前已使用过的序列号（0-4095）
			// 可用数量：MaxSequence - s.sequence（例如：s.sequence=9，则剩余4095-9=4086个）
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
				// 即使用满了69年，后续也会改进更好的方案，理论上不用检查
				//timeDiff = timestamp - Epoch
				//if timeDiff > maxTimestampDiff {
				//	return ids, fmt.Errorf("%w: timestamp after wait is invalid (generated %d IDs)",
				//		ErrTimestampOverflow, len(ids))
				//}

				// 新毫秒，序列号重置为-1，下面循环会从0开始
				s.sequence = -1
				s.lastTimestamp = timestamp
				availableInCurrentMs = MaxSequence + 1 // 0-4095，共4096个
			}
		} else {
			// 新的毫秒，序列号重置为-1，下面循环会从0开始
			s.sequence = -1
			s.lastTimestamp = timestamp
			availableInCurrentMs = MaxSequence + 1 // 0-4095，共4096个
		}

		// 本轮生成数量
		batchSize := remainingIDs
		if batchSize > availableInCurrentMs {
			batchSize = availableInCurrentMs
		}

		// timeDiff必须在所有timestamp更新完成后计算
		// 这样可以确保序列号溢出等待下一毫秒后，使用的是新的timestamp
		timeDiff := timestamp - Epoch

		// 使用预计算的部分（在循环外计算以提高性能）
		baseID := (timeDiff << TimestampShift) | s.precomputedPart

		// 生成ID：先递增序列号，再使用
		for i := 0; i < batchSize; i++ {
			s.sequence++
			id := baseID | s.sequence
			ids = append(ids, id)
		}

		// 剩下的remainingIDs会进入下一个循环
		remainingIDs -= batchSize
	}

	if s.enableMetrics && s.metrics != nil {
		s.metrics.IDCount.Add(uint64(n))
	}

	return ids, nil
}

// nextIDUnsafe 内部使用的不加锁版本的ID生成方法
func (s *Snowflake) nextIDUnsafe() (int64, error) {
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

	// 永远不会触发（当前时间总在Epoch之后）
	//if timeDiff < 0 {
	//	return 0, fmt.Errorf("%w: current time %d is before epoch %d",
	//		ErrTimestampOverflow, timestamp, Epoch)
	//}
	// 即使用满了69年，后续也会改进更好的方案，理论上不用检查
	//if timeDiff > maxTimestampDiff {
	//	return 0, fmt.Errorf("%w: timestamp difference %d exceeds maximum %d",
	//		ErrTimestampOverflow, timeDiff, maxTimestampDiff)
	//}

	// 同一毫秒内，序列号递增；不同毫秒，序列号重置
	if timestamp == s.lastTimestamp {
		// 先检查是否会溢出，再递增
		if s.sequence >= MaxSequence {
			// 序列号已达上限，需要等待下一毫秒
			if s.enableMetrics && s.metrics != nil {
				s.metrics.SequenceOverflow.Add(1)
				s.metrics.WaitCount.Add(1)
			}
			startTime := time.Now()
			timestamp = s.waitNextMillis(s.lastTimestamp)
			if s.enableMetrics && s.metrics != nil {
				s.metrics.TotalWaitTimeNs.Add(uint64(time.Since(startTime).Nanoseconds()))
			}

			// 重置序列号为-1，后面会递增为0
			s.sequence = -1
			s.lastTimestamp = timestamp
		}
		// 序列号递增（溢出后也会执行到这里）
		s.sequence++
	} else {
		// 不同毫秒，序列号重置为0
		s.sequence = 0
		s.lastTimestamp = timestamp
	}

	// timeDiff必须在所有timestamp更新完成后计算
	// 这样可以确保序列号溢出等待下一毫秒后，使用的是新的timestamp
	timeDiff := timestamp - Epoch

	// 使用预计算的部分组装ID，减少位运算
	id := (timeDiff << TimestampShift) | s.precomputedPart | s.sequence

	// 只在启用监控时才更新指标
	if s.enableMetrics && s.metrics != nil {
		s.metrics.IDCount.Add(1)
	}

	return id, nil
}

// GetMetrics 获取性能监控指标（线程安全）
func (s *Snowflake) GetMetrics() map[string]uint64 {
	if !s.enableMetrics || s.metrics == nil {
		return map[string]uint64{
			"metrics_enabled": 0,
		}
	}

	// 原子读取，避免数据竞态
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

// GetMetricsSnapshot 获取性能指标的快照
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

// ResetMetrics 重置性能监控指标（仅用于测试）
func (s *Snowflake) ResetMetrics() {
	if !s.enableMetrics || s.metrics == nil {
		return
	}
	s.metrics.IDCount.Store(0)
	s.metrics.SequenceOverflow.Store(0)
	s.metrics.ClockBackward.Store(0)
	s.metrics.WaitCount.Store(0)
	s.metrics.TotalWaitTimeNs.Store(0)
}
