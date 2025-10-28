package collector

import (
	"sync"

	"katydid-common-account/pkg/validator/v6/core"
)

// ErrorCollectorImpl 错误收集器实现
// 职责：收集和管理验证错误
// 设计原则：单一职责、线程安全（如果需要）
type ErrorCollectorImpl struct {
	errors    []*core.FieldError
	maxErrors int
	mu        sync.RWMutex // 如果需要并发安全
}

// NewErrorCollector 创建错误收集器
func NewErrorCollector(maxErrors int) core.ErrorCollector {
	return &ErrorCollectorImpl{
		errors:    make([]*core.FieldError, 0, 4), // 预分配容量
		maxErrors: maxErrors,
	}
}

// Add 添加错误
// 返回 false 表示已达到最大错误数
func (c *ErrorCollectorImpl) Add(err *core.FieldError) bool {
	if err == nil {
		return true
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否已达到最大错误数
	if len(c.errors) >= c.maxErrors {
		return false
	}

	c.errors = append(c.errors, err)
	return true
}

// AddAll 批量添加错误
func (c *ErrorCollectorImpl) AddAll(errs []*core.FieldError) {
	if len(errs) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, err := range errs {
		if len(c.errors) >= c.maxErrors {
			break
		}
		if err != nil {
			c.errors = append(c.errors, err)
		}
	}
}

// GetAll 获取所有错误
func (c *ErrorCollectorImpl) GetAll() []*core.FieldError {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 返回副本，避免外部修改
	result := make([]*core.FieldError, len(c.errors))
	copy(result, c.errors)
	return result
}

// HasErrors 是否有错误
func (c *ErrorCollectorImpl) HasErrors() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.errors) > 0
}

// Count 错误数量
func (c *ErrorCollectorImpl) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.errors)
}

// Clear 清空错误
func (c *ErrorCollectorImpl) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.errors = c.errors[:0] // 保留底层数组
}

// SetMaxErrors 设置最大错误数
func (c *ErrorCollectorImpl) SetMaxErrors(max int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if max > 0 {
		c.maxErrors = max
	}
}
