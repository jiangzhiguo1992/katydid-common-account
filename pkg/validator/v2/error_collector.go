package v2

import (
	"fmt"
	"sync"
)

// ============================================================================
// 错误收集器实现 - 单一职责：专注于错误收集和管理
// ============================================================================

// defaultErrorCollector 默认错误收集器实现
type defaultErrorCollector struct {
	errors []ValidationError
	mu     sync.RWMutex // 支持并发安全
}

// NewErrorCollector 创建新的错误收集器
func NewErrorCollector() ErrorCollector {
	return &defaultErrorCollector{
		errors: make([]ValidationError, 0),
	}
}

// AddError 添加错误
func (c *defaultErrorCollector) AddError(field, tag string, params ...interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var param string
	var message string

	if len(params) > 0 {
		if p, ok := params[0].(string); ok {
			param = p
		}
	}
	if len(params) > 1 {
		if msg, ok := params[1].(string); ok {
			message = msg
		}
	}

	// 如果没有提供消息，生成默认消息
	if message == "" {
		message = c.generateDefaultMessage(field, tag, param)
	}

	c.errors = append(c.errors, ValidationError{
		Field:   field,
		Tag:     tag,
		Param:   param,
		Message: message,
	})
}

// AddFieldError 添加字段错误
func (c *defaultErrorCollector) AddFieldError(field, tag, param, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if message == "" {
		message = c.generateDefaultMessage(field, tag, param)
	}

	c.errors = append(c.errors, ValidationError{
		Field:   field,
		Tag:     tag,
		Param:   param,
		Message: message,
	})
}

// HasErrors 是否有错误
func (c *defaultErrorCollector) HasErrors() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.errors) > 0
}

// GetErrors 获取所有错误
func (c *defaultErrorCollector) GetErrors() ValidationErrors {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 返回副本，避免外部修改
	result := make(ValidationErrors, len(c.errors))
	copy(result, c.errors)
	return result
}

// Clear 清空错误
func (c *defaultErrorCollector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors = c.errors[:0]
}

// generateDefaultMessage 生成默认错误消息
func (c *defaultErrorCollector) generateDefaultMessage(field, tag, param string) string {
	switch tag {
	case "required":
		return fmt.Sprintf("字段 '%s' 是必填的", field)
	case "min":
		return fmt.Sprintf("字段 '%s' 的最小值/长度为 %s", field, param)
	case "max":
		return fmt.Sprintf("字段 '%s' 的最大值/长度为 %s", field, param)
	case "email":
		return fmt.Sprintf("字段 '%s' 必须是有效的邮箱地址", field)
	case "len":
		return fmt.Sprintf("字段 '%s' 的长度必须为 %s", field, param)
	case "eq":
		return fmt.Sprintf("字段 '%s' 必须等于 %s", field, param)
	case "ne":
		return fmt.Sprintf("字段 '%s' 不能等于 %s", field, param)
	case "gt":
		return fmt.Sprintf("字段 '%s' 必须大于 %s", field, param)
	case "gte":
		return fmt.Sprintf("字段 '%s' 必须大于等于 %s", field, param)
	case "lt":
		return fmt.Sprintf("字段 '%s' 必须小于 %s", field, param)
	case "lte":
		return fmt.Sprintf("字段 '%s' 必须小于等于 %s", field, param)
	case "alpha":
		return fmt.Sprintf("字段 '%s' 只能包含字母", field)
	case "alphanum":
		return fmt.Sprintf("字段 '%s' 只能包含字母和数字", field)
	case "numeric":
		return fmt.Sprintf("字段 '%s' 只能包含数字", field)
	case "url":
		return fmt.Sprintf("字段 '%s' 必须是有效的URL", field)
	case "uri":
		return fmt.Sprintf("字段 '%s' 必须是有效的URI", field)
	case "uuid":
		return fmt.Sprintf("字段 '%s' 必须是有效的UUID", field)
	case "oneof":
		return fmt.Sprintf("字段 '%s' 必须是以下值之一: %s", field, param)
	default:
		return fmt.Sprintf("字段 '%s' 验证失败: %s", field, tag)
	}
}

// ============================================================================
// 池化的错误收集器 - 性能优化
// ============================================================================

var errorCollectorPool = sync.Pool{
	New: func() interface{} {
		return &defaultErrorCollector{
			errors: make([]ValidationError, 0, 10), // 预分配容量
		}
	},
}

// GetPooledErrorCollector 从池中获取错误收集器
func GetPooledErrorCollector() ErrorCollector {
	collector := errorCollectorPool.Get().(*defaultErrorCollector)
	collector.Clear()
	return collector
}

// PutPooledErrorCollector 归还错误收集器到池
func PutPooledErrorCollector(collector ErrorCollector) {
	if c, ok := collector.(*defaultErrorCollector); ok {
		c.Clear()
		errorCollectorPool.Put(c)
	}
}
