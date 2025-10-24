package registry

import (
	"fmt"
	"katydid-common-account/pkg/idgen/core"
	"log"
	"sync"
)

// FactoryRegistry 工厂注册表
type FactoryRegistry struct {
	factories map[core.GeneratorType]core.GeneratorFactory // 工厂映射表
	mu        sync.RWMutex                                 // 读写锁，保护并发访问
}

var (
	// globalFactoryRegistry 全局工厂注册表实例（单例）
	globalFactoryRegistry *FactoryRegistry

	// factoryRegistryOnce 确保工厂注册表只初始化一次
	factoryRegistryOnce sync.Once
)

// GetFactoryRegistry 获取全局工厂注册表
func GetFactoryRegistry() *FactoryRegistry {
	factoryRegistryOnce.Do(func() {
		globalFactoryRegistry = &FactoryRegistry{
			factories: make(map[core.GeneratorType]core.GeneratorFactory),
		}
	})
	return globalFactoryRegistry
}

// Register 注册工厂
func (r *FactoryRegistry) Register(generatorType core.GeneratorType, factory core.GeneratorFactory) error {
	// 验证生成器类型
	if !generatorType.IsValid() {
		return fmt.Errorf("%w: %s", core.ErrInvalidGeneratorType, generatorType)
	}

	// 验证工厂不为nil
	if factory == nil {
		return fmt.Errorf("factory cannot be nil")
	}

	// 注册工厂（允许覆盖已有工厂）
	r.factories[generatorType] = factory

	log.Println("工厂已注册", "type", generatorType)

	return nil
}

// Get 获取工厂
func (r *FactoryRegistry) Get(generatorType core.GeneratorType) (core.GeneratorFactory, error) {
	factory, exists := r.factories[generatorType]
	if !exists {
		return nil, fmt.Errorf("%w: %s", core.ErrFactoryNotFound, generatorType)
	}
	return factory, nil
}

// Has 检查工厂是否存在
func (r *FactoryRegistry) Has(generatorType core.GeneratorType) bool {
	_, exists := r.factories[generatorType]
	return exists
}

// List 列出所有已注册的工厂类型
func (r *FactoryRegistry) List() []core.GeneratorType {
	types := make([]core.GeneratorType, 0, len(r.factories))
	for t := range r.factories {
		types = append(types, t)
	}
	return types
}
