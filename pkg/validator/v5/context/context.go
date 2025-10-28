package context

import (
	"context"
	"katydid-common-account/pkg/validator/v5/core"
)

const (
	// MetadataKeyValidateFields 指定字段验证的元数据键
	MetadataKeyValidateFields = "validate_fields"
	// MetadataKeyExcludeFields 排除字段验证的元数据键
	MetadataKeyExcludeFields = "exclude_fields"
)

// ValidationContext 验证上下文
// 职责：携带验证过程中的上下文信息
type ValidationContext struct {
	// context Go 标准上下文
	context context.Context
	// scene 当前验证场景
	scene core.Scene
	// depth 嵌套深度
	depth int8
	// errors 错误收集
	errors []core.IFieldError
	// metadata 元数据（用于扩展）
	metadata map[string]any
	// 最大错误数
	maxErrors int
}

// NewValidationContext 创建验证上下文
func NewValidationContext(scene core.Scene, maxErrors int, opts ...ValidationContextOption) core.IValidationContext {
	// 使用对象池优化内存分配
	ctx := acquireValidationContext(scene, maxErrors)

	// 应用选项
	for _, opt := range opts {
		opt(ctx)
	}

	return ctx
}

// ValidationContextOption 验证上下文选项函数
type ValidationContextOption func(*ValidationContext)

// WithContext 设置 Go 标准上下文
func WithContext(ctx context.Context) ValidationContextOption {
	return func(c *ValidationContext) {
		c.context = ctx
	}
}

// WithDepth 设置嵌套深度
func WithDepth(depth int8) ValidationContextOption {
	return func(c *ValidationContext) {
		c.depth = depth
	}
}

// WithErrors 设置错误列表
func WithErrors(errors []core.IFieldError) ValidationContextOption {
	return func(c *ValidationContext) {
		c.errors = errors
	}
}

// WithMetadata 设置元数据
func WithMetadata(metadata map[string]any) ValidationContextOption {
	return func(c *ValidationContext) {
		c.metadata = metadata
	}
}

// WithAddMetadata 添加元数据
func WithAddMetadata(key string, value any) ValidationContextOption {
	return func(c *ValidationContext) {
		if c.metadata == nil {
			c.metadata = make(map[string]any)
		}
		c.metadata[key] = value
	}
}

// Release 释放验证上下文到对象池
// 使用完毕后应该调用此方法
func (vc *ValidationContext) Release() {
	releaseValidationContext(vc)
}

// Context 获取 Go 标准上下文
func (vc *ValidationContext) Context() context.Context {
	return vc.context
}

// Scene 获取当前验证场景
func (vc *ValidationContext) Scene() core.Scene {
	return vc.scene
}

// Depth 获取当前嵌套深度
func (vc *ValidationContext) Depth() int8 {
	return vc.depth
}

// Errors 获取所有错误
func (vc *ValidationContext) Errors() []core.IFieldError {
	return vc.errors
}

// Metadata 获取所有元数据
func (vc *ValidationContext) Metadata() map[string]any {
	return vc.metadata
}

// MaxErrors 获取最大错误数
func (vc *ValidationContext) MaxErrors() int {
	return vc.maxErrors
}

// AddError 添加错误
func (vc *ValidationContext) AddError(err core.IFieldError) bool {
	// 检查是否超过最大错误数
	if vc.ErrorCount() >= vc.maxErrors {
		return false
	}
	vc.errors = append(vc.errors, err)
	return true
}

// AddErrors 批量添加错误
func (vc *ValidationContext) AddErrors(errs []core.IFieldError) bool {
	// 检查是否超过最大错误数
	if vc.ErrorCount() >= (vc.maxErrors - len(errs)) {
		return false
	}
	vc.errors = append(vc.errors, errs...)
	return true
}

// HasErrors 是否有错误
func (vc *ValidationContext) HasErrors() bool {
	return len(vc.errors) > 0
}

// ErrorCount 错误数量
func (vc *ValidationContext) ErrorCount() int {
	return len(vc.errors)
}

// GetMetadata 获取元数据
func (vc *ValidationContext) GetMetadata(key string) (any, bool) {
	if vc.metadata == nil {
		return nil, false
	}
	val, ok := vc.metadata[key]
	return val, ok
}
