package core

import (
	"strings"
	"sync"
)

var (
	// stringBuilderPool 字符串构建器对象池
	stringBuilderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}
)

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
