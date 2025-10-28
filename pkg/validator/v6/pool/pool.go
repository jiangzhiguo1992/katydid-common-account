package pool

import (
	"sync"

	"katydid-common-account/pkg/validator/v6/collector"
	"katydid-common-account/pkg/validator/v6/context"
	"katydid-common-account/pkg/validator/v6/core"
)

// ValidationContextPool 验证上下文对象池
// 职责：管理 ValidationContext 对象的复用
// 设计模式：对象池模式
// 优势：减少内存分配，降低 GC 压力
type ValidationContextPool struct {
	pool sync.Pool
}

// NewValidationContextPool 创建验证上下文对象池
func NewValidationContextPool() *ValidationContextPool {
	return &ValidationContextPool{
		pool: sync.Pool{
			New: func() interface{} {
				// 创建新的 ValidationContext
				// 注意：这里创建的是零值，需要在 Get 时初始化
				return &validationContextWrapper{}
			},
		},
	}
}

// validationContextWrapper 上下文包装器
// 用于池化管理
type validationContextWrapper struct {
	ctx core.ValidationContext
}

// Get 从池中获取 ValidationContext
func (p *ValidationContextPool) Get(req *core.ValidationRequest, maxErrors int) core.ValidationContext {
	wrapper := p.pool.Get().(*validationContextWrapper)

	// 创建新的上下文（或重用）
	if wrapper.ctx == nil {
		wrapper.ctx = context.NewValidationContext(req, maxErrors)
	} else {
		// 重置上下文
		// 注意：这里需要确保上下文可以被重置
		wrapper.ctx = context.NewValidationContext(req, maxErrors)
	}

	return wrapper.ctx
}

// Put 将 ValidationContext 放回池中
func (p *ValidationContextPool) Put(ctx core.ValidationContext) {
	if ctx == nil {
		return
	}

	// 释放资源
	ctx.Release()

	// 放回池中
	wrapper := &validationContextWrapper{ctx: ctx}
	p.pool.Put(wrapper)
}

// ErrorCollectorPool 错误收集器对象池
// 职责：管理 ErrorCollector 对象的复用
type ErrorCollectorPool struct {
	pool sync.Pool
}

// NewErrorCollectorPool 创建错误收集器对象池
func NewErrorCollectorPool(maxErrors int) *ErrorCollectorPool {
	return &ErrorCollectorPool{
		pool: sync.Pool{
			New: func() interface{} {
				return collector.NewErrorCollector(maxErrors)
			},
		},
	}
}

// Get 从池中获取 ErrorCollector
func (p *ErrorCollectorPool) Get() core.ErrorCollector {
	ec := p.pool.Get().(core.ErrorCollector)
	ec.Clear() // 清空之前的错误
	return ec
}

// Put 将 ErrorCollector 放回池中
func (p *ErrorCollectorPool) Put(ec core.ErrorCollector) {
	if ec == nil {
		return
	}

	// 清空错误
	ec.Clear()

	// 放回池中
	p.pool.Put(ec)
}

// FieldErrorPool 字段错误对象池
// 职责：管理 FieldError 对象的复用
type FieldErrorPool struct {
	pool sync.Pool
}

// NewFieldErrorPool 创建字段错误对象池
func NewFieldErrorPool() *FieldErrorPool {
	return &FieldErrorPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &core.FieldError{}
			},
		},
	}
}

// Get 从池中获取 FieldError
func (p *FieldErrorPool) Get() *core.FieldError {
	err := p.pool.Get().(*core.FieldError)
	// 重置所有字段
	err.Namespace = ""
	err.Field = ""
	err.Tag = ""
	err.Param = ""
	err.Value = nil
	err.Message = ""
	return err
}

// Put 将 FieldError 放回池中
func (p *FieldErrorPool) Put(err *core.FieldError) {
	if err == nil {
		return
	}

	// 清空字段
	err.Namespace = ""
	err.Field = ""
	err.Tag = ""
	err.Param = ""
	err.Value = nil
	err.Message = ""

	// 放回池中
	p.pool.Put(err)
}

// GlobalPool 全局对象池管理器
// 职责：提供全局的对象池访问
var GlobalPool = struct {
	ValidationContext *ValidationContextPool
	ErrorCollector    *ErrorCollectorPool
	FieldError        *FieldErrorPool
}{
	ValidationContext: NewValidationContextPool(),
	ErrorCollector:    NewErrorCollectorPool(100),
	FieldError:        NewFieldErrorPool(),
}
