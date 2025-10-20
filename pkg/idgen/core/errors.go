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
	ErrInvalidBatchSize = errors.New("invalid batch size: must be positive and within limits")

	// ErrNilConfig 配置对象为nil
	ErrNilConfig = errors.New("config cannot be nil")

	// ErrGeneratorNotFound 生成器未找到
	ErrGeneratorNotFound = errors.New("generator not found: key does not exist in registry")

	// ErrGeneratorAlreadyExists 生成器已存在
	ErrGeneratorAlreadyExists = errors.New("generator already exists: key is already registered")

	// ErrInvalidGeneratorType 无效的生成器类型
	ErrInvalidGeneratorType = errors.New("invalid generator type: type is not supported")

	// ErrInvalidKey 无效的键
	ErrInvalidKey = errors.New("invalid key: key is empty, too long, or contains invalid characters")

	// ErrFactoryNotFound 工厂未找到
	ErrFactoryNotFound = errors.New("factory not found: no factory registered for the specified type")

	// ErrMaxGeneratorsReached 达到最大生成器数量
	ErrMaxGeneratorsReached = errors.New("maximum number of generators reached: cannot create more generators")

	// ErrParserNotFound 解析器未找到
	ErrParserNotFound = errors.New("parser not found: no parser registered for the specified type")

	// ErrValidatorNotFound 验证器未找到
	ErrValidatorNotFound = errors.New("validator not found: no validator registered for the specified type")

	// ErrInvalidKeyFormat 无效的键格式
	ErrInvalidKeyFormat = errors.New("invalid key format: only alphanumeric, underscore, hyphen, and dot allowed")
)
