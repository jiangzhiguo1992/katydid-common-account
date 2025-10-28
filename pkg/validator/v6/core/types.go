package core

import "context"

// ValidationRequest 验证请求
// 职责：封装验证所需的所有输入参数
// 设计原则：值对象，不可变
type ValidationRequest struct {
	// Target 待验证的目标对象
	Target any

	// Scene 验证场景
	Scene Scene

	// Fields 指定要验证的字段（空则验证所有）
	Fields []string

	// ExcludeFields 要排除的字段
	ExcludeFields []string

	// Context 上下文（用于超时、取消等）
	Context context.Context

	// Metadata 元数据（扩展用）
	Metadata map[string]any
}

// NewValidationRequest 创建验证请求
func NewValidationRequest(target any, scene Scene) *ValidationRequest {
	return &ValidationRequest{
		Target:   target,
		Scene:    scene,
		Context:  context.Background(),
		Metadata: make(map[string]any),
	}
}

// WithFields 设置要验证的字段
func (r *ValidationRequest) WithFields(fields ...string) *ValidationRequest {
	r.Fields = fields
	return r
}

// WithExcludeFields 设置要排除的字段
func (r *ValidationRequest) WithExcludeFields(fields ...string) *ValidationRequest {
	r.ExcludeFields = fields
	return r
}

// WithContext 设置上下文
func (r *ValidationRequest) WithContext(ctx context.Context) *ValidationRequest {
	r.Context = ctx
	return r
}

// WithMetadata 设置元数据
func (r *ValidationRequest) WithMetadata(key string, value any) *ValidationRequest {
	if r.Metadata == nil {
		r.Metadata = make(map[string]any)
	}
	r.Metadata[key] = value
	return r
}

// GetMetadata 获取元数据
func (r *ValidationRequest) GetMetadata(key string) (any, bool) {
	val, ok := r.Metadata[key]
	return val, ok
}

// ValidationResult 验证结果
// 职责：封装验证结果
type ValidationResult struct {
	// Success 是否验证成功
	Success bool

	// Errors 错误列表
	Errors []*FieldError

	// Metadata 元数据（用于返回额外信息）
	Metadata map[string]any
}

// NewValidationResult 创建验证结果
func NewValidationResult(success bool) *ValidationResult {
	return &ValidationResult{
		Success:  success,
		Errors:   make([]*FieldError, 0),
		Metadata: make(map[string]any),
	}
}

// WithErrors 设置错误
func (r *ValidationResult) WithErrors(errors []*FieldError) *ValidationResult {
	r.Errors = errors
	r.Success = len(errors) == 0
	return r
}

// AddError 添加错误
func (r *ValidationResult) AddError(err *FieldError) *ValidationResult {
	r.Errors = append(r.Errors, err)
	r.Success = false
	return r
}

// HasErrors 是否有错误
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// ToError 转换为 error
func (r *ValidationResult) ToError() error {
	if !r.HasErrors() {
		return nil
	}
	return NewValidationError(r.Errors)
}
