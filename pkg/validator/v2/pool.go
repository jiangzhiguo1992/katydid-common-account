package v2

import (
	"sync"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// 验证器对象池 - 单一职责：对象复用，减少GC压力
// ============================================================================

// defaultValidatorPool 默认验证器池
type defaultValidatorPool struct {
	pool *sync.Pool
}

// NewValidatorPool 创建验证器池
func NewValidatorPool() ValidatorPool {
	return &defaultValidatorPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return validator.New()
			},
		},
	}
}

// Get 从池中获取验证器
func (p *defaultValidatorPool) Get() *validator.Validate {
	return p.pool.Get().(*validator.Validate)
}

// Put 归还验证器到池
func (p *defaultValidatorPool) Put(v *validator.Validate) {
	p.pool.Put(v)
}

// ============================================================================
// 带初始化功能的验证器池 - 支持自定义配置
// ============================================================================

// InitFunc 验证器初始化函数
type InitFunc func(*validator.Validate)

// ConfigurableValidatorPool 可配置的验证器池
type ConfigurableValidatorPool struct {
	pool     *sync.Pool
	initFunc InitFunc
}

// NewConfigurableValidatorPool 创建可配置验证器池
func NewConfigurableValidatorPool(initFunc InitFunc) ValidatorPool {
	return &ConfigurableValidatorPool{
		initFunc: initFunc,
		pool: &sync.Pool{
			New: func() interface{} {
				v := validator.New()
				if initFunc != nil {
					initFunc(v)
				}
				return v
			},
		},
	}
}

// Get 从池中获取验证器
func (p *ConfigurableValidatorPool) Get() *validator.Validate {
	return p.pool.Get().(*validator.Validate)
}

// Put 归还验证器到池
func (p *ConfigurableValidatorPool) Put(v *validator.Validate) {
	p.pool.Put(v)
}
