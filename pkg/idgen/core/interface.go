package core

// IDGenerator 定义ID生成器的基础接口（单一职责：只负责生成ID）
// 遵循接口隔离原则：客户端只需要知道如何生成ID
type IDGenerator interface {
	// NextID 生成下一个唯一ID
	// 返回：生成的ID和可能的错误
	NextID() (int64, error)
}

// BatchGenerator 批量生成接口（接口隔离：将批量功能独立出来）
// 不是所有生成器都支持批量生成，因此独立为单独接口
type BatchGenerator interface {
	IDGenerator
	// NextIDBatch 批量生成指定数量的ID
	// 参数：n - 要生成的ID数量
	// 返回：ID切片和可能的错误
	NextIDBatch(n int) ([]int64, error)
}

// ConfigurableGenerator 可配置的生成器接口（接口隔离）
// 支持运行时获取配置信息
type ConfigurableGenerator interface {
	// GetWorkerID 获取工作机器ID
	GetWorkerID() int64
	// GetDatacenterID 获取数据中心ID
	GetDatacenterID() int64
}

// MonitorableGenerator 可监控的生成器接口（接口隔离）
// 支持性能监控和指标收集
type MonitorableGenerator interface {
	// GetMetrics 获取性能监控指标
	GetMetrics() map[string]uint64
	// ResetMetrics 重置性能监控指标
	ResetMetrics()
	// GetIDCount 获取已生成的ID总数
	GetIDCount() uint64
}

// ParseableGenerator 可解析的生成器接口（接口隔离）
// 支持从ID中提取元信息
type ParseableGenerator interface {
	// ParseID 解析ID，提取其中的时间戳、机器ID等信息
	ParseID(id int64) (*IDInfo, error)
	// ValidateID 验证ID的有效性
	ValidateID(id int64) error
}

// FullFeaturedGenerator 完整功能的生成器接口
// 组合所有功能接口，遵循接口组合原则
type FullFeaturedGenerator interface {
	IDGenerator
	BatchGenerator
	ConfigurableGenerator
	MonitorableGenerator
	ParseableGenerator
}

// GeneratorFactory 生成器工厂接口（依赖倒置：依赖抽象而非具体实现）
// 用于创建不同类型的ID生成器
type GeneratorFactory interface {
	// Create 根据配置创建生成器实例
	// 参数：config - 生成器配置（使用any以支持不同类型的配置）
	// 返回：生成器实例和可能的错误
	Create(config any) (IDGenerator, error)
}

// IDInfo ID信息结构（用于解析结果）
type IDInfo struct {
	ID           int64 // 原始ID
	Timestamp    int64 // 时间戳（毫秒）
	DatacenterID int64 // 数据中心ID
	WorkerID     int64 // 工作机器ID
	Sequence     int64 // 序列号
}
