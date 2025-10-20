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

	// ErrKeyTooLong key长度超限
	ErrKeyTooLong = errors.New("generator key too long")

	// ErrInvalidKeyCharacters key包含非法字符
	ErrInvalidKeyCharacters = errors.New("generator key contains invalid characters")
)

// GeneratorType 生成器类型定义
type GeneratorType string

const (
	// SnowflakeGeneratorType Snowflake算法生成器
	SnowflakeGeneratorType GeneratorType = "snowflake"

	// 预留其他生成器类型
	// UUIDGeneratorType     GeneratorType = "uuid"
	// ObjectIDGeneratorType GeneratorType = "objectId"

	// 默认最大生成器数量，防止内存泄漏
	defaultMaxGenerators = 1000

	// key长度限制，防止恶意超长key导致内存问题
	maxKeyLength = 256

	// 最小key长度
	minKeyLength = 1

	// 生成器类型最大长度
	maxGeneratorTypeLength = 64
)

// GeneratorFactory ID生成器工厂接口
type GeneratorFactory interface {
	Create(config any) (IDGenerator, error)
}

// GeneratorRegistry 生成器注册表（用于管理不同类型的ID生成器工厂）
type GeneratorRegistry struct {
	factories     map[GeneratorType]GeneratorFactory // 已注册的生成器工厂
	generators    map[string]IDGenerator             // 缓存已创建的生成器实例
	maxGenerators int                                // 最大生成器数量限制
	mu            sync.RWMutex
}

var (
	registryInstance *GeneratorRegistry // 生成器注册表单例
	registryOnce     sync.Once          // 单例初始化控制
)

// GetRegistry 获取生成器注册表的单例实例（线程安全）
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

// NewGenerator 便捷函数：创建ID生成器（简化生成器的创建过程）
func NewGenerator(key string, generatorType GeneratorType, config any) (IDGenerator, error) {
	return GetRegistry().CreateGenerator(key, generatorType, config)
}

// GetGeneratorFromRegistry 从注册表获取已缓存的生成器
func GetGeneratorFromRegistry(key string) (IDGenerator, bool) {
	return GetRegistry().GetGenerator(key)
}

// RegisterFactory 注册生成器工厂
func (r *GeneratorRegistry) RegisterFactory(generatorType GeneratorType, factory GeneratorFactory) error {
	if err := validateGeneratorType(generatorType); err != nil {
		return err
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

// CreateGenerator 创建ID生成器（如果已存在相同key的生成器，直接返回缓存的实例）
func (r *GeneratorRegistry) CreateGenerator(key string, generatorType GeneratorType, config any) (IDGenerator, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	// 使用generatorType验证函数
	if err := validateGeneratorType(generatorType); err != nil {
		return nil, err
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
	generator, err := factory.Create(config)
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
func (r *GeneratorRegistry) GetGenerator(key string) (IDGenerator, bool) {
	if err := validateKey(key); err != nil {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	gen, exists := r.generators[key]
	return gen, exists
}

// RemoveGenerator 移除已缓存的生成器
func (r *GeneratorRegistry) RemoveGenerator(key string) {
	if err := validateKey(key); err != nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.generators, key)
}

// ClearGenerators 清空所有缓存的生成器（注意：此方法会清空所有生成器，谨慎使用）
func (r *GeneratorRegistry) ClearGenerators() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 创建新map而不是遍历删除，更高效且避免潜在的map迭代问题
	r.generators = make(map[string]IDGenerator)
}

// GetGeneratorCount 获取当前缓存的生成器数量
func (r *GeneratorRegistry) GetGeneratorCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.generators)
}

// SetMaxGenerators 设置最大生成器数量限制
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
func (r *GeneratorRegistry) ListGeneratorKeys() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]string, 0, len(r.generators))
	for k := range r.generators {
		keys = append(keys, k)
	}
	return keys
}

// validateKey 验证生成器key的有效性
func validateKey(key string) error {
	if key == "" {
		return ErrInvalidGeneratorKey
	}

	// 安全优化：检查key长度
	if len(key) < minKeyLength {
		return fmt.Errorf("%w: minimum length is %d", ErrInvalidGeneratorKey, minKeyLength)
	}
	if len(key) > maxKeyLength {
		return fmt.Errorf("%w: maximum length is %d, got %d", ErrKeyTooLong, maxKeyLength, len(key))
	}

	// 安全优化：检查key是否包含控制字符或非法字符（防止注入攻击）
	for i, r := range key {
		// 只允许：字母、数字、下划线、连字符、点号
		isValid := (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-' || r == '.'

		if !isValid {
			return fmt.Errorf("%w: invalid character at position %d", ErrInvalidKeyCharacters, i)
		}
	}

	return nil
}

// validateGeneratorType 验证生成器类型的有效性
func validateGeneratorType(generatorType GeneratorType) error {
	if generatorType == "" {
		return ErrInvalidGeneratorType
	}

	if len(generatorType) > maxGeneratorTypeLength {
		return fmt.Errorf("%w: maximum length is %d", ErrInvalidGeneratorType, maxGeneratorTypeLength)
	}

	return nil
}
