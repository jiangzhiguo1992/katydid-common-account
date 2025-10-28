package core

// IIDGenerator ID生成器基础接口
type IIDGenerator interface {
	// NextID 生成下一个唯一ID（线程安全）
	NextID() (int64, error)
}

// IBatchGenerator 批量ID生成接口
type IBatchGenerator interface {
	IIDGenerator

	// NextIDBatch 批量生成指定数量的ID（线程安全）
	NextIDBatch(n int) ([]int64, error)
}

// IConfigurableGenerator 可配置的生成器接口
type IConfigurableGenerator interface {
	// GetWorkerID 获取工作机器ID
	// 返回值：工作机器ID（0-31）
	GetWorkerID() int64

	// GetDatacenterID 获取数据中心ID
	// 返回值：数据中心ID（0-31）
	GetDatacenterID() int64
}

// IMonitorableGenerator 可监控的生成器接口
type IMonitorableGenerator interface {
	// GetMetrics 获取性能监控指标
	GetMetrics() map[string]uint64

	// ResetMetrics 重置性能监控指标
	ResetMetrics()

	// GetIDCount 获取已生成的ID总数
	GetIDCount() uint64
}

// IValidaParseableGenerator 可验证+解析的生成器接口
type IValidaParseableGenerator interface {
	// ParseID 解析ID，提取其中的时间戳、机器ID等元信息
	ParseID(id int64) (*IDInfo, error)

	// ValidateID 验证ID的有效性
	ValidateID(id int64) error
}

// IGenerator 完整功能的生成器接口
type IGenerator interface {
	IIDGenerator
	IBatchGenerator
	IConfigurableGenerator
	IMonitorableGenerator
	IValidaParseableGenerator
}

// IGeneratorFactory 生成器工厂接口
type IGeneratorFactory interface {
	// Create 根据配置创建生成器实例
	Create(config any) (IGenerator, error)
}

// IDInfo ID信息结构
type IDInfo struct {
	ID           int64 // 原始ID值
	Timestamp    int64 // 时间戳（Unix毫秒）
	DatacenterID int64 // 数据中心ID（0-31）
	WorkerID     int64 // 工作机器ID（0-31）
	Sequence     int64 // 序列号（0-4095，同一毫秒内的序号）
}

// IIDParser ID解析器接口
type IIDParser interface {
	// Parse 解析ID，提取完整的元信息
	Parse(id int64) (*IDInfo, error)

	// ExtractTimestamp 提取时间戳（Unix毫秒）
	ExtractTimestamp(id int64) int64

	// ExtractDatacenterID 提取数据中心ID
	ExtractDatacenterID(id int64) int64

	// ExtractWorkerID 提取工作机器ID
	ExtractWorkerID(id int64) int64

	// ExtractSequence 提取序列号
	ExtractSequence(id int64) int64
}

// IIDValidator ID验证器接口
type IIDValidator interface {
	// Validate 验证ID的有效性
	Validate(id int64) error

	// ValidateBatch 批量验证ID
	ValidateBatch(ids []int64) error
}
