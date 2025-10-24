package registry

import (
	"fmt"
	"katydid-common-account/pkg/idgen/core"
	"log"
	"sync"
)

// ValidatorRegistry 验证器注册表
type ValidatorRegistry struct {
	validators map[core.GeneratorType]core.IDValidator // 验证器映射表
	mu         sync.RWMutex                            // 读写锁，保护并发访问
}

var (
	// globalValidatorRegistry 全局验证器注册表实例（单例）
	globalValidatorRegistry *ValidatorRegistry

	// validatorRegistryOnce 确保验证器注册表只初始化一次
	validatorRegistryOnce sync.Once
)

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
