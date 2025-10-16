package idgen

import (
	"errors"
	"sync"
	"time"
)

const (
	// Epoch 起始时间戳 (2026-01-01 00:00:00 UTC)
	Epoch int64 = 1767196800000 // 1735660800000

	// 位数分配
	WorkerIDBits     = 5  // 工作机器ID位数
	DatacenterIDBits = 5  // 数据中心ID位数
	SequenceBits     = 12 // 序列号位数

	// 最大值
	MaxWorkerID     = -1 ^ (-1 << WorkerIDBits)     // 31
	MaxDatacenterID = -1 ^ (-1 << DatacenterIDBits) // 31
	MaxSequence     = -1 ^ (-1 << SequenceBits)     // 4095

	// 位移
	WorkerIDShift     = SequenceBits                                   // 12
	DatacenterIDShift = SequenceBits + WorkerIDBits                    // 17
	TimestampShift    = SequenceBits + WorkerIDBits + DatacenterIDBits // 22

	// 性能优化：等待下一毫秒时的休眠时间（微秒）
	sleepDuration = 100 * time.Microsecond
)

var (
	ErrInvalidWorkerID     = errors.New("worker ID must be between 0 and 31")
	ErrInvalidDatacenterID = errors.New("datacenter ID must be between 0 and 31")
	ErrClockMovedBackwards = errors.New("clock moved backwards")
)

// Snowflake ID生成器
type Snowflake struct {
	mu            sync.Mutex
	lastTimestamp int64
	workerID      int64
	datacenterID  int64
	sequence      int64
}

// NewSnowflake 创建一个新的Snowflake ID生成器
// 参数:
//
//	datacenterID: 数据中心ID，取值范围 [0, 31]
//	workerID: 工作机器ID，取值范围 [0, 31]
//
// 返回:
//
//	*Snowflake: Snowflake ID生成器实例
//	error: 参数验证失败时返回错误
func NewSnowflake(datacenterID, workerID int64) (*Snowflake, error) {
	if workerID < 0 || workerID > MaxWorkerID {
		return nil, ErrInvalidWorkerID
	}
	if datacenterID < 0 || datacenterID > MaxDatacenterID {
		return nil, ErrInvalidDatacenterID
	}

	return &Snowflake{
		workerID:      workerID,
		datacenterID:  datacenterID,
		lastTimestamp: -1,
		sequence:      0,
	}, nil
}

// NextID 生成下一个唯一ID
// 该方法是线程安全的，可以在多个goroutine中并发调用
// 返回:
//
//	int64: 生成的唯一ID
//	error: 当检测到时钟回拨时返回错误
func (s *Snowflake) NextID() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取当前毫秒时间戳
	timestamp := s.currentTimestamp()

	// 时钟回拨检测：如果当前时间小于上次时间戳，说明发生了时钟回拨
	if timestamp < s.lastTimestamp {
		return 0, ErrClockMovedBackwards
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

	s.lastTimestamp = timestamp

	// 组装ID：时间戳(41位) | 数据中心ID(5位) | 工作机器ID(5位) | 序列号(12位)
	id := ((timestamp - Epoch) << TimestampShift) |
		(s.datacenterID << DatacenterIDShift) |
		(s.workerID << WorkerIDShift) |
		s.sequence

	return id, nil
}

// currentTimestamp 获取当前时间戳（毫秒）
func (s *Snowflake) currentTimestamp() int64 {
	return time.Now().UnixNano() / 1e6
}

// waitNextMillis 等待直到获取到比lastTimestamp更大的时间戳
// 使用短暂休眠代替忙等待，减少CPU占用
// 参数:
//
//	lastTimestamp: 上一次的时间戳
//
// 返回:
//
//	int64: 新的时间戳（保证大于lastTimestamp）
func (s *Snowflake) waitNextMillis(lastTimestamp int64) int64 {
	timestamp := s.currentTimestamp()
	for timestamp <= lastTimestamp {
		// 短暂休眠，避免CPU空转 (只阻塞当前goroutine)
		time.Sleep(sleepDuration)
		timestamp = s.currentTimestamp()
	}
	return timestamp
}

// ParseSnowflakeID 解析Snowflake ID，提取其中的时间戳、数据中心ID、工作机器ID和序列号
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
