package idgen

import (
	"errors"
	"fmt"
	"sync"
)

// SnowflakeFactory Snowflake生成器工厂
type SnowflakeFactory struct{}

// DefaultSnowflakeGenerator 默认的Snowflake生成器
// 使用datacenterID=0, workerID=0
var (
	defaultSnowflake     IDGenerator
	defaultSnowflakeOnce sync.Once
)

// Create 创建Snowflake生成器（实现GeneratorFactory接口）
func (f *SnowflakeFactory) Create(config any) (IDGenerator, error) {
	// 类型断言
	sfConfig, ok := config.(*SnowflakeConfig)
	if !ok {
		return nil, errors.New("invalid config type: expected *SnowflakeConfig")
	}

	return NewSnowflakeWithConfig(sfConfig)
}

// GetDefaultGenerator 获取默认的Snowflake生成器（适用于简单场景，不需要手动配置）
func GetDefaultGenerator() (IDGenerator, error) {
	var initErr error
	defaultSnowflakeOnce.Do(func() {
		var err error
		defaultSnowflake, err = NewSnowflake(0, 0)
		if err != nil {
			// 将错误保存到外部变量，避免闭包陷阱
			initErr = err
		}
	})

	// 如果初始化失败，返回错误
	if initErr != nil {
		return nil, initErr
	}

	// 如果生成器为 nil（理论上不应该发生，但增加防御性检查）
	if defaultSnowflake == nil {
		return nil, errors.New("default generator initialization failed")
	}

	return defaultSnowflake, nil
}

// GenerateID 使用默认生成器生成ID的便捷函数（这是最简单的使用方式，适合快速原型开发）
func GenerateID() (int64, error) {
	gen, err := GetDefaultGenerator()
	if err != nil {
		return 0, fmt.Errorf("failed to get default generator: %w", err)
	}
	return gen.NextID()
}

// GenerateIDs 使用默认生成器批量生成ID的便捷函数（适合需要一次性生成多个ID的场景）
func GenerateIDs(count int) ([]int64, error) {
	gen, err := GetDefaultGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to get default generator: %w", err)
	}
	return gen.NextIDBatch(count)
}
