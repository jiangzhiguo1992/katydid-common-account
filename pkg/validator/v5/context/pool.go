package context

import (
	"katydid-common-account/pkg/validator/v5/core"
	"sync"
)

var (
	// validationContextPool 验证上下文对象池
	validationContextPool = sync.Pool{
		New: func() interface{} {
			return &ValidationContext{
				errors:   make([]core.IFieldError, 0, 4), // 预分配容量，减少扩容
				metadata: make(map[string]any, 2),        // 预分配容量
			}
		},
	}
)

// NewValidationContext 创建验证上下文
func acquireValidationContext(scene core.Scene, maxErrors int) *ValidationContext {
	// 使用对象池优化内存分配
	ctx := validationContextPool.Get().(*ValidationContext)
	ctx.scene = scene
	ctx.depth = 0
	clear(ctx.errors)
	clear(ctx.metadata)
	ctx.maxErrors = maxErrors

	return ctx
}

// releaseValidationContext 释放验证上下文到对象池
// 使用完毕后应该调用此方法
func releaseValidationContext(ctx *ValidationContext) {
	// 清空字段
	ctx.context = nil
	ctx.scene = core.SceneNone
	ctx.depth = 0
	clear(ctx.errors)
	clear(ctx.metadata)
	ctx.maxErrors = 0

	validationContextPool.Put(ctx)
}
