package validator

import (
	"strings"
	"sync"
)

// ============================================================================
// 对象池优化 - 减少内存分配和 GC 压力
// ============================================================================

var (
	// validationContextPool ValidationContext 对象池
	// 用途：复用 ValidationContext 对象，减少频繁的内存分配
	// 线程安全：sync.Pool 是线程安全的
	validationContextPool = sync.Pool{
		New: func() interface{} {
			return &ValidationContext{
				Errors: make([]*FieldError, 0, 8), // 预分配8个错误容量
			}
		},
	}

	// stringBuilderPool strings.Builder 对象池
	// 用途：复用字符串构建器，减少字符串拼接时的内存分配
	stringBuilderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}
)

// acquireValidationContext 从对象池获取 ValidationContext
// 使用后必须调用 releaseValidationContext 归还
// 参数：
//   - scene: 验证场景标识
//
// 返回：
//   - 已重置的 ValidationContext 实例
func acquireValidationContext(scene ValidateScene) *ValidationContext {
	ctx := validationContextPool.Get().(*ValidationContext)
	ctx.Scene = scene
	ctx.Message = ""
	ctx.Errors = ctx.Errors[:0] // 清空错误列表，保留底层数组
	return ctx
}

// releaseValidationContext 将 ValidationContext 归还到对象池
// 参数：
//   - ctx: 待归还的 ValidationContext
func releaseValidationContext(ctx *ValidationContext) {
	if ctx == nil {
		return
	}

	// 防止内存泄漏：清空大容量的错误列表
	if cap(ctx.Errors) > 1000 {
		ctx.Errors = make([]*FieldError, 0, 8) // 重新分配小容量切片
	} else {
		// 清空错误引用，帮助 GC 回收
		for i := range ctx.Errors {
			ctx.Errors[i] = nil
		}
		ctx.Errors = ctx.Errors[:0]
	}

	// 清空字符串字段
	ctx.Message = ""

	validationContextPool.Put(ctx)
}

// acquireStringBuilder 从对象池获取 strings.Builder
// 使用后必须调用 releaseStringBuilder 归还
// 返回：
//   - 已重置的 strings.Builder 实例
func acquireStringBuilder() *strings.Builder {
	sb := stringBuilderPool.Get().(*strings.Builder)
	sb.Reset()
	return sb
}

// releaseStringBuilder 将 strings.Builder 归还到对象池
// 参数：
//   - sb: 待归还的 strings.Builder
func releaseStringBuilder(sb *strings.Builder) {
	if sb == nil {
		return
	}

	// 防止内存泄漏：重置过大的 Builder
	if sb.Cap() > 10*1024 { // 超过 10KB
		// 不归还到池中，让其被 GC 回收
		return
	}

	sb.Reset()
	stringBuilderPool.Put(sb)
}

// ============================================================================
// 内存安全检查
// ============================================================================

// checkMemorySafety 检查内存使用是否安全
// 用于防止恶意数据导致内存溢出
// 参数：
//   - errorCount: 当前错误数量
//   - valueSize: 值的大小
//
// 返回：
//   - true 表示安全，false 表示不安全
func checkMemorySafety(errorCount int, valueSize int) bool {
	// 检查错误数量
	if errorCount >= maxErrorsCapacity {
		return false
	}

	// 检查单个值大小
	if valueSize > maxValueSize {
		return false
	}

	return true
}

// ============================================================================
// 对象池统计 - 用于监控和调试
// ============================================================================

// PoolStats 对象池统计信息
type PoolStats struct {
	// ValidationContextPoolHits ValidationContext 池命中次数（估算）
	ValidationContextPoolHits int64
	// FieldErrorPoolHits FieldError 池命中次数（估算）
	FieldErrorPoolHits int64
	// StringBuilderPoolHits StringBuilder 池命中次数（估算）
	StringBuilderPoolHits int64
}

// GetPoolStats 获取对象池统计信息
// 注意：sync.Pool 不提供精确的统计信息，这里只是估算
// 返回：
//   - 对象池统计信息
func GetPoolStats() PoolStats {
	// sync.Pool 不提供统计信息，返回空结构
	// 实际生产环境可以使用自定义的带统计功能的对象池
	return PoolStats{}
}

// ResetPools 重置所有对象池
// 用于测试或需要强制清空对象池的场景
// 注意：此操作会影响性能，仅用于特殊场景
func ResetPools() {
	validationContextPool = sync.Pool{
		New: func() interface{} {
			return &ValidationContext{
				Errors: make([]*FieldError, 0, 8),
			}
		},
	}

	stringBuilderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}
}
