package v2

// ============================================================================
// 验证上下文 - 单一职责：管理验证过程中的状态和错误收集
// 设计原则：高内聚低耦合，集中管理验证环境信息
// ============================================================================

// ValidationContext 验证上下文
// 用于在验证过程中传递状态信息和收集错误
type ValidationContext struct {
	// Scene 当前验证场景
	Scene Scene

	// Depth 当前递归深度（用于嵌套验证）
	Depth int

	// MaxDepth 最大递归深度限制
	MaxDepth int

	// Errors 错误收集器
	Errors ErrorCollector

	// Visited 已访问的对象（防止循环引用）
	Visited map[uintptr]bool

	// Options 验证选项
	Options *ValidateOptions

	// CustomData 自定义数据（扩展用）
	CustomData map[string]interface{}
}

// NewValidationContext 创建验证上下文
func NewValidationContext(scene Scene, opts *ValidateOptions) *ValidationContext {
	if opts == nil {
		opts = DefaultValidateOptions()
		opts.Scene = scene
	}

	ctx := &ValidationContext{
		Scene:      scene,
		Depth:      0,
		MaxDepth:   100, // 默认最大深度
		Errors:     GetPooledErrorCollector(),
		Visited:    make(map[uintptr]bool),
		Options:    opts,
		CustomData: make(map[string]interface{}),
	}

	return ctx
}

// Release 释放上下文资源
func (ctx *ValidationContext) Release() {
	if ctx.Errors != nil {
		PutPooledErrorCollector(ctx.Errors)
		ctx.Errors = nil
	}
	ctx.Visited = nil
	ctx.CustomData = nil
}

// IncrementDepth 增加递归深度
func (ctx *ValidationContext) IncrementDepth() bool {
	ctx.Depth++
	return ctx.Depth <= ctx.MaxDepth
}

// DecrementDepth 减少递归深度
func (ctx *ValidationContext) DecrementDepth() {
	if ctx.Depth > 0 {
		ctx.Depth--
	}
}

// IsVisited 检查对象是否已访问（防止循环引用）
func (ctx *ValidationContext) IsVisited(ptr uintptr) bool {
	return ctx.Visited[ptr]
}

// MarkVisited 标记对象为已访问
func (ctx *ValidationContext) MarkVisited(ptr uintptr) {
	ctx.Visited[ptr] = true
}

// HasErrors 是否有错误
func (ctx *ValidationContext) HasErrors() bool {
	return ctx.Errors != nil && ctx.Errors.HasErrors()
}

// GetErrors 获取所有错误
func (ctx *ValidationContext) GetErrors() error {
	if ctx.Errors == nil || !ctx.Errors.HasErrors() {
		return nil
	}
	return ctx.Errors.GetErrors()
}

// ShouldStop 是否应该停止验证（快速失败模式）
func (ctx *ValidationContext) ShouldStop() bool {
	return ctx.Options != nil && ctx.Options.FailFast && ctx.HasErrors()
}

// SetCustomData 设置自定义数据
func (ctx *ValidationContext) SetCustomData(key string, value interface{}) {
	if ctx.CustomData == nil {
		ctx.CustomData = make(map[string]interface{})
	}
	ctx.CustomData[key] = value
}

// GetCustomData 获取自定义数据
func (ctx *ValidationContext) GetCustomData(key string) (interface{}, bool) {
	if ctx.CustomData == nil {
		return nil, false
	}
	val, ok := ctx.CustomData[key]
	return val, ok
}
