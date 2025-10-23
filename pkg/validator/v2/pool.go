package v2

import (
	"strings"
	"sync"
)

// ============================================================================
// 对象池优化 - 减少内存分配和 GC 压力
// 设计模式：对象池模式（Object Pool Pattern）
// 设计原则：性能优化，减少重复的内存分配
// ============================================================================

var (
	// errorCollectorPool 错误收集器对象池
	// 用途：复用 ErrorCollector 对象，减少频繁的内存分配
	// 线程安全：sync.Pool 是线程安全的
	errorCollectorPool = sync.Pool{
		New: func() interface{} {
			return &DefaultErrorCollector{
				errors: make([]*FieldError, 0, 8), // 预分配8个错误容量
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

	// fieldErrorSlicePool FieldError 切片对象池
	// 用途：复用切片，减少动态扩容的开销
	fieldErrorSlicePool = sync.Pool{
		New: func() interface{} {
			slice := make([]*FieldError, 0, 8)
			return &slice
		},
	}
)

// ============================================================================
// ErrorCollector 池管理
// ============================================================================

// AcquireErrorCollector 从对象池获取 ErrorCollector
// 使用后必须调用 ReleaseErrorCollector 归还
// 设计原则：资源管理，确保资源正确回收
//
// 返回：已重置的 ErrorCollector 实例
func AcquireErrorCollector() ErrorCollector {
	collector := errorCollectorPool.Get().(*DefaultErrorCollector)
	collector.Clear() // 重置状态
	return collector
}

// ReleaseErrorCollector 将 ErrorCollector 归还到对象池
// 必须与 AcquireErrorCollector 配对使用，建议使用 defer
//
// 参数：
//   - collector: 待归还的 ErrorCollector
func ReleaseErrorCollector(collector ErrorCollector) {
	if collector == nil {
		return
	}

	// 类型断言为具体类型
	if dc, ok := collector.(*DefaultErrorCollector); ok {
		// 防止内存泄漏：清空大容量的错误列表
		if cap(dc.errors) > 100 {
			dc.errors = make([]*FieldError, 0, 8) // 重新分配小容量切片
		} else {
			// 清空错误引用，帮助 GC 回收
			for i := range dc.errors {
				dc.errors[i] = nil
			}
			dc.errors = dc.errors[:0]
		}

		errorCollectorPool.Put(dc)
	}
}

// ============================================================================
// StringBuilder 池管理
// ============================================================================

// AcquireStringBuilder 从对象池获取 strings.Builder
// 使用后必须调用 ReleaseStringBuilder 归还
//
// 返回：已重置的 strings.Builder 实例
func AcquireStringBuilder() *strings.Builder {
	sb := stringBuilderPool.Get().(*strings.Builder)
	sb.Reset()
	return sb
}

// ReleaseStringBuilder 将 strings.Builder 归还到对象池
//
// 参数：
//   - sb: 待归还的 strings.Builder
func ReleaseStringBuilder(sb *strings.Builder) {
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
// FieldError 切片池管理
// ============================================================================

// AcquireFieldErrorSlice 从对象池获取 FieldError 切片
// 使用后必须调用 ReleaseFieldErrorSlice 归还
//
// 返回：已重置的 FieldError 切片指针
func AcquireFieldErrorSlice() *[]*FieldError {
	slice := fieldErrorSlicePool.Get().(*[]*FieldError)
	*slice = (*slice)[:0] // 清空切片，保留底层数组
	return slice
}

// ReleaseFieldErrorSlice 将 FieldError 切片归还到对象池
//
// 参数：
//   - slice: 待归还的 FieldError 切片指针
func ReleaseFieldErrorSlice(slice *[]*FieldError) {
	if slice == nil {
		return
	}

	// 防止内存泄漏：清空大容量切片
	if cap(*slice) > 100 {
		newSlice := make([]*FieldError, 0, 8)
		fieldErrorSlicePool.Put(&newSlice)
	} else {
		// 清空引用，帮助 GC
		for i := range *slice {
			(*slice)[i] = nil
		}
		*slice = (*slice)[:0]
		fieldErrorSlicePool.Put(slice)
	}
}

// ============================================================================
// 池统计信息（用于监控和调试）
// ============================================================================

// PoolStats 对象池统计信息
type PoolStats struct {
	// ErrorCollectorPoolSize 错误收集器池中的对象数量（近似值）
	ErrorCollectorPoolSize int

	// StringBuilderPoolSize 字符串构建器池中的对象数量（近似值）
	StringBuilderPoolSize int

	// FieldErrorSlicePoolSize 错误切片池中的对象数量（近似值）
	FieldErrorSlicePoolSize int
}

// GetPoolStats 获取对象池统计信息
// 注意：sync.Pool 不提供精确的统计信息，这只是一个估算
//
// 返回：对象池统计信息
func GetPoolStats() PoolStats {
	// sync.Pool 不提供直接的统计接口
	// 这里只能返回一个空的统计信息
	// 实际的池大小由运行时动态管理
	return PoolStats{
		ErrorCollectorPoolSize:  0, // 无法精确获取
		StringBuilderPoolSize:   0, // 无法精确获取
		FieldErrorSlicePoolSize: 0, // 无法精确获取
	}
}

// ResetPools 重置所有对象池
// 警告：此操作会清空所有池中的对象，仅用于测试或特殊场景
func ResetPools() {
	errorCollectorPool = sync.Pool{
		New: func() interface{} {
			return &DefaultErrorCollector{
				errors: make([]*FieldError, 0, 8),
			}
		},
	}

	stringBuilderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}

	fieldErrorSlicePool = sync.Pool{
		New: func() interface{} {
			slice := make([]*FieldError, 0, 8)
			return &slice
		},
	}
}
