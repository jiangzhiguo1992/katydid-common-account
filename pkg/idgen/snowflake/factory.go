package snowflake

import (
	"fmt"
	"katydid-common-account/pkg/idgen/core"
)

// Factory Snowflake生成器工厂
type Factory struct{}

// NewFactory 创建Snowflake工厂实例
// 说明：工厂本身是无状态的，可以创建多个实例或共享单个实例
func NewFactory() *Factory {
	return &Factory{}
}

// Create 创建Snowflake生成器实例
// 实现core.GeneratorFactory接口
func (f *Factory) Create(config any) (core.IGenerator, error) {
	// 类型断言：将通用配置转换为Snowflake配置
	sfConfig, ok := config.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type: expected *snowflake.Config, got %T", config)
	}

	// 使用snowflake包创建生成器
	// 注意：NewWithConfig内部会验证配置
	return NewWithConfig(sfConfig)
}
