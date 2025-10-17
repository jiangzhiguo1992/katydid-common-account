package idgen

import (
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrGeneratorNotFound 找不到指定的生成器
	ErrGeneratorNotFound = errors.New("generator not found: the specified generator type is not registered")

	// ErrGeneratorAlreadyExists 生成器已存在
	ErrGeneratorAlreadyExists = errors.New("generator already exists: cannot register duplicate generator")

	// ErrInvalidGeneratorType 无效的生成器类型
	ErrInvalidGeneratorType = errors.New("invalid generator type: generator type cannot be empty")
)

// GeneratorType 生成器类型定义
type GeneratorType string

const (
	// SnowflakeGeneratorType Snowflake算法生成器
	SnowflakeGeneratorType GeneratorType = "snowflake"

	// 预留其他生成器类型
	// UUIDGeneratorType     GeneratorType = "uuid"
	// ObjectIDGeneratorType GeneratorType = "objectid"
)

// GeneratorFactory ID生成器工厂接口（抽象工厂模式）
// 遵循开放封闭原则，支持扩展新的生成器类型
type GeneratorFactory interface {
	// Create 创建指定类型的ID生成器
	Create(generatorType GeneratorType, config interface{}) (IDGenerator, error)
}

// GeneratorRegistry 生成器注册表（单例模式）
// 用于管理不同类型的ID生成器工厂
type GeneratorRegistry struct {
	mu         sync.RWMutex
	factories  map[GeneratorType]GeneratorFactory
	generators map[string]IDGenerator // 缓存已创建的生成器实例
}

var (
	// 全局单例实例
	registryInstance *GeneratorRegistry
	registryOnce     sync.Once
)

// GetRegistry 获取生成器注册表的单例实例
// 线程安全，使用sync.Once确保只初始化一次
//
// 返回:
//
//	*GeneratorRegistry: 注册表单例实例
func GetRegistry() *GeneratorRegistry {
	registryOnce.Do(func() {
		registryInstance = &GeneratorRegistry{
			factories:  make(map[GeneratorType]GeneratorFactory),
			generators: make(map[string]IDGenerator),
		}
		// 默认注册Snowflake工厂
		// 忽略错误，因为这是初始化时的内部注册，必然成功
		_ = registryInstance.RegisterFactory(SnowflakeGeneratorType, &SnowflakeFactory{})
	})
	return registryInstance
}

// RegisterFactory 注册生成器工厂
//
// 参数:
//
//	generatorType: 生成器类型
//	factory: 生成器工厂实例
//
// 返回:
//
//	error: 注册失败时返回错误
func (r *GeneratorRegistry) RegisterFactory(generatorType GeneratorType, factory GeneratorFactory) error {
	if generatorType == "" {
		return ErrInvalidGeneratorType
	}

	if factory == nil {
		return errors.New("factory cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[generatorType]; exists {
		return fmt.Errorf("%w: type=%s", ErrGeneratorAlreadyExists, generatorType)
	}

	r.factories[generatorType] = factory
	return nil
}

// CreateGenerator 创建ID生成器（工厂方法）
// 如果已存在相同key的生成器，直接返回缓存的实例
//
// 参数:
//
//	key: 生成器唯一标识（用于缓存）
//	generatorType: 生成器类型
//	config: 生成器配置
//
// 返回:
//
//	IDGenerator: 生成器实例
//	error: 创建失败时返回错误
func (r *GeneratorRegistry) CreateGenerator(key string, generatorType GeneratorType, config interface{}) (IDGenerator, error) {
	// 检查缓存
	r.mu.RLock()
	if gen, exists := r.generators[key]; exists {
		r.mu.RUnlock()
		return gen, nil
	}
	r.mu.RUnlock()

	// 获取工厂
	r.mu.RLock()
	factory, exists := r.factories[generatorType]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: type=%s", ErrGeneratorNotFound, generatorType)
	}

	// 创建生成器
	generator, err := factory.Create(generatorType, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create generator: %w", err)
	}

	// 缓存生成器
	r.mu.Lock()
	// 双重检查，避免并发创建
	if gen, exists := r.generators[key]; exists {
		r.mu.Unlock()
		return gen, nil
	}
	r.generators[key] = generator
	r.mu.Unlock()

	return generator, nil
}

