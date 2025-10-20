package core

import "errors"

var (
	// ErrInvalidWorkerID 工作机器ID超出有效范围
	ErrInvalidWorkerID = errors.New("invalid worker id: must be between 0 and 31")

	// ErrInvalidDatacenterID 数据中心ID超出有效范围
	ErrInvalidDatacenterID = errors.New("invalid datacenter id: must be between 0 and 31")

	// ErrClockMovedBackwards 检测到时钟回拨
	ErrClockMovedBackwards = errors.New("clock moved backwards: refusing to generate id")

	// ErrInvalidSnowflakeID 无效的Snowflake ID
	ErrInvalidSnowflakeID = errors.New("invalid snowflake id: id must be positive")

	// ErrInvalidBatchSize 批量生成数量无效
	ErrInvalidBatchSize = errors.New("invalid batch size")

	// ErrNilConfig 配置为nil
	ErrNilConfig = errors.New("config cannot be nil")

	// ErrGeneratorNotFound 生成器未找到
	ErrGeneratorNotFound = errors.New("generator not found")

	// ErrGeneratorAlreadyExists 生成器已存在
	ErrGeneratorAlreadyExists = errors.New("generator already exists")

	// ErrInvalidGeneratorType 无效的生成器类型
	ErrInvalidGeneratorType = errors.New("invalid generator type")

	// ErrInvalidKey 无效的键
	ErrInvalidKey = errors.New("invalid key")

	// ErrFactoryNotFound 工厂未找到
	ErrFactoryNotFound = errors.New("factory not found")

	// ErrMaxGeneratorsReached 达到最大生成器数量
	ErrMaxGeneratorsReached = errors.New("maximum number of generators reached")
)
