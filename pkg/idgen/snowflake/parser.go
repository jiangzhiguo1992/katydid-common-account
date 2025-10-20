package snowflake

import (
	"fmt"
	"time"

	"katydid-common-account/pkg/idgen/core"
)

// Parser Snowflake ID解析器
type Parser struct {
	validator core.IDValidator // 验证器，用于解析前验证ID有效性
}

// NewParser 创建新的解析器实例
func NewParser() *Parser {
	return &Parser{
		validator: NewValidator(),
	}
}

// Parse 解析Snowflake ID，提取完整的元信息
// 实现core.IDParser接口
func (p *Parser) Parse(id int64) (*core.IDInfo, error) {
	// 步骤1：先验证ID的有效性
	// 说明：只解析有效的ID，避免返回错误的元信息
	if err := p.validator.Validate(id); err != nil {
		return nil, fmt.Errorf("invalid snowflake ID: %w", err)
	}

	// 步骤2：提取各部分信息（使用位运算）
	// 时间戳：右移22位，加上Epoch得到Unix毫秒时间戳
	timestamp := (id >> TimestampShift) + Epoch

	// 数据中心ID：右移17位后与掩码31进行与运算，提取5位
	datacenterID := (id >> DatacenterIDShift) & MaxDatacenterID

	// 工作机器ID：右移12位后与掩码31进行与运算，提取5位
	workerID := (id >> WorkerIDShift) & MaxWorkerID

	// 序列号：与掩码4095进行与运算，提取低12位
	sequence := id & MaxSequence

	// 步骤3：返回完整信息
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
	// 快速失败：无效ID直接返回0
	if id <= 0 {
		return 0
	}
	// 位运算提取时间戳部分并加上Epoch
	return (id >> TimestampShift) + Epoch
}

// ExtractTimestampAsTime 从Snowflake ID中提取时间戳并转换为time.Time
func (p *Parser) ExtractTimestampAsTime(id int64) time.Time {
	timestamp := p.ExtractTimestamp(id)
	// 无效时间戳返回零值时间
	if timestamp <= 0 {
		return time.Time{}
	}
	// 将Unix毫秒时间戳转换为time.Time
	return time.UnixMilli(timestamp)
}

// ExtractDatacenterID 从Snowflake ID中提取数据中心ID
// 实现core.IDParser接口
func (p *Parser) ExtractDatacenterID(id int64) int64 {
	// 快速失败：无效ID返回-1
	if id <= 0 {
		return -1
	}
	// 位运算提取数据中心ID（右移17位，取低5位）
	return (id >> DatacenterIDShift) & MaxDatacenterID
}

// ExtractWorkerID 从Snowflake ID中提取工作机器ID
// 实现core.IDParser接口
func (p *Parser) ExtractWorkerID(id int64) int64 {
	// 快速失败：无效ID返回-1
	if id <= 0 {
		return -1
	}
	// 位运算提取工作机器ID（右移12位，取低5位）
	return (id >> WorkerIDShift) & MaxWorkerID
}

// ExtractSequence 从Snowflake ID中提取序列号
// 实现core.IDParser接口
func (p *Parser) ExtractSequence(id int64) int64 {
	// 快速失败：无效ID返回-1
	if id <= 0 {
		return -1
	}
	// 位运算提取序列号（取低12位）
	return id & MaxSequence
}

// ParseSnowflakeID 全局解析函数
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

// GetTimestamp 全局时间戳提取函数
func GetTimestamp(id int64) time.Time {
	return NewParser().ExtractTimestampAsTime(id)
}
