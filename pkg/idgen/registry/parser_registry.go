package registry

import (
	"sync"

	"katydid-common-account/pkg/idgen/core"
)

// ParserRegistry 解析器注册表（单例模式，管理解析器实例）
// 遵循依赖倒置原则：通过接口管理解析器
type ParserRegistry struct {
	parsers map[core.GeneratorType]core.IDParser
	mu      sync.RWMutex
}

var (
	globalParserRegistry    *ParserRegistry
	parserRegistryOnce      sync.Once
	globalValidatorRegistry *ValidatorRegistry
	validatorRegistryOnce   sync.Once
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
	if !generatorType.IsValid() {
		return core.ErrInvalidGeneratorType
	}

	if parser == nil {
		return core.ErrNilConfig
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.parsers[generatorType] = parser
	return nil
}

// Get 获取解析器
func (r *ParserRegistry) Get(generatorType core.GeneratorType) (core.IDParser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	parser, exists := r.parsers[generatorType]
	if !exists {
		return nil, core.ErrGeneratorNotFound
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

// ValidatorRegistry 验证器注册表（单例模式）
type ValidatorRegistry struct {
	validators map[core.GeneratorType]core.IDValidator
	mu         sync.RWMutex
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
	if !generatorType.IsValid() {
		return core.ErrInvalidGeneratorType
	}

	if validator == nil {
		return core.ErrNilConfig
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.validators[generatorType] = validator
	return nil
}

// Get 获取验证器
func (r *ValidatorRegistry) Get(generatorType core.GeneratorType) (core.IDValidator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	validator, exists := r.validators[generatorType]
	if !exists {
		return nil, core.ErrGeneratorNotFound
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
