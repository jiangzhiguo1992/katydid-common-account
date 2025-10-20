package registry

import (
	"log"
	"sync"

	"katydid-common-account/pkg/idgen/core"
	"katydid-common-account/pkg/idgen/snowflake"
)

const (
	// DefaultGeneratorKey 默认生成器的注册键
	DefaultGeneratorKey = "default"
)

var (
	// defaultGenerator 默认生成器实例（单例）
	defaultGenerator core.IDGenerator

	// defaultGeneratorOnce 确保默认生成器只初始化一次
	defaultGeneratorOnce sync.Once

	// defaultGeneratorErr 默认生成器初始化时的错误
	defaultGeneratorErr error
)

// GetDefaultGenerator 获取默认的Snowflake生成器
func GetDefaultGenerator() (core.IDGenerator, error) {
	defaultGeneratorOnce.Do(func() {
		// 创建默认配置
		config := &snowflake.Config{
			DatacenterID:  0,     // 默认数据中心ID
			WorkerID:      0,     // 默认工作机器ID
			EnableMetrics: false, // 默认关闭监控（性能优先）
		}

		// 从工厂创建生成器
		factory := NewSnowflakeFactory()
		defaultGenerator, defaultGeneratorErr = factory.Create(config)

		if defaultGeneratorErr != nil {
			log.Println("默认生成器初始化失败", "error", defaultGeneratorErr)
		} else {
			log.Println("默认生成器初始化成功", "datacenter_id", 0, "worker_id", 0)
		}
	})

	return defaultGenerator, defaultGeneratorErr
}

// GetOrCreateDefaultGenerator 获取或创建默认生成器
// 说明：优先从注册表获取，不存在时创建新的默认生成器
func GetOrCreateDefaultGenerator() (core.IDGenerator, error) {
	registry := GetRegistry()

	// 步骤1：尝试从注册表获取
	if registry.Has(DefaultGeneratorKey) {
		return registry.Get(DefaultGeneratorKey)
	}

	// 步骤2：不存在时创建默认配置
	config := &snowflake.Config{
		DatacenterID:  0,     // 默认数据中心ID
		WorkerID:      0,     // 默认工作机器ID
		EnableMetrics: false, // 默认关闭监控（性能优先）
	}

	// 步骤3：创建并注册到注册表
	generator, err := registry.GetOrCreate(DefaultGeneratorKey, core.GeneratorTypeSnowflake, config)

	if err != nil {
		log.Println("默认生成器创建失败", "error", err)
	} else {
		log.Println("默认生成器创建成功", "key", DefaultGeneratorKey)
	}

	return generator, err
}

// ResetDefaultGenerator 重置默认生成器
// 警告：此函数仅用于测试！
// 用途：在单元测试中重置默认生成器，避免测试间相互影响
func ResetDefaultGenerator() {
	defaultGeneratorOnce = sync.Once{}
	defaultGenerator = nil
	defaultGeneratorErr = nil

	// 日志建议：此处可添加日志记录
	log.Println("默认生成器已重置", "warning", "此操作仅用于测试")
}