// GetGenerator 获取已缓存的生成器
//
// 参数:
//
//	key: 生成器唯一标识
//
// 返回:
//
//	IDGenerator: 生成器实例
//	bool: 是否找到
func (r *GeneratorRegistry) GetGenerator(key string) (IDGenerator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	gen, exists := r.generators[key]
	return gen, exists
}

// RemoveGenerator 移除已缓存的生成器
//
// 参数:
//
//	key: 生成器唯一标识
func (r *GeneratorRegistry) RemoveGenerator(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.generators, key)
}

// ListGeneratorTypes 列出所有已注册的生成器类型
//
// 返回:
//
//	[]GeneratorType: 生成器类型列表
func (r *GeneratorRegistry) ListGeneratorTypes() []GeneratorType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]GeneratorType, 0, len(r.factories))
	for t := range r.factories {
		types = append(types, t)
	}
	return types
}

// SnowflakeFactory Snowflake生成器工厂
type SnowflakeFactory struct{}

// Create 创建Snowflake生成器
// 实现GeneratorFactory接口
//
// 参数:
//
//	generatorType: 生成器类型（必须为SnowflakeGeneratorType）
//	config: 配置对象（*SnowflakeConfig类型）
//
// 返回:
//
//	IDGenerator: Snowflake生成器实例
//	error: 创建失败时返回错误
func (f *SnowflakeFactory) Create(generatorType GeneratorType, config interface{}) (IDGenerator, error) {
	if generatorType != SnowflakeGeneratorType {
		return nil, fmt.Errorf("unsupported generator type: %s", generatorType)
	}

	// 类型断言
	sfConfig, ok := config.(*SnowflakeConfig)
	if !ok {
		return nil, errors.New("invalid config type: expected *SnowflakeConfig")
	}

	return NewSnowflakeWithConfig(sfConfig)
}

// NewGenerator 便捷函数：创建ID生成器
// 这是一个全局函数，简化生成器的创建过程
//
// 参数:
//
//	key: 生成器唯一标识
//	generatorType: 生成器类型
//	config: 生成器配置
//
// 返回:
//
//	IDGenerator: 生成器实例
//	error: 创建失败时返回错误
//
// 示例:
//
//	gen, err := NewGenerator("server-1", SnowflakeGeneratorType, &SnowflakeConfig{
//	    DatacenterID: 1,
//	    WorkerID: 1,
//	})
func NewGenerator(key string, generatorType GeneratorType, config interface{}) (IDGenerator, error) {
	return GetRegistry().CreateGenerator(key, generatorType, config)
}

// GetGeneratorFromRegistry 从注册表获取已缓存的生成器
// 这是一个全局函数
//
// 参数:
//
//	key: 生成器唯一标识
//
// 返回:
//
//	IDGenerator: 生成器实例
//	bool: 是否找到
func GetGeneratorFromRegistry(key string) (IDGenerator, bool) {
	return GetRegistry().GetGenerator(key)
}

// DefaultSnowflakeGenerator 默认的Snowflake生成器
// 使用datacenterID=0, workerID=0
var defaultSnowflake IDGenerator
var defaultSnowflakeOnce sync.Once

// GetDefaultGenerator 获取默认的Snowflake生成器（单例）
// 适用于简单场景，不需要手动配置
//
// 返回:
//
//	IDGenerator: 默认生成器实例
//	error: 创建失败时返回错误
func GetDefaultGenerator() (IDGenerator, error) {
	var err error
	defaultSnowflakeOnce.Do(func() {
		defaultSnowflake, err = NewSnowflake(0, 0)
	})
	return defaultSnowflake, err
}

// GenerateID 使用默认生成器生成ID的便捷函数
// 这是最简单的使用方式，适合快速原型开发
//
// 返回:
//
//	int64: 生成的ID
//	error: 生成失败时返回错误
//
// 示例:
//
//	id, err := GenerateID()
func GenerateID() (int64, error) {
	gen, err := GetDefaultGenerator()
	if err != nil {
		return 0, fmt.Errorf("failed to get default generator: %w", err)
	}
	return gen.NextID()
}
