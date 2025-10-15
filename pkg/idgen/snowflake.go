package idgen

import (
	"errors"
	"sync"
	"time"
)

const (
	// Epoch 起始时间戳 (2026-01-01 00:00:00 UTC)
	Epoch int64 = 1767196800000

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

// NextID 生成下一个ID
func (s *Snowflake) NextID() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取当前毫秒时间戳
	timestamp := s.currentTimestamp()

	// 时钟回拨检测
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
		// 不同毫秒，序列号重置
		s.sequence = 0
	}

	s.lastTimestamp = timestamp

	// 组装ID
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

// waitNextMillis 等待下一毫秒
func (s *Snowflake) waitNextMillis(lastTimestamp int64) int64 {
	timestamp := s.currentTimestamp()
	for timestamp <= lastTimestamp {
		timestamp = s.currentTimestamp()
	}
	return timestamp
}

// ParseSnowflakeID 解析ID，返回时间戳、数据中心ID、工作机器ID和序列号
func ParseSnowflakeID(id int64) (timestamp int64, datacenterID int64, workerID int64, sequence int64) {
	timestamp = (id >> TimestampShift) + Epoch
	datacenterID = (id >> DatacenterIDShift) & MaxDatacenterID
	workerID = (id >> WorkerIDShift) & MaxWorkerID
	sequence = id & MaxSequence
	return
}

// GetTimestamp 从ID中提取时间戳
func GetTimestamp(id int64) time.Time {
	timestamp := (id >> TimestampShift) + Epoch
	return time.UnixMilli(timestamp)
}
