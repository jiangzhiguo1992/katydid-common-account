package domain

import (
	"fmt"
	"katydid-common-account/pkg/idgen/core"
	"katydid-common-account/pkg/idgen/registry"
	"time"
)

// Parse 解析ID，提取元信息
// 说明：使用默认生成器类型（Snowflake）进行解析
func (id ID) Parse() (*core.IDInfo, error) {
	return id.ParseWithType(defaultGeneratorType)
}

// ParseWithType 使用指定生成器类型解析ID
func (id ID) ParseWithType(generatorType core.GeneratorType) (*core.IDInfo, error) {
	if !id.IsValid() {
		return nil, fmt.Errorf("%w: got %d", core.ErrInvalidSnowflakeID, id)
	}

	if !generatorType.IsValid() {
		return nil, fmt.Errorf("%w: %s", core.ErrInvalidGeneratorType, generatorType)
	}

	parser, err := registry.GetParserRegistry().Get(generatorType)
	if err != nil {
		return nil, fmt.Errorf("failed to get parser: %w", err)
	}

	return parser.Parse(int64(id))
}

// ExtractTime 提取时间戳
func (id ID) ExtractTime() time.Time {
	return id.ExtractTimeWithType(defaultGeneratorType)
}

// ExtractTimeWithType 使用指定生成器类型提取时间戳
func (id ID) ExtractTimeWithType(generatorType core.GeneratorType) time.Time {
	if !id.IsValid() {
		return time.Time{} // ID无效，返回零值
	}

	if !generatorType.IsValid() {
		return time.Time{} // 类型无效，返回零值
	}

	parser, err := registry.GetParserRegistry().Get(generatorType)
	if err != nil {
		return time.Time{} // 解析器获取失败，返回零值
	}
	timestamp := parser.ExtractTimestamp(int64(id))

	// 确保时间戳合理
	if timestamp <= 0 {
		return time.Time{}
	}

	return time.UnixMilli(timestamp)
}

// ExtractDatacenterID 提取数据中心ID
func (id ID) ExtractDatacenterID() int64 {
	return id.ExtractDatacenterIDWithType(defaultGeneratorType)
}

// ExtractDatacenterIDWithType 使用指定生成器类型提取数据中心ID
func (id ID) ExtractDatacenterIDWithType(generatorType core.GeneratorType) int64 {
	if !id.IsValid() {
		return -1 // ID无效
	}

	if !generatorType.IsValid() {
		return -1 // 类型无效
	}

	parser, err := registry.GetParserRegistry().Get(generatorType)
	if err != nil {
		return -1 // 解析器获取失败
	}
	return parser.ExtractDatacenterID(int64(id))
}

// ExtractWorkerID 提取工作机器ID
func (id ID) ExtractWorkerID() int64 {
	return id.ExtractWorkerIDWithType(defaultGeneratorType)
}

// ExtractWorkerIDWithType 使用指定生成器类型提取工作机器ID
func (id ID) ExtractWorkerIDWithType(generatorType core.GeneratorType) int64 {
	if !id.IsValid() {
		return -1 // ID无效
	}

	if !generatorType.IsValid() {
		return -1 // 类型无效
	}

	parser, err := registry.GetParserRegistry().Get(generatorType)
	if err != nil {
		return -1 // 解析器获取失败
	}
	return parser.ExtractWorkerID(int64(id))
}

// ExtractSequence 提取序列号
func (id ID) ExtractSequence() int64 {
	return id.ExtractSequenceWithType(defaultGeneratorType)
}

// ExtractSequenceWithType 使用指定生成器类型提取序列号
func (id ID) ExtractSequenceWithType(generatorType core.GeneratorType) int64 {
	if !id.IsValid() {
		return -1 // ID无效
	}

	if !generatorType.IsValid() {
		return -1 // 类型无效
	}

	parser, err := registry.GetParserRegistry().Get(generatorType)
	if err != nil {
		return -1 // 解析器获取失败
	}
	return parser.ExtractSequence(int64(id))
}
