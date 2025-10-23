package v2

import "sync"

// ============================================================================
// 错误收集器实现 - 单一职责原则（SRP）
// ============================================================================

// errorCollector 错误收集器的默认实现
type errorCollector struct {
	errors []ValidationError
	mu     sync.Mutex // 支持并发安全
}

// NewErrorCollector 创建错误收集器（工厂方法）
func NewErrorCollector() ErrorCollector {
	return &errorCollector{
		errors: make([]ValidationError, 0, 8),
	}
}

// Add 添加单个错误
func (c *errorCollector) Add(err ValidationError) {
	if err == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.errors = append(c.errors, err)
}

// AddAll 批量添加错误
func (c *errorCollector) AddAll(errs []ValidationError) {
	if len(errs) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, err := range errs {
		if err != nil {
			c.errors = append(c.errors, err)
		}
	}
}

// HasErrors 检查是否有错误
func (c *errorCollector) HasErrors() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.errors) > 0
}

// GetAll 获取所有错误
func (c *errorCollector) GetAll() []ValidationError {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.errors) == 0 {
		return nil
	}

	// 返回副本，避免外部修改
	result := make([]ValidationError, len(c.errors))
	copy(result, c.errors)
	return result
}

// Count 获取错误数量
func (c *errorCollector) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.errors)
}

// Clear 清空错误
func (c *errorCollector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.errors = c.errors[:0]
}
