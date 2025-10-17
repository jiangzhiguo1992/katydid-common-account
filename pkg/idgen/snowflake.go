package idgen

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Epoch 起始时间戳 (2026-01-01 00:00:00 UTC)
	//Epoch int64 = 1767196800000 // 毫秒时间戳
	Epoch int64 = 1672502400000 // 毫秒时间戳 (前期测试用2024)

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

	// 性能优化：等待下一毫秒时的休眠时间（微秒）
	sleepDuration = 100 * time.Microsecond

	// 时钟回拨最大容忍时间（毫秒）
	maxClockBackwardTolerance = 5
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
)

// IDGenerator 定义ID生成器接口（接口隔离原则）
// 该接口只包含ID生成的核心功能，遵循最小接口原则
type IDGenerator interface {
	// NextID 生成下一个唯一ID
	// 返回生成的ID和可能的错误
	NextID() (int64, error)
}

// IDParser 定义ID解析器接口（接口隔离原则）
// 将解析功能与生成功能分离，提高灵活性
type IDParser interface {
	// Parse 解析ID，提取其中的各个组成部分
	Parse(id int64) (*IDInfo, error)
}

// IDInfo ID解析后的信息结构体
type IDInfo struct {
	ID           int64     `json:"id"`            // 原始ID
	Timestamp    int64     `json:"timestamp"`     // 时间戳（毫秒）
	Time         time.Time `json:"time"`          // 时间对象
	DatacenterID int64     `json:"datacenter_id"` // 数据中心ID
	WorkerID     int64     `json:"worker_id"`     // 工作机器ID
	Sequence     int64     `json:"sequence"`      // 序列号
}

// Snowflake Snowflake算法的ID生成器实现
// 实现了IDGenerator和IDParser接口（里氏替换原则）
type Snowflake struct {
	// 使用互斥锁保护并发访问（线程安全）
	mu sync.Mutex

	// 上次生成ID的时间戳
	lastTimestamp int64

	// 工作机器ID（0-31）
	workerID int64

	// 数据中心ID（0-31）
	datacenterID int64

	// 当前毫秒内的序列号（0-4095）
	sequence int64

	// 使用atomic计数器统计生成的ID数量（性能监控）
	idCount atomic.Uint64 // TODO:GG

	// 时间提供者函数（依赖倒置原则 - 便于测试）
	timeFunc func() int64
}

// SnowflakeConfig Snowflake配置选项（易用性优化）
type SnowflakeConfig struct {
	DatacenterID int64        // 数据中心ID
	WorkerID     int64        // 工作机器ID
	TimeFunc     func() int64 // 自定义时间函数（可选，用于测试）
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
		DatacenterID: datacenterID,
		WorkerID:     workerID,
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

	if config.WorkerID < 0 || config.WorkerID > MaxWorkerID {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidWorkerID, config.WorkerID)
	}

	if config.DatacenterID < 0 || config.DatacenterID > MaxDatacenterID {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidDatacenterID, config.DatacenterID)
	}

	// 默认时间函数
	timeFunc := config.TimeFunc
	if timeFunc == nil {
		timeFunc = func() int64 {
			return time.Now().UnixNano() / 1e6
		}
	}

	return &Snowflake{
		workerID:      config.WorkerID,
		datacenterID:  config.DatacenterID,
		lastTimestamp: -1,
		sequence:      0,
		timeFunc:      timeFunc,
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

	// 获取当前毫秒时间戳
	timestamp := s.timeFunc()

	// 时钟回拨检测：如果当前时间小于上次时间戳，说明发生了时钟回拨
	if timestamp < s.lastTimestamp {
		// 计算回拨的时间差
		offset := s.lastTimestamp - timestamp

		// 如果回拨时间在容忍范围内，等待追上
		if offset <= maxClockBackwardTolerance {
			time.Sleep(time.Duration(offset) * time.Millisecond)
			timestamp = s.timeFunc()

			// 再次检查，如果还是回拨则返回错误
			if timestamp < s.lastTimestamp {
				return 0, fmt.Errorf("%w: backward %dms", ErrClockMovedBackwards, offset)
			}
		} else {
			// 回拨时间过长，直接返回错误
			return 0, fmt.Errorf("%w: backward %dms", ErrClockMovedBackwards, offset)
		}
	}

	// 检查时间戳是否会溢出（健壮性）
	timeDiff := timestamp - Epoch
	if timeDiff < 0 {
		return 0, fmt.Errorf("%w: current time %d is before epoch %d",
			ErrTimestampOverflow, timestamp, Epoch)
	}

	// 检查时间戳位数是否超出限制（41位）
	maxTimestamp := int64(1<<41 - 1)
	if timeDiff > maxTimestamp {
		return 0, fmt.Errorf("%w: timestamp difference %d exceeds maximum %d",
			ErrTimestampOverflow, timeDiff, maxTimestamp)
	}

	// 同一毫秒内，序列号递增
	if timestamp == s.lastTimestamp {
		s.sequence = (s.sequence + 1) & MaxSequence
		// 序列号溢出，等待下一毫秒
		if s.sequence == 0 {
			timestamp = s.waitNextMillis(s.lastTimestamp)
		}
	} else {
		// 不同毫秒，序列号重置为0
		s.sequence = 0
	}

	// 更新最后时间戳
	s.lastTimestamp = timestamp

	// 组装ID：时间戳(41位) | 数据中心ID(5位) | 工作机器ID(5位) | 序列号(12位)
	id := ((timestamp - Epoch) << TimestampShift) |
		(s.datacenterID << DatacenterIDShift) |
		(s.workerID << WorkerIDShift) |
		s.sequence

	// 增加计数器（用于监控）
	s.idCount.Add(1)

	return id, nil
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
	timestamp := s.timeFunc()
	for timestamp <= lastTimestamp {
		// 短暂休眠，避免CPU空转（只阻塞当前goroutine）
		time.Sleep(sleepDuration)
		timestamp = s.timeFunc()
	}
	return timestamp
}

// Parse 解析Snowflake ID，提取其中的各个组成部分
// 实现IDParser接口
//
// 参数:
//
//	id: 要解析的Snowflake ID
//
// 返回:
//
//	*IDInfo: ID信息结构体
//	error: 解析失败时返回错误
func (s *Snowflake) Parse(id int64) (*IDInfo, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidSnowflakeID, id)
	}

	timestamp := (id >> TimestampShift) + Epoch
	datacenterID := (id >> DatacenterIDShift) & MaxDatacenterID
	workerID := (id >> WorkerIDShift) & MaxWorkerID
	sequence := id & MaxSequence

	return &IDInfo{
		ID:           id,
		Timestamp:    timestamp,
		Time:         time.UnixMilli(timestamp),
		DatacenterID: datacenterID,
		WorkerID:     workerID,
		Sequence:     sequence,
	}, nil
}

// GetIDCount 获取已生成的ID总数（用于监控）
//
// 返回:
//
//	uint64: 已生成的ID数量
func (s *Snowflake) GetIDCount() uint64 {
	return s.idCount.Load()
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
