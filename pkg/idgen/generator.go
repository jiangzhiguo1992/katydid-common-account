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

	// ErrInvalidGeneratorKey 无效的生成器key
	ErrInvalidGeneratorKey = errors.New("invalid generator key: key cannot be empty")

	// ErrRegistryFull 注册表已满
	ErrRegistryFull = errors.New("registry full: maximum number of generators reached")
)

// GeneratorType 生成器类型定义
type GeneratorType string

const (
	// SnowflakeGeneratorType Snowflake算法生成器
	SnowflakeGeneratorType GeneratorType = "snowflake"

	// 预留其他生成器类型
	// UUIDGeneratorType     GeneratorType = "uuid"
	// ObjectIDGeneratorType GeneratorType = "objectid"

	// 默认最大生成器数量，防止内存泄漏
	defaultMaxGenerators = 1000
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
	mu            sync.RWMutex
	factories     map[GeneratorType]GeneratorFactory
	generators    map[string]IDGenerator // 缓存已创建的生成器实例
	maxGenerators int                    // 最大生成器数量限制
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
			factories:     make(map[GeneratorType]GeneratorFactory),
			generators:    make(map[string]IDGenerator),
			maxGenerators: defaultMaxGenerators,
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
	// 验证key不为空
	if key == "" {
		return nil, ErrInvalidGeneratorKey
	}

	// 验证generatorType不为空
	if generatorType == "" {
		return nil, ErrInvalidGeneratorType
	}

	// 添加config空值检查
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	// 第一次检查缓存（快速路径，使用读锁）
	r.mu.RLock()
	if gen, exists := r.generators[key]; exists {
		r.mu.RUnlock()
		return gen, nil
	}
	// 检查是否超过最大数量限制
	exceedsLimit := len(r.generators) >= r.maxGenerators
	r.mu.RUnlock()

	if exceedsLimit {
		return nil, fmt.Errorf("%w: current=%d, max=%d", ErrRegistryFull, len(r.generators), r.maxGenerators)
	}

	// 获取工厂（使用读锁）
	r.mu.RLock()
	factory, exists := r.factories[generatorType]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: type=%s", ErrGeneratorNotFound, generatorType)
	}

	// 创建生成器（在锁外执行，避免长时间持有锁）
	generator, err := factory.Create(generatorType, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create generator: %w", err)
	}

	// 验证创建的生成器不为nil
	if generator == nil {
		return nil, errors.New("factory created nil generator")
	}

	// 缓存生成器（双重检查锁定）
	r.mu.Lock()
	defer r.mu.Unlock()

	// 双重检查，避免并发创建
	if gen, exists := r.generators[key]; exists {
		return gen, nil
	}

	// 再次检查数量限制（双重检查）
	if len(r.generators) >= r.maxGenerators {
		return nil, fmt.Errorf("%w: current=%d, max=%d", ErrRegistryFull, len(r.generators), r.maxGenerators)
	}

	r.generators[key] = generator
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
	if key == "" {
		return nil, false
	}

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
	if key == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.generators, key)
}

// ClearGenerators 清空所有缓存的生成器
// 注意：此方法会清空所有生成器，谨慎使用
func (r *GeneratorRegistry) ClearGenerators() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.generators = make(map[string]IDGenerator)
}

// GetGeneratorCount 获取当前缓存的生成器数量
//
// 返回:
//
//	int: 生成器数量
func (r *GeneratorRegistry) GetGeneratorCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.generators)
}

// SetMaxGenerators 设置最大生成器数量限制
//
// 参数:
//
//	max: 最大数量（必须大于0）
//
// 返回:
//
//	error: 设置失败时返回错误
func (r *GeneratorRegistry) SetMaxGenerators(max int) error {
	if max <= 0 {
		return errors.New("max generators must be positive")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.maxGenerators = max
	return nil
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

// ListGeneratorKeys 列出所有已缓存的生成器key
//
// 返回:
//
//	[]string: 生成器key列表
func (r *GeneratorRegistry) ListGeneratorKeys() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]string, 0, len(r.generators))
	for k := range r.generators {
		keys = append(keys, k)
	}
	return keys
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

// GenerateIDs 使用默认生成器批量生成ID的便捷函数
// 适合需要一次性生成多个ID的场景
//
// 参数:
//
//	count: 要生成的ID数量，必须在 [1, 100000] 范围内
//
// 返回:
//
//	[]int64: 生成的ID列表
//	error: 生成失败时返回错误
//
// 示例:
//
//	ids, err := GenerateIDs(100)
func GenerateIDs(count int) ([]int64, error) {
	gen, err := GetDefaultGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to get default generator: %w", err)
	}
	return gen.NextIDBatch(count)
}
