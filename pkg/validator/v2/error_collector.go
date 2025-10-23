package v2

import (
	"sync"
)

// ============================================================================
// ErrorCollector 实现 - 线程安全的错误收集器
// ============================================================================

// DefaultErrorCollector 默认错误收集器实现
// 设计原则：
//   - 单一职责：只负责错误的收集和管理
//   - 线程安全：使用互斥锁保护并发访问
type DefaultErrorCollector struct {
	errors []*FieldError
	mu     sync.RWMutex
}

// NewErrorCollector 创建错误收集器 - 工厂方法
func NewErrorCollector() *DefaultErrorCollector {
	return &DefaultErrorCollector{
		errors: make([]*FieldError, 0, 8), // 预分配容量
	}
}

// Report 报告一个验证错误 - 实现 ErrorReporter 接口
func (c *DefaultErrorCollector) Report(namespace, tag, param string) {
	c.ReportMsg(namespace, tag, param, "")
}

// ReportMsg 报告一个带自定义消息的验证错误
func (c *DefaultErrorCollector) ReportMsg(namespace, tag, param, message string) {
	// 从 namespace 中提取字段名
	field := extractFieldName(namespace)

	err := NewFieldError(namespace, field, tag, param)
	if message != "" {
		err = err.WithMessage(message)
	}

	c.Add(err)
}

// ReportWithValue 报告一个带值的验证错误
// 用于需要记录实际值的场景
func (c *DefaultErrorCollector) ReportWithValue(namespace, tag, param string, value any) {
	field := extractFieldName(namespace)
	err := NewFieldError(namespace, field, tag, param).WithValue(value)
	c.Add(err)
}

// ReportDetail 报告一个详细的验证错误
// 同时包含值和自定义消息
func (c *DefaultErrorCollector) ReportDetail(namespace, tag, param string, value any, message string) {
	field := extractFieldName(namespace)
	err := NewFieldError(namespace, field, tag, param).WithValue(value)
	if message != "" {
		err = err.WithMessage(message)
	}
	c.Add(err)
}

// Add 添加一个错误
func (c *DefaultErrorCollector) Add(err *FieldError) {
	if err == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 防止收集过多错误
	if len(c.errors) >= 1000 {
		return
	}

	c.errors = append(c.errors, err)
}

// AddAll 批量添加错误
func (c *DefaultErrorCollector) AddAll(errs []*FieldError) {
	if len(errs) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, err := range errs {
		if err == nil {
			continue
		}

		// 防止收集过多错误
		if len(c.errors) >= 1000 {
			break
		}

		c.errors = append(c.errors, err)
	}
}

// HasErrors 是否存在错误
func (c *DefaultErrorCollector) HasErrors() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.errors) > 0
}

// GetErrors 获取所有错误
func (c *DefaultErrorCollector) GetErrors() []*FieldError {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 返回副本，防止外部修改
	result := make([]*FieldError, len(c.errors))
	copy(result, c.errors)
	return result
}

// Clear 清空所有错误
func (c *DefaultErrorCollector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.errors = c.errors[:0]
}

// ============================================================================
// 辅助函数
// ============================================================================

// extractFieldName 从 namespace 中提取字段名
// 例如：User.Profile.Email -> Email
func extractFieldName(namespace string) string {
	if namespace == "" {
		return ""
	}

	// 从最后一个点号后面提取
	for i := len(namespace) - 1; i >= 0; i-- {
		if namespace[i] == '.' {
			return namespace[i+1:]
		}
	}

	return namespace
}
