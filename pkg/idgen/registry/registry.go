package registry

import (
	"fmt"
	"regexp"
	"sync"

	"katydid-common-account/pkg/idgen/core"
)

const (
	// 默认最大生成器数量
	defaultMaxGenerators = 100
	// 绝对最大生成器数量（硬性上限）
	absoluteMaxGenerators = 100_000
	// 键的最大长度
	maxKeyLength = 256
)

// 键的合法字符正则表达式（只允许字母、数字、下划线、连字符、点）
var keyFormatRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)

// Registry 生成器注册表（单例模式，管理生成器实例的生命周期）
type Registry struct {
	generators    map[string]core.IDGenerator
	maxGenerators int
	mu            sync.RWMutex
}

// globalRegistry 全局生成器注册表实例
var (
	globalRegistry *Registry
	registryOnce   sync.Once
)

// GetRegistry 获取全局生成器注册表（单例模式，线程安全）
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = &Registry{
			generators:    make(map[string]core.IDGenerator),
			maxGenerators: defaultMaxGenerators,
		}
	})
	return globalRegistry
}

// Create 创建并注册一个新的生成器
func (r *Registry) Create(key string, generatorType core.GeneratorType, config any) (core.IDGenerator, error) {
	// 验证参数
	if err := validateKey(key); err != nil {
		return nil, err
	}

	if !generatorType.IsValid() {
		return nil, fmt.Errorf("%w: %s", core.ErrInvalidGeneratorType, generatorType)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查是否已存在
	if _, exists := r.generators[key]; exists {
		return nil, fmt.Errorf("%w: key %s", core.ErrGeneratorAlreadyExists, key)
	}

	// 检查数量限制
	if len(r.generators) >= r.maxGenerators {
		return nil, fmt.Errorf("%w: current %d, max %d",
			core.ErrMaxGeneratorsReached, len(r.generators), r.maxGenerators)
	}

	// 从工厂注册表获取工厂
	factory, err := GetFactoryRegistry().Get(generatorType)
	if err != nil {
		return nil, err
	}

	// 使用工厂创建生成器
	generator, err := factory.Create(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create generator: %w", err)
	}

	// 注册生成器
	r.generators[key] = generator
	return generator, nil
}

// Get 获取已注册的生成器
func (r *Registry) Get(key string) (core.IDGenerator, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	generator, exists := r.generators[key]
	if !exists {
		return nil, fmt.Errorf("%w: key %s", core.ErrGeneratorNotFound, key)
	}

	return generator, nil
}

// GetOrCreate 获取生成器，如果不存在则创建
func (r *Registry) GetOrCreate(key string, generatorType core.GeneratorType, config any) (core.IDGenerator, error) {
	// 先尝试获取（使用读锁，性能更好）
	r.mu.RLock()
	if generator, exists := r.generators[key]; exists {
		r.mu.RUnlock()
		return generator, nil
	}
	r.mu.RUnlock()

	// 不存在则创建（Create方法内部会加写锁）
	return r.Create(key, generatorType, config)
}

// Has 检查生成器是否存在
func (r *Registry) Has(key string) bool {
	if err := validateKey(key); err != nil {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.generators[key]
	return exists
}

// Remove 移除生成器
func (r *Registry) Remove(key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.generators[key]; !exists {
		return fmt.Errorf("%w: key %s", core.ErrGeneratorNotFound, key)
	}

	delete(r.generators, key)
	return nil
}

// Clear 清空所有生成器
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.generators = make(map[string]core.IDGenerator)
}

// Count 获取生成器数量
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.generators)
}

// ListKeys 列出所有生成器的键
func (r *Registry) ListKeys() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]string, 0, len(r.generators))
	for key := range r.generators {
		keys = append(keys, key)
	}
	return keys
}

// SetMaxGenerators 设置最大生成器数量
func (r *Registry) SetMaxGenerators(max int) error {
	if max <= 0 {
		return fmt.Errorf("max generators must be positive")
	}

	// 添加绝对上限检查
	if max > absoluteMaxGenerators {
		return fmt.Errorf("max generators cannot exceed absolute limit %d", absoluteMaxGenerators)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.generators) > max {
		return fmt.Errorf("current generator count %d exceeds new max %d",
			len(r.generators), max)
	}

	r.maxGenerators = max
	return nil
}

// GetMaxGenerators 获取最大生成器数量
func (r *Registry) GetMaxGenerators() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.maxGenerators
}

// validateKey 验证键的有效性
func validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("%w: key cannot be empty", core.ErrInvalidKey)
	}

	if len(key) > maxKeyLength {
		return fmt.Errorf("%w: key too long (max %d), got %d",
			core.ErrInvalidKey, maxKeyLength, len(key))
	}

	// 验证键格式（只允许安全字符）
	if !keyFormatRegex.MatchString(key) {
		return fmt.Errorf("%w: key contains invalid characters", core.ErrInvalidKeyFormat)
	}

	return nil
}
