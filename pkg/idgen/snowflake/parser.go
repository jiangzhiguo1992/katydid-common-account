package snowflake

import (
	"fmt"
	"time"

	"katydid-common-account/pkg/idgen/core"
)

// Parser Snowflake ID解析器（单一职责：只负责ID解析）
// 实现core.IDParser接口（里氏替换原则）
type Parser struct {
	validator core.IDValidator
}

// NewParser 创建新的解析器实例
func NewParser() *Parser {
	return &Parser{
		validator: NewValidator(),
	}
}

// Parse 解析Snowflake ID，提取时间戳、数据中心ID、工作机器ID、序列号
// 实现core.IDParser接口
func (p *Parser) Parse(id int64) (*core.IDInfo, error) {
	// 先验证ID的有效性
	if err := p.validator.Validate(id); err != nil {
		return nil, fmt.Errorf("invalid snowflake ID: %w", err)
	}

	// 提取各部分信息，使用位运算和掩码
	timestamp := (id >> TimestampShift) + Epoch
	datacenterID := (id >> DatacenterIDShift) & MaxDatacenterID
	workerID := (id >> WorkerIDShift) & MaxWorkerID
	sequence := id & MaxSequence

	return &core.IDInfo{
		ID:           id,
		Timestamp:    timestamp,
		DatacenterID: datacenterID,
		WorkerID:     workerID,
		Sequence:     sequence,
	}, nil
}

// ExtractTimestamp 从Snowflake ID中提取时间戳（Unix毫秒）
// 实现core.IDParser接口
func (p *Parser) ExtractTimestamp(id int64) int64 {
	if id <= 0 {
		return 0 // 无效ID返回0
	}
	return (id >> TimestampShift) + Epoch
}

// ExtractTimestampAsTime 从Snowflake ID中提取时间戳并转换为time.Time
func (p *Parser) ExtractTimestampAsTime(id int64) time.Time {
	timestamp := p.ExtractTimestamp(id)
	if timestamp <= 0 {
		return time.Time{} // 返回零值时间
	}
	return time.UnixMilli(timestamp)
}

// ExtractDatacenterID 从Snowflake ID中提取数据中心ID
// 实现core.IDParser接口
func (p *Parser) ExtractDatacenterID(id int64) int64 {
	if id <= 0 {
		return -1 // 无效ID返回-1
	}
	return (id >> DatacenterIDShift) & MaxDatacenterID
}

// ExtractWorkerID 从Snowflake ID中提取工作机器ID
// 实现core.IDParser接口
func (p *Parser) ExtractWorkerID(id int64) int64 {
	if id <= 0 {
		return -1 // 无效ID返回-1
	}
	return (id >> WorkerIDShift) & MaxWorkerID
}

// ExtractSequence 从Snowflake ID中提取序列号
// 实现core.IDParser接口
func (p *Parser) ExtractSequence(id int64) int64 {
	if id <= 0 {
		return -1 // 无效ID返回-1
	}
	return id & MaxSequence
}

// ParseSnowflakeID 全局解析函数（向后兼容）
func ParseSnowflakeID(id int64) (timestamp int64, datacenterID int64, workerID int64, sequence int64) {
	if id <= 0 {
		return 0, -1, -1, -1
	}
	timestamp = (id >> TimestampShift) + Epoch
	datacenterID = (id >> DatacenterIDShift) & MaxDatacenterID
	workerID = (id >> WorkerIDShift) & MaxWorkerID
	sequence = id & MaxSequence
	return
}

// GetTimestamp 全局时间戳提取函数（向后兼容）
func GetTimestamp(id int64) time.Time {
	return NewParser().ExtractTimestampAsTime(id)
}
