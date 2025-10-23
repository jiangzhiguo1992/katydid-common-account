package v2

import "sync"

// ============================================================================
// 错误收集器实现
// ============================================================================

// errorCollector 错误收集器
type errorCollector struct {
	errors ValidationErrors
	mu     sync.Mutex // 保证并发安全
}

// newErrorCollector 创建错误收集器
func newErrorCollector() *errorCollector {
	return &errorCollector{
		errors: make(ValidationErrors, 0, 10),
	}
}

// Report 报告错误
func (c *errorCollector) Report(namespace, tag, param string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 防止错误数量过多
	if len(c.errors) >= MaxValidationErrors {
		return
	}

	c.errors = append(c.errors, NewFieldError(namespace, tag, param))
}

// AddError 添加已构造的错误
func (c *errorCollector) AddError(err *FieldError) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.errors) >= MaxValidationErrors {
		return
	}

	c.errors = append(c.errors, err)
}

// HasErrors 是否有错误
func (c *errorCollector) HasErrors() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.errors) > 0
}

// GetErrors 获取所有错误
func (c *errorCollector) GetErrors() ValidationErrors {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 返回副本，避免外部修改
	result := make(ValidationErrors, len(c.errors))
	copy(result, c.errors)
	return result
}

// Clear 清空错误
func (c *errorCollector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors = c.errors[:0]
}

// ============================================================================
// 对象池优化 - 减少内存分配
// ============================================================================

var errorCollectorPool = sync.Pool{
	New: func() interface{} {
		return newErrorCollector()
	},
}

// getErrorCollector 从对象池获取错误收集器
func getErrorCollector() *errorCollector {
	collector := errorCollectorPool.Get().(*errorCollector)
	collector.Clear()
	return collector
}

// putErrorCollector 归还错误收集器到对象池
func putErrorCollector(collector *errorCollector) {
	if collector != nil {
		collector.Clear()
		errorCollectorPool.Put(collector)
	}
}
