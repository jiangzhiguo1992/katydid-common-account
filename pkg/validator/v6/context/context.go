package context

import (
	"sync"

	"katydid-common-account/pkg/validator/v6/collector"
	"katydid-common-account/pkg/validator/v6/core"
)

// ValidationContextImpl 验证上下文实现
// 职责：携带验证过程中的上下文信息
// 设计原则：上下文对象模式
type ValidationContextImpl struct {
	request        *core.ValidationRequest
	errorCollector core.ErrorCollector
	depth          int
	data           map[string]any
	mu             sync.RWMutex
}

// NewValidationContext 创建验证上下文
func NewValidationContext(req *core.ValidationRequest, maxErrors int) core.ValidationContext {
	return &ValidationContextImpl{
		request:        req,
		errorCollector: collector.NewErrorCollector(maxErrors),
		depth:          0,
		data:           make(map[string]any, 4),
	}
}

// Request 获取验证请求
func (c *ValidationContextImpl) Request() *core.ValidationRequest {
	return c.request
}

// ErrorCollector 获取错误收集器
func (c *ValidationContextImpl) ErrorCollector() core.ErrorCollector {
	return c.errorCollector
}

// Depth 当前嵌套深度
func (c *ValidationContextImpl) Depth() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.depth
}

// IncreaseDepth 增加深度
func (c *ValidationContextImpl) IncreaseDepth() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.depth++
	return c.depth
}

// DecreaseDepth 减少深度
func (c *ValidationContextImpl) DecreaseDepth() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.depth > 0 {
		c.depth--
	}
	return c.depth
}

// Set 设置上下文值
func (c *ValidationContextImpl) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = value
}

// Get 获取上下文值
func (c *ValidationContextImpl) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, ok := c.data[key]
	return val, ok
}

// Clone 克隆上下文（用于嵌套验证）
func (c *ValidationContextImpl) Clone() core.ValidationContext {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 创建新上下文，共享请求和错误收集器
	newCtx := &ValidationContextImpl{
		request:        c.request,
		errorCollector: c.errorCollector, // 共享错误收集器
		depth:          c.depth,
		data:           make(map[string]any, len(c.data)),
	}

	// 复制数据
	for k, v := range c.data {
		newCtx.data[k] = v
	}

	return newCtx
}

// Release 释放资源
func (c *ValidationContextImpl) Release() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 清空数据
	c.depth = 0
	clear(c.data)

	// 注意：不清空 request 和 errorCollector，它们可能还在使用
}
