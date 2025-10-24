package registry

import (
	"fmt"
	"katydid-common-account/pkg/idgen/snowflake"
	"log"
	"regexp"
	"sync"

	"katydid-common-account/pkg/idgen/core"
)

// init 初始化全局工厂注册表
// 说明：在包加载时自动执行，注册默认的工厂、解析器和验证器
func init() {
	// 注册Snowflake工厂
	_ = GetFactoryRegistry().Register(core.GeneratorTypeSnowflake, snowflake.NewFactory())
	// 注册Snowflake解析器
	_ = GetParserRegistry().Register(core.GeneratorTypeSnowflake, snowflake.NewParser())
	// 注册Snowflake验证器
	_ = GetValidatorRegistry().Register(core.GeneratorTypeSnowflake, snowflake.NewValidator())

	// ...注册其他的

	log.Println("ID生成器工厂初始化完成", "registered_types", []string{"snowflake"})
}

const (
	// defaultMaxGenerators 默认最大生成器数量
	// 说明：限制注册表中可存储的生成器数量，防止内存泄漏
	defaultMaxGenerators = 100

	// absoluteMaxGenerators 绝对最大生成器数量（硬性上限）
	// 说明：即使通过SetMaxGenerators也不能超过此限制
	// 目的：保护系统资源，防止恶意或错误配置
	absoluteMaxGenerators = 100_000

	// maxKeyLength 键的最大长度
	// 说明：限制key的长度，防止过长的key占用过多内存
	maxKeyLength = 256
)

// keyFormatRegex 键的合法字符正则表达式
// 允许字符：字母（a-z, A-Z）、数字（0-9）、下划线(_)、连字符(-)、点(.)
// 目的：防止注入攻击和特殊字符导致的问题
var keyFormatRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-.]+$`)

// Registry 生成器注册表
type Registry struct {
	generators    map[string]core.IDGenerator // 生成器映射表
	maxGenerators int                         // 最大生成器数量限制
	mu            sync.RWMutex                // 读写锁，保护并发访问
}

var (
	// globalRegistry 全局生成器注册表实例（单例）
	globalRegistry *Registry

	// registryOnce 确保注册表只初始化一次
	registryOnce sync.Once
)

// GetRegistry 获取全局生成器注册表
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
	// 步骤1：验证参数
	if err := validateKey(key); err != nil {
		return nil, err
	}

	if !generatorType.IsValid() {
		return nil, fmt.Errorf("%w: %s", core.ErrInvalidGeneratorType, generatorType)
	}

	// 步骤2：加写锁，保护注册表
	r.mu.Lock()
	defer r.mu.Unlock()

	// 步骤3：检查key是否已存在
	if _, exists := r.generators[key]; exists {
		return nil, fmt.Errorf("%w: key '%s'", core.ErrGeneratorAlreadyExists, key)
	}

	// 步骤4：检查数量限制
	if len(r.generators) >= r.maxGenerators {
		return nil, fmt.Errorf("%w: current %d, max %d",
			core.ErrMaxGeneratorsReached, len(r.generators), r.maxGenerators)
	}

	// 步骤5：从工厂注册表获取工厂
	factory, err := GetFactoryRegistry().Get(generatorType)
	if err != nil {
		return nil, err
	}

	// 步骤6：使用工厂创建生成器
	generator, err := factory.Create(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create generator: %w", err)
	}

	// 步骤7：注册生成器
	r.generators[key] = generator

	log.Println("生成器创建成功", "key", key, "type", generatorType)

	return generator, nil
}

// Get 获取已注册的生成器
func (r *Registry) Get(key string) (core.IDGenerator, error) {
	// 验证key
	if err := validateKey(key); err != nil {
		return nil, err
	}

	// 使用读锁，允许并发读取
	r.mu.RLock()
	defer r.mu.RUnlock()

	generator, exists := r.generators[key]
	if !exists {
		return nil, fmt.Errorf("%w: key '%s'", core.ErrGeneratorNotFound, key)
	}

	return generator, nil
}

// GetOrCreate 获取生成器，如果不存在则创建
func (r *Registry) GetOrCreate(key string, generatorType core.GeneratorType, config any) (core.IDGenerator, error) {
	// 步骤1：验证参数
	if err := validateKey(key); err != nil {
		return nil, err
	}

	if !generatorType.IsValid() {
		return nil, fmt.Errorf("%w: %s", core.ErrInvalidGeneratorType, generatorType)
	}

	// 步骤2：加写锁，保护注册表
	r.mu.Lock()
	defer r.mu.Unlock()

	// 步骤3：检查key是否已存在
	if generator, exists := r.generators[key]; exists {
		return generator, nil
	}

	// 步骤4：检查数量限制
	if len(r.generators) >= r.maxGenerators {
		return nil, fmt.Errorf("%w: current %d, max %d",
			core.ErrMaxGeneratorsReached, len(r.generators), r.maxGenerators)
	}

	// 步骤5：从工厂注册表获取工厂
	factory, err := GetFactoryRegistry().Get(generatorType)
	if err != nil {
		return nil, err
	}

	// 步骤6：使用工厂创建生成器
	generator, err := factory.Create(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create generator: %w", err)
	}

	// 步骤7：注册生成器
	r.generators[key] = generator

	log.Println("生成器创建成功", "key", key, "type", generatorType)

	return generator, nil
}

// Has 检查生成器是否存在
func (r *Registry) Has(key string) bool {
	// 验证失败直接返回false
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
	// 验证key
	if err := validateKey(key); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查是否存在
	if _, exists := r.generators[key]; !exists {
		return fmt.Errorf("%w: key '%s'", core.ErrGeneratorNotFound, key)
	}

	// 删除生成器
	delete(r.generators, key)

	log.Println("生成器已移除", "key", key)

	return nil
}

// Clear 清空所有生成器
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 创建新的map，让GC回收旧的map
	r.generators = make(map[string]core.IDGenerator)

	// 日志建议：此处可添加日志记录
	log.Println("注册表已清空", "操作", "Clear")
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
	// 验证参数
	if max <= 0 {
		return fmt.Errorf("max generators must be positive, got %d", max)
	}

	// 检查绝对上限
	if max > absoluteMaxGenerators {
		return fmt.Errorf("max generators cannot exceed absolute limit %d, got %d",
			absoluteMaxGenerators, max)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查当前数量是否已超过新的限制
	if len(r.generators) > max {
		return fmt.Errorf("current generator count %d exceeds new max %d",
			len(r.generators), max)
	}

	r.maxGenerators = max

	log.Println("注册表容量已调整", "new_max", max, "current_count", len(r.generators))

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
	// 规则1：不能为空
	if len(key) == 0 {
		return fmt.Errorf("%w: key cannot be empty", core.ErrInvalidKey)
	}

	// 规则2：长度限制
	if len(key) > maxKeyLength {
		return fmt.Errorf("%w: key too long (max %d), got %d",
			core.ErrInvalidKey, maxKeyLength, len(key))
	}

	// 规则3：格式验证（只允许安全字符）
	if !keyFormatRegex.MatchString(key) {
		return fmt.Errorf("%w: key '%s' contains invalid characters",
			core.ErrInvalidKeyFormat, key)
	}

	return nil
}
