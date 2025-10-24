package registry

import (
	"fmt"
	"katydid-common-account/pkg/idgen/core"
	"katydid-common-account/pkg/idgen/snowflake"
	"log"
)

// init 初始化全局工厂注册表
// 说明：在包加载时自动执行，注册默认的工厂、解析器和验证器
func init() {
	// 创建工厂注册表
	globalFactoryRegistry = &FactoryRegistry{
		factories: make(map[core.GeneratorType]core.GeneratorFactory),
	}

	// 注册Snowflake工厂
	_ = GetFactoryRegistry().Register(core.GeneratorTypeSnowflake, snowflake.NewFactory())

	// 注册Snowflake解析器和验证器
	_ = GetParserRegistry().Register(core.GeneratorTypeSnowflake, snowflake.NewParser())
	_ = GetValidatorRegistry().Register(core.GeneratorTypeSnowflake, snowflake.NewValidator())

	log.Println("ID生成器工厂初始化完成", "registered_types", []string{"snowflake"})

	// ...注册其他的
}

// FactoryRegistry 工厂注册表
type FactoryRegistry struct {
	factories map[core.GeneratorType]core.GeneratorFactory // 工厂映射表
}

// globalFactoryRegistry 全局工厂注册表实例（单例）
var globalFactoryRegistry *FactoryRegistry

// GetFactoryRegistry 获取全局工厂注册表
func GetFactoryRegistry() *FactoryRegistry {
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
