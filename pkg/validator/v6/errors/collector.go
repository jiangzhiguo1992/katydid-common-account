package errors

import (
	"katydid-common-account/pkg/validator/v6/core"
	"sync"
)

// ============================================================================
// 列表错误收集器 - 保持错误顺序
// ============================================================================

// listErrorCollector 基于列表的错误收集器
// 优点：保持错误顺序，适合顺序展示
// 缺点：查找特定字段错误较慢
type listErrorCollector struct {
	errors    []core.FieldError
	maxErrors int
}

// NewListErrorCollector 创建列表错误收集器
func NewListErrorCollector(maxErrors int) core.ErrorCollector {
	if maxErrors <= 0 {
		maxErrors = 100
	}
	return &listErrorCollector{
		errors:    make([]core.FieldError, 0, maxErrors),
		maxErrors: maxErrors,
	}
}

// Collect 收集错误
func (c *listErrorCollector) Collect(err core.FieldError) bool {
	if c.Count() >= c.maxErrors {
		return false
	}
	c.errors = append(c.errors, err)
	return true
}

// CollectAll 批量收集错误
func (c *listErrorCollector) CollectAll(errs []core.FieldError) bool {
	for _, err := range errs {
		if !c.Collect(err) {
			return false
		}
	}
	return true
}

// Errors 获取所有错误
func (c *listErrorCollector) Errors() []core.FieldError {
	return c.errors
}

// HasErrors 是否有错误
func (c *listErrorCollector) HasErrors() bool {
	return len(c.errors) > 0
}

// Count 错误数量
func (c *listErrorCollector) Count() int {
	return len(c.errors)
}

// Clear 清空错误
func (c *listErrorCollector) Clear() {
	c.errors = c.errors[:0]
}

// MaxErrors 最大错误数
func (c *listErrorCollector) MaxErrors() int {
	return c.maxErrors
}

// ============================================================================
// Map 错误收集器 - 按字段分组
// ============================================================================

// mapErrorCollector 基于 Map 的错误收集器
// 优点：按字段分组，便于查找特定字段错误
// 缺点：不保证错误顺序
type mapErrorCollector struct {
	errors    map[string][]core.FieldError
	count     int
	maxErrors int
}

// NewMapErrorCollector 创建 Map 错误收集器
func NewMapErrorCollector(maxErrors int) core.ErrorCollector {
	if maxErrors <= 0 {
		maxErrors = 100
	}
	return &mapErrorCollector{
		errors:    make(map[string][]core.FieldError),
		maxErrors: maxErrors,
	}
}

// Collect 收集错误
func (c *mapErrorCollector) Collect(err core.FieldError) bool {
	if c.count >= c.maxErrors {
		return false
	}

	field := err.Field()
	c.errors[field] = append(c.errors[field], err)
	c.count++
	return true
}

// CollectAll 批量收集错误
func (c *mapErrorCollector) CollectAll(errs []core.FieldError) bool {
	for _, err := range errs {
		if !c.Collect(err) {
			return false
		}
	}
	return true
}

// Errors 获取所有错误（展平）
func (c *mapErrorCollector) Errors() []core.FieldError {
	result := make([]core.FieldError, 0, c.count)
	for _, errs := range c.errors {
		result = append(result, errs...)
	}
	return result
}

// ErrorsByField 获取指定字段的错误
func (c *mapErrorCollector) ErrorsByField(field string) []core.FieldError {
	return c.errors[field]
}

// HasErrors 是否有错误
func (c *mapErrorCollector) HasErrors() bool {
	return c.count > 0
}

// Count 错误数量
func (c *mapErrorCollector) Count() int {
	return c.count
}

// Clear 清空错误
func (c *mapErrorCollector) Clear() {
	c.errors = make(map[string][]core.FieldError)
	c.count = 0
}

// MaxErrors 最大错误数
func (c *mapErrorCollector) MaxErrors() int {
	return c.maxErrors
}

// ============================================================================
// 错误收集器对象池
// ============================================================================

var (
	listCollectorPool = sync.Pool{
		New: func() any {
			return &listErrorCollector{
				errors:    make([]core.FieldError, 0, 10),
				maxErrors: 100,
			}
		},
	}

	mapCollectorPool = sync.Pool{
		New: func() any {
			return &mapErrorCollector{
				errors:    make(map[string][]core.FieldError),
				maxErrors: 100,
			}
		},
	}
)

// AcquireListCollector 从对象池获取列表收集器
func AcquireListCollector(maxErrors int) core.ErrorCollector {
	c := listCollectorPool.Get().(*listErrorCollector)
	c.maxErrors = maxErrors
	c.Clear()
	return c
}

// ReleaseListCollector 释放列表收集器到对象池
func ReleaseListCollector(collector core.ErrorCollector) {
	if c, ok := collector.(*listErrorCollector); ok {
		c.Clear()
		listCollectorPool.Put(c)
	}
}

// AcquireMapCollector 从对象池获取 Map 收集器
func AcquireMapCollector(maxErrors int) core.ErrorCollector {
	c := mapCollectorPool.Get().(*mapErrorCollector)
	c.maxErrors = maxErrors
	c.Clear()
	return c
}

// ReleaseMapCollector 释放 Map 收集器到对象池
func ReleaseMapCollector(collector core.ErrorCollector) {
	if c, ok := collector.(*mapErrorCollector); ok {
		c.Clear()
		mapCollectorPool.Put(c)
	}
}
