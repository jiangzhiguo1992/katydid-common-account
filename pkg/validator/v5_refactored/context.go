package v5_refactored

import (
	"context"
	"sync"
)

// ============================================================================
// 验证上下文
// ============================================================================

// ValidationContext 验证上下文
// 职责：携带验证过程中的上下文信息
// 设计原则：单一职责 - 只负责上下文数据的存储和传递
type ValidationContext struct {
	// Context Go 标准上下文
	Context context.Context

	// Scene 当前验证场景
	Scene Scene

	// Target 验证目标对象
	Target any

	// Depth 嵌套深度（防止循环引用）
	Depth int

	// MaxDepth 最大嵌套深度
	MaxDepth int

	// Metadata 元数据（用于扩展）
	Metadata map[string]any

	// mu 保护元数据的并发访问
	mu sync.RWMutex
}

// NewValidationContext 创建验证上下文
func NewValidationContext(scene Scene, target any) *ValidationContext {
	return &ValidationContext{
		Context:  context.Background(),
		Scene:    scene,
		Target:   target,
		Depth:    0,
		MaxDepth: 100,
		Metadata: make(map[string]any),
	}
}

// WithContext 设置 Go 标准上下文
func (vc *ValidationContext) WithContext(ctx context.Context) *ValidationContext {
	vc.Context = ctx
	return vc
}

// WithMaxDepth 设置最大嵌套深度
func (vc *ValidationContext) WithMaxDepth(depth int) *ValidationContext {
	vc.MaxDepth = depth
	return vc
}

// WithMetadata 设置元数据
func (vc *ValidationContext) WithMetadata(key string, value any) *ValidationContext {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	if vc.Metadata == nil {
		vc.Metadata = make(map[string]any)
	}
	vc.Metadata[key] = value
	return vc
}

// GetMetadata 获取元数据
func (vc *ValidationContext) GetMetadata(key string) (any, bool) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	val, ok := vc.Metadata[key]
	return val, ok
}

// IncrementDepth 增加深度
func (vc *ValidationContext) IncrementDepth() *ValidationContext {
	vc.Depth++
	return vc
}

// DecrementDepth 减少深度
func (vc *ValidationContext) DecrementDepth() *ValidationContext {
	vc.Depth--
	return vc
}

// IsMaxDepthReached 是否达到最大深度
func (vc *ValidationContext) IsMaxDepthReached() bool {
	return vc.Depth >= vc.MaxDepth
}

// Clone 克隆上下文（用于嵌套验证）
func (vc *ValidationContext) Clone() *ValidationContext {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	metadata := make(map[string]any, len(vc.Metadata))
	for k, v := range vc.Metadata {
		metadata[k] = v
	}

	return &ValidationContext{
		Context:  vc.Context,
		Scene:    vc.Scene,
		Target:   vc.Target,
		Depth:    vc.Depth,
		MaxDepth: vc.MaxDepth,
		Metadata: metadata,
	}
}

// Reset 重置上下文（对象池复用）
func (vc *ValidationContext) Reset() {
	vc.Context = nil
	vc.Scene = SceneNone
	vc.Target = nil
	vc.Depth = 0
	vc.MaxDepth = 100

	vc.mu.Lock()
	clear(vc.Metadata)
	vc.mu.Unlock()
}

// ============================================================================
// 上下文池
// ============================================================================

var (
	// contextPool 验证上下文对象池
	contextPool = sync.Pool{
		New: func() interface{} {
			return &ValidationContext{
				Metadata: make(map[string]any, 4),
			}
		},
	}
)

// AcquireContext 从池中获取上下文
func AcquireContext(scene Scene, target any) *ValidationContext {
	ctx := contextPool.Get().(*ValidationContext)
	ctx.Scene = scene
	ctx.Target = target
	ctx.Context = context.Background()
	ctx.Depth = 0
	ctx.MaxDepth = 100
	return ctx
}

// ReleaseContext 释放上下文到池
func ReleaseContext(ctx *ValidationContext) {
	ctx.Reset()
	contextPool.Put(ctx)
}

// ============================================================================
// 元数据键定义
// ============================================================================

const (
	// MetadataKeyValidateFields 指定字段验证的元数据键
	MetadataKeyValidateFields = "validate_fields"

	// MetadataKeyExcludeFields 排除字段验证的元数据键
	MetadataKeyExcludeFields = "exclude_fields"

	// MetadataKeyCurrentField 当前正在验证的字段
	MetadataKeyCurrentField = "current_field"

	// MetadataKeyParentNamespace 父级命名空间
	MetadataKeyParentNamespace = "parent_namespace"
)
