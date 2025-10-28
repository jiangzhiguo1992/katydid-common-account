package v5_refactored

import "sync"

// ============================================================================
// 默认错误收集器
// ============================================================================

// DefaultErrorCollector 默认错误收集器（非线程安全）
// 职责：收集验证错误
// 设计原则：单一职责 - 只负责错误收集
type DefaultErrorCollector struct {
	// errors 错误列表
	errors []*FieldError

	// maxErrors 最大错误数
	maxErrors int

	// fieldIndex 字段索引（用于快速查找）
	fieldIndex map[string][]*FieldError
}

// NewDefaultErrorCollector 创建默认错误收集器
func NewDefaultErrorCollector(maxErrors int) *DefaultErrorCollector {
	if maxErrors <= 0 {
		maxErrors = 100 // 默认最大 100 个错误
	}

	return &DefaultErrorCollector{
		errors:     make([]*FieldError, 0, 8),
		maxErrors:  maxErrors,
		fieldIndex: make(map[string][]*FieldError),
	}
}

// Add 添加错误
func (c *DefaultErrorCollector) Add(err *FieldError) bool {
	if err == nil {
		return true
	}

	// 检查是否已满
	if c.IsFull() {
		return false
	}

	// 添加到列表
	c.errors = append(c.errors, err)

	// 更新字段索引
	if err.Field != "" {
		c.fieldIndex[err.Field] = append(c.fieldIndex[err.Field], err)
	}
	if err.Namespace != "" && err.Namespace != err.Field {
		c.fieldIndex[err.Namespace] = append(c.fieldIndex[err.Namespace], err)
	}

	return true
}

// GetAll 获取所有错误
func (c *DefaultErrorCollector) GetAll() []*FieldError {
	return c.errors
}

// GetByField 按字段获取错误
func (c *DefaultErrorCollector) GetByField(field string) []*FieldError {
	if errs, ok := c.fieldIndex[field]; ok {
		return errs
	}
	return nil
}

// HasErrors 是否有错误
func (c *DefaultErrorCollector) HasErrors() bool {
	return len(c.errors) > 0
}

// Count 错误数量
func (c *DefaultErrorCollector) Count() int {
	return len(c.errors)
}

// Clear 清空错误
func (c *DefaultErrorCollector) Clear() {
	c.errors = c.errors[:0]
	clear(c.fieldIndex)
}

// IsFull 是否已满
func (c *DefaultErrorCollector) IsFull() bool {
	return len(c.errors) >= c.maxErrors
}

// ============================================================================
// 并发安全的错误收集器
// ============================================================================

// ConcurrentErrorCollector 并发安全的错误收集器
// 职责：在并发场景下收集验证错误
// 设计原则：线程安全，适用于并发验证
type ConcurrentErrorCollector struct {
	// errors 错误列表
	errors []*FieldError

	// maxErrors 最大错误数
	maxErrors int

	// fieldIndex 字段索引
	fieldIndex map[string][]*FieldError

	// mu 互斥锁
	mu sync.RWMutex
}

// NewConcurrentErrorCollector 创建并发错误收集器
func NewConcurrentErrorCollector(maxErrors int) *ConcurrentErrorCollector {
	if maxErrors <= 0 {
		maxErrors = 100
	}

	return &ConcurrentErrorCollector{
		errors:     make([]*FieldError, 0, 8),
		maxErrors:  maxErrors,
		fieldIndex: make(map[string][]*FieldError),
	}
}

// Add 添加错误
func (c *ConcurrentErrorCollector) Add(err *FieldError) bool {
	if err == nil {
		return true
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否已满
	if len(c.errors) >= c.maxErrors {
		return false
	}

	// 添加到列表
	c.errors = append(c.errors, err)

	// 更新字段索引
	if err.Field != "" {
		c.fieldIndex[err.Field] = append(c.fieldIndex[err.Field], err)
	}
	if err.Namespace != "" && err.Namespace != err.Field {
		c.fieldIndex[err.Namespace] = append(c.fieldIndex[err.Namespace], err)
	}

	return true
}

// GetAll 获取所有错误
func (c *ConcurrentErrorCollector) GetAll() []*FieldError {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 返回副本，避免外部修改
	result := make([]*FieldError, len(c.errors))
	copy(result, c.errors)
	return result
}

// GetByField 按字段获取错误
func (c *ConcurrentErrorCollector) GetByField(field string) []*FieldError {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if errs, ok := c.fieldIndex[field]; ok {
		// 返回副本
		result := make([]*FieldError, len(errs))
		copy(result, errs)
		return result
	}
	return nil
}

// HasErrors 是否有错误
func (c *ConcurrentErrorCollector) HasErrors() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.errors) > 0
}

// Count 错误数量
func (c *ConcurrentErrorCollector) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.errors)
}

// Clear 清空错误
func (c *ConcurrentErrorCollector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.errors = c.errors[:0]
	clear(c.fieldIndex)
}

// IsFull 是否已满
func (c *ConcurrentErrorCollector) IsFull() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.errors) >= c.maxErrors
}

// ============================================================================
// 错误收集器工厂
// ============================================================================

// DefaultErrorCollectorFactory 默认错误收集器工厂
type DefaultErrorCollectorFactory struct {
	// concurrent 是否使用并发安全的收集器
	concurrent bool
}

// NewDefaultErrorCollectorFactory 创建默认错误收集器工厂
func NewDefaultErrorCollectorFactory(concurrent bool) *DefaultErrorCollectorFactory {
	return &DefaultErrorCollectorFactory{
		concurrent: concurrent,
	}
}

// Create 创建错误收集器
func (f *DefaultErrorCollectorFactory) Create(maxErrors int) ErrorCollector {
	if f.concurrent {
		return NewConcurrentErrorCollector(maxErrors)
	}
	return NewDefaultErrorCollector(maxErrors)
}

// ============================================================================
// 错误收集器池
// ============================================================================

var (
	// errorCollectorPool 错误收集器对象池
	errorCollectorPool = sync.Pool{
		New: func() interface{} {
			return NewDefaultErrorCollector(100)
		},
	}
)

// AcquireErrorCollector 从池中获取错误收集器
func AcquireErrorCollector(maxErrors int) ErrorCollector {
	collector := errorCollectorPool.Get().(*DefaultErrorCollector)
	collector.maxErrors = maxErrors
	collector.Clear()
	return collector
}

// ReleaseErrorCollector 释放错误收集器到池
func ReleaseErrorCollector(collector ErrorCollector) {
	if c, ok := collector.(*DefaultErrorCollector); ok {
		c.Clear()
		errorCollectorPool.Put(c)
	}
}
