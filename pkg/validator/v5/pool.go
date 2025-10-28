package v5

import (
	"katydid-common-account/pkg/validator/v5/core"
	"strings"
	"sync"
)

var (
	// validationContextPool 验证上下文对象池
	validationContextPool = sync.Pool{
		New: func() interface{} {
			return &ValidationContext{
				//errors:   make([]*FieldError, 0, 4), // 预分配容量，减少扩容
				Metadata: make(map[string]any, 2), // 预分配容量
			}
		},
	}

	// stringBuilderPool 字符串构建器对象池
	stringBuilderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}
)

// NewValidationContext 创建验证上下文
func acquireValidationContext(scene core.Scene, maxErrors int) *ValidationContext {
	// 使用对象池优化内存分配
	ctx := validationContextPool.Get().(*ValidationContext)
	ctx.Scene = scene
	ctx.MaxErrors = maxErrors
	ctx.Depth = 0

	clear(ctx.errors)
	clear(ctx.Metadata)

	return ctx
}

// Release 释放验证上下文到对象池
// 使用完毕后应该调用此方法
func releaseContext(ctx *ValidationContext) {
	// 清空字段
	ctx.Context = nil
	ctx.Scene = core.SceneNone
	ctx.Depth = 0
	clear(ctx.errors)
	clear(ctx.Metadata)

	validationContextPool.Put(ctx)
}

// AcquireStringBuilder 从对象池获取字符串构建器
func AcquireStringBuilder() *strings.Builder {
	sb := stringBuilderPool.Get().(*strings.Builder)
	sb.Reset()
	return sb
}

// ReleaseStringBuilder 归还字符串构建器到对象池
func ReleaseStringBuilder(sb *strings.Builder) {
	if sb == nil {
		return
	}

	// 防止内存泄漏：不归还过大的Builder
	if sb.Cap() > 10*1024 { // 超过10KB
		return
	}

	sb.Reset()
	stringBuilderPool.Put(sb)
}
