package context

import (
	"context"
	"katydid-common-account/pkg/validator/v6/core"
	"sync"
)

// validationContext 验证上下文实现
// 设计原则：单一职责 - 只管理上下文信息，不管理错误
type validationContext struct {
	goCtx    context.Context
	scene    core.Scene
	depth    int
	metadata core.IMetadata
}

// NewContext 创建新的验证上下文
func NewContext(scene core.Scene, opts ...ContextOption) core.IContext {
	ctx := acquireContext()
	ctx.scene = scene
	ctx.depth = 0
	ctx.goCtx = context.Background()
	ctx.metadata = NewMetadata()

	// 应用选项
	for _, opt := range opts {
		opt(ctx)
	}

	return ctx
}

// ContextOption 上下文选项
type ContextOption func(*validationContext)

// WithGoContext 设置 Go 标准上下文
func WithGoContext(goCtx context.Context) ContextOption {
	return func(c *validationContext) {
		c.goCtx = goCtx
	}
}

// WithDepth 设置深度
func WithDepth(depth int) ContextOption {
	return func(c *validationContext) {
		c.depth = depth
	}
}

// WithMetadata 设置元数据
func WithMetadata(key string, value any) ContextOption {
	return func(c *validationContext) {
		c.metadata.Set(key, value)
	}
}

// GoContext 实现 IContext 接口
func (c *validationContext) GoContext() context.Context {
	return c.goCtx
}

// Scene 实现 IContext 接口
func (c *validationContext) Scene() core.Scene {
	return c.scene
}

// Depth 实现 IContext 接口
func (c *validationContext) Depth() int {
	return c.depth
}

// Metadata 实现 IContext 接口
func (c *validationContext) Metadata() core.IMetadata {
	return c.metadata
}

// WithDepth 创建新的上下文，增加深度
func (c *validationContext) WithDepth(depth int) core.IContext {
	newCtx := acquireContext()
	newCtx.goCtx = c.goCtx
	newCtx.scene = c.scene
	newCtx.depth = depth
	newCtx.metadata = c.metadata // 共享元数据
	return newCtx
}

// Release 实现 IContext 接口
func (c *validationContext) Release() {
	releaseContext(c)
}

// ============================================================================
// 元数据实现
// ============================================================================

// metadata 元数据实现
type metadata struct {
	data map[string]any
	mu   sync.RWMutex
}

// NewMetadata 创建新的元数据
func NewMetadata() core.IMetadata {
	return &metadata{
		data: make(map[string]any),
	}
}

// Get 获取元数据
func (m *metadata) Get(key string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.data[key]
	return val, ok
}

// Set 设置元数据
func (m *metadata) Set(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

// Has 检查是否存在
func (m *metadata) Has(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.data[key]
	return ok
}

// Delete 删除元数据
func (m *metadata) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

// Clear 清空所有元数据
func (m *metadata) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]any)
}

// All 获取所有元数据
func (m *metadata) All() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// 返回副本
	result := make(map[string]any, len(m.data))
	for k, v := range m.data {
		result[k] = v
	}
	return result
}

// ============================================================================
// 上下文对象池
// ============================================================================

var contextPool = sync.Pool{
	New: func() any {
		return &validationContext{
			goCtx:    context.Background(),
			scene:    core.SceneNone,
			depth:    0,
			metadata: NewMetadata(),
		}
	},
}

// acquireContext 从对象池获取上下文
func acquireContext() *validationContext {
	return contextPool.Get().(*validationContext)
}

// releaseContext 释放上下文到对象池
func releaseContext(ctx *validationContext) {
	ctx.metadata.Clear()
	ctx.depth = 0
	ctx.scene = core.SceneNone
	ctx.goCtx = context.Background()
	contextPool.Put(ctx)
}

// ============================================================================
// 预定义元数据键
// ============================================================================

const (
	MetadataKeyTarget         = "target"          // 验证目标对象
	MetadataKeyValidateFields = "validate_fields" // 指定验证字段
	MetadataKeyExcludeFields  = "exclude_fields"  // 排除验证字段
	MetadataKeyMaxDepth       = "max_depth"       // 最大深度
)
