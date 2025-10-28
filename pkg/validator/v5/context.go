package v5

import (
	"context"
	"strings"
	"sync"
)

const (
	metadataKeyValidateFields = "validate_fields" // 指定字段验证的元数据键
	metadataKeyExcludeFields  = "exclude_fields"  // 排除字段验证的元数据键
)

var (
	// validationContextPool 验证上下文对象池
	validationContextPool = sync.Pool{
		New: func() interface{} {
			return &ValidationContext{
				errors:   make([]*FieldError, 0),
				Metadata: make(map[string]any),
			}
		},
	}

	// stringBuilderPool 字符串构建器对象池
	stringBuilderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}
)

// ValidationContext 验证上下文
// 职责：携带验证过程中的上下文信息
type ValidationContext struct {
	// Context Go 标准上下文
	Context context.Context
	// Scene 当前验证场景
	Scene Scene
	// 最大错误数
	MaxErrors int
	// Depth 嵌套深度
	Depth int
	// errors 错误收集
	errors []*FieldError
	// Metadata 元数据（用于扩展）
	Metadata map[string]any
}

// NewValidationContext 创建验证上下文
func NewValidationContext(scene Scene, maxErrors int) *ValidationContext {
	// 使用对象池优化内存分配
	ctx := validationContextPool.Get().(*ValidationContext)
	ctx.Scene = scene
	ctx.MaxErrors = maxErrors
	ctx.Depth = 0

	clear(ctx.errors)
	clear(ctx.Metadata)

	return ctx
}

// WithContext 设置 Go 标准上下文
func (vc *ValidationContext) WithContext(ctx context.Context) *ValidationContext {
	vc.Context = ctx
	return vc
}

// WithErrors 设置错误列表
func (vc *ValidationContext) WithErrors(errors []*FieldError) *ValidationContext {
	vc.errors = errors
	return vc
}

// WithMetadata 设置元数据
func (vc *ValidationContext) WithMetadata(key string, value any) *ValidationContext {
	if vc.Metadata == nil {
		vc.Metadata = make(map[string]any)
	}
	vc.Metadata[key] = value
	return vc
}

// Release 释放验证上下文到对象池
// 使用完毕后应该调用此方法
func (vc *ValidationContext) Release() {
	// 清空字段
	vc.Context = nil
	vc.Scene = SceneNone
	vc.Depth = 0
	clear(vc.errors)
	clear(vc.Metadata)

	validationContextPool.Put(vc)
}

// AddError 添加错误
func (vc *ValidationContext) AddError(err *FieldError) bool {
	// 检查是否超过最大错误数
	if vc.ErrorCount() >= vc.MaxErrors {
		return false
	}
	vc.errors = append(vc.errors, err)
	return true
}

// AddErrors 批量添加错误
func (vc *ValidationContext) AddErrors(errs []*FieldError) bool {
	// 检查是否超过最大错误数
	if vc.ErrorCount() >= (vc.MaxErrors - len(errs)) {
		return false
	}
	vc.errors = append(vc.errors, errs...)
	return true
}

// GetErrors 获取所有错误
func (vc *ValidationContext) GetErrors() []*FieldError {
	return vc.errors
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
	if vc.Metadata == nil {
		return nil, false
	}
	val, ok := vc.Metadata[key]
	return val, ok
}

// acquireStringBuilder 从对象池获取字符串构建器
func acquireStringBuilder() *strings.Builder {
	sb := stringBuilderPool.Get().(*strings.Builder)
	sb.Reset()
	return sb
}

// releaseStringBuilder 归还字符串构建器到对象池
func releaseStringBuilder(sb *strings.Builder) {
	if sb == nil {
		return
	}

	// 防止内存泄漏：不归还过大的Builder
	if sb.Cap() > 10*1024 { // 超过10KB
		return
	}

	sb.Reset()
	stringBuilderPool.Put(sb)
}
