package v5

import (
	"context"
	"katydid-common-account/pkg/validator/v5/core"
	error2 "katydid-common-account/pkg/validator/v5/error"
)

const (
	metadataKeyValidateFields = "validate_fields" // 指定字段验证的元数据键
	metadataKeyExcludeFields  = "exclude_fields"  // 排除字段验证的元数据键
)

// ValidationContext 验证上下文
// 职责：携带验证过程中的上下文信息
type ValidationContext struct {
	// Context Go 标准上下文
	Context context.Context
	// Scene 当前验证场景
	Scene core.Scene
	// 最大错误数
	MaxErrors int
	// Depth 嵌套深度
	Depth int
	// errors 错误收集
	errors []*error2.FieldError
	// Metadata 元数据（用于扩展）
	Metadata map[string]any
}

// NewValidationContext 创建验证上下文
func NewValidationContext(scene core.Scene, maxErrors int) *ValidationContext {
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
func (vc *ValidationContext) WithErrors(errors []*error2.FieldError) *ValidationContext {
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
	vc.Scene = core.SceneNone
	vc.Depth = 0
	clear(vc.errors)
	clear(vc.Metadata)

	validationContextPool.Put(vc)
}

// AddError 添加错误
func (vc *ValidationContext) AddError(err *error2.FieldError) bool {
	// 检查是否超过最大错误数
	if vc.ErrorCount() >= vc.MaxErrors {
		return false
	}
	vc.errors = append(vc.errors, err)
	return true
}

// AddErrors 批量添加错误
func (vc *ValidationContext) AddErrors(errs []*error2.FieldError) bool {
	// 检查是否超过最大错误数
	if vc.ErrorCount() >= (vc.MaxErrors - len(errs)) {
		return false
	}
	vc.errors = append(vc.errors, errs...)
	return true
}

// GetErrors 获取所有错误
func (vc *ValidationContext) GetErrors() []*error2.FieldError {
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
