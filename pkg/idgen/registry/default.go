package registry

import (
	"fmt"
	"sync"

	"katydid-common-account/pkg/idgen/core"
	"katydid-common-account/pkg/idgen/snowflake"
)

const (
	// DefaultGeneratorKey 默认生成器的键
	DefaultGeneratorKey = "default"
)

var (
	// 默认生成器实例
	defaultGenerator     core.IDGenerator
	defaultGeneratorOnce sync.Once
	defaultGeneratorErr  error
)

// GetDefaultGenerator 获取默认的Snowflake生成器（单例模式）
// 使用默认配置：datacenterID=0, workerID=0
func GetDefaultGenerator() (core.IDGenerator, error) {
	defaultGeneratorOnce.Do(func() {
		// 创建默认配置
		config := &snowflake.Config{
			DatacenterID:  0,
			WorkerID:      0,
			EnableMetrics: false,
		}

		// 从工厂创建生成器
		factory := NewSnowflakeFactory()
		defaultGenerator, defaultGeneratorErr = factory.Create(config)
	})

	return defaultGenerator, defaultGeneratorErr
}

// GetOrCreateDefaultGenerator 获取或创建默认生成器（优先从注册表获取）
func GetOrCreateDefaultGenerator() (core.IDGenerator, error) {
	registry := GetRegistry()

	// 尝试从注册表获取
	if registry.Has(DefaultGeneratorKey) {
		return registry.Get(DefaultGeneratorKey)
	}

	// 创建默认配置
	config := &snowflake.Config{
		DatacenterID:  0,
		WorkerID:      0,
		EnableMetrics: false,
	}

	// 创建并注册
	return registry.GetOrCreate(DefaultGeneratorKey, core.GeneratorTypeSnowflake, config)
}

// ResetDefaultGenerator 重置默认生成器（仅用于测试）
func ResetDefaultGenerator() {
	defaultGeneratorOnce = sync.Once{}
	defaultGenerator = nil
	defaultGeneratorErr = nil
}
