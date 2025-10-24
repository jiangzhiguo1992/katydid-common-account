package v5

import "context"

// ============================================================================
// 数据结构
// ============================================================================

// ValidationContext 验证上下文
// 职责：携带验证过程中的上下文信息
// 设计原则：单一职责 - 只负责数据传递
type ValidationContext struct {
	// Context Go 标准上下文
	Context context.Context
	// Scene 当前验证场景
	Scene Scene
	// Target 验证目标对象
	Target any
	// Depth 嵌套深度
	Depth int
	// Metadata 元数据（用于扩展）
	Metadata map[string]any
	// errorCollector 错误收集器（私有，通过方法访问）
	errorCollector ErrorCollector
}

// NewValidationContext 创建验证上下文
func NewValidationContext(scene Scene, target any) *ValidationContext {
	return &ValidationContext{
		Context:        context.Background(),
		Scene:          scene,
		Target:         target,
		Depth:          0,
		Metadata:       make(map[string]any),
		errorCollector: NewDefaultErrorCollector(),
	}
}

// AddError 添加错误
func (vc *ValidationContext) AddError(err *FieldError) {
	if vc.errorCollector != nil {
		vc.errorCollector.AddError(err)
	}
}

// AddErrors 批量添加错误
func (vc *ValidationContext) AddErrors(errs []*FieldError) {
	if vc.errorCollector != nil {
		vc.errorCollector.AddErrors(errs)
	}
}

// GetErrors 获取所有错误
func (vc *ValidationContext) GetErrors() []*FieldError {
	if vc.errorCollector != nil {
		return vc.errorCollector.GetErrors()
	}
	return nil
}

// HasErrors 是否有错误
func (vc *ValidationContext) HasErrors() bool {
	if vc.errorCollector != nil {
		return vc.errorCollector.HasErrors()
	}
	return false
}

// ErrorCount 错误数量
func (vc *ValidationContext) ErrorCount() int {
	if vc.errorCollector != nil {
		return vc.errorCollector.ErrorCount()
	}
	return 0
}

// WithContext 设置 Go 标准上下文
func (vc *ValidationContext) WithContext(ctx context.Context) *ValidationContext {
	vc.Context = ctx
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

// GetMetadata 获取元数据
func (vc *ValidationContext) GetMetadata(key string) (any, bool) {
	if vc.Metadata == nil {
		return nil, false
	}
	val, ok := vc.Metadata[key]
	return val, ok
}
