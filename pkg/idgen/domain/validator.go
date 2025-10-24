package domain

import (
	"fmt"
	"katydid-common-account/pkg/idgen/core"
	"katydid-common-account/pkg/idgen/registry"
)

// Validate 验证ID的有效性
// 说明：使用默认生成器类型（Snowflake）进行验证
func (id ID) Validate() error {
	return id.ValidateWithType(defaultGeneratorType)
}

// ValidateWithType 使用指定生成器类型验证ID
func (id ID) ValidateWithType(generatorType core.GeneratorType) error {
	if !generatorType.IsValid() {
		return fmt.Errorf("%w: %s", core.ErrInvalidGeneratorType, generatorType)
	}

	validator, err := registry.GetValidatorRegistry().Get(generatorType)
	if err != nil {
		return fmt.Errorf("failed to get validator: %w", err)
	}
	return validator.Validate(int64(id))
}
