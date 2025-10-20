package registry

import (
	"fmt"
	"log"
	"sync"

	"katydid-common-account/pkg/idgen/core"
)

// ParserRegistry 解析器注册表
type ParserRegistry struct {
	parsers map[core.GeneratorType]core.IDParser // 解析器映射表
	mu      sync.RWMutex                         // 读写锁，保护并发访问
}

var (
	// globalParserRegistry 全局解析器注册表实例（单例）
	globalParserRegistry *ParserRegistry

	// parserRegistryOnce 确保解析器注册表只初始化一次
	parserRegistryOnce sync.Once

	// globalValidatorRegistry 全局验证器注册表实例（单例）
	globalValidatorRegistry *ValidatorRegistry

	// validatorRegistryOnce 确保验证器注册表只初始化一次
	validatorRegistryOnce sync.Once
)

// GetParserRegistry 获取全局解析器注册表
func GetParserRegistry() *ParserRegistry {
	parserRegistryOnce.Do(func() {
		globalParserRegistry = &ParserRegistry{
			parsers: make(map[core.GeneratorType]core.IDParser),
		}
	})
	return globalParserRegistry
}

// Register 注册解析器
func (r *ParserRegistry) Register(generatorType core.GeneratorType, parser core.IDParser) error {
	// 验证生成器类型
	if !generatorType.IsValid() {
		return core.ErrInvalidGeneratorType
	}

	// 验证解析器不为nil
	if parser == nil {
		return fmt.Errorf("parser cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// 注册解析器（允许覆盖已有解析器）
	r.parsers[generatorType] = parser

	log.Println("解析器已注册", "type", generatorType)

	return nil
}

// Get 获取解析器
func (r *ParserRegistry) Get(generatorType core.GeneratorType) (core.IDParser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	parser, exists := r.parsers[generatorType]
	if !exists {
		return nil, core.ErrParserNotFound
	}

	return parser, nil
}

// Has 检查解析器是否存在
func (r *ParserRegistry) Has(generatorType core.GeneratorType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.parsers[generatorType]
	return exists
}

// ValidatorRegistry 验证器注册表
type ValidatorRegistry struct {
	validators map[core.GeneratorType]core.IDValidator // 验证器映射表
	mu         sync.RWMutex                            // 读写锁，保护并发访问
}

// GetValidatorRegistry 获取全局验证器注册表
func GetValidatorRegistry() *ValidatorRegistry {
	validatorRegistryOnce.Do(func() {
		globalValidatorRegistry = &ValidatorRegistry{
			validators: make(map[core.GeneratorType]core.IDValidator),
		}
	})
	return globalValidatorRegistry
}

// Register 注册验证器
func (r *ValidatorRegistry) Register(generatorType core.GeneratorType, validator core.IDValidator) error {
	// 验证生成器类型
	if !generatorType.IsValid() {
		return core.ErrInvalidGeneratorType
	}

	// 验证验证器不为nil
	if validator == nil {
		return fmt.Errorf("validator cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// 注册验证器（允许覆盖已有验证器）
	r.validators[generatorType] = validator

	log.Println("验证器已注册", "type", generatorType)

	return nil
}

// Get 获取验证器
func (r *ValidatorRegistry) Get(generatorType core.GeneratorType) (core.IDValidator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	validator, exists := r.validators[generatorType]
	if !exists {
		return nil, core.ErrValidatorNotFound
	}

	return validator, nil
}

// Has 检查验证器是否存在
func (r *ValidatorRegistry) Has(generatorType core.GeneratorType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.validators[generatorType]
	return exists
}
