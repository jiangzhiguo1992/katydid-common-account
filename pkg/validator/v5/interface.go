package v5

// ============================================================================
// 外部实现
// ============================================================================

// RuleValidation 规则验证器接口
// 职责：提供字段级别的验证规则（required, min, max等）
// 设计原则：单一职责 - 只负责提供规则，不执行验证
type RuleValidation interface {
	// ValidateRules 获取指定场景的验证规则
	// 返回格式：map[场景]map[字段名]规则字符串
	ValidateRules() map[Scene]map[string]string
}

// BusinessValidation 自定义验证器接口
// 职责：执行复杂的业务逻辑验证（跨字段、数据库检查等）
// 设计原则：单一职责 - 只负责业务逻辑验证
type BusinessValidation interface {
	// ValidateBusiness 执行业务验证
	// 通过 ctx.AddError 添加错误
	ValidateBusiness(scene Scene, ctx *ValidationContext) error
}

// LifecycleHooks 生命周期钩子接口
// 职责：在验证前后执行自定义逻辑
// 设计原则：开放封闭 - 通过钩子扩展功能
type LifecycleHooks interface {
	// BeforeValidation 验证前执行
	BeforeValidation(ctx *ValidationContext) error
	// AfterValidation 验证后执行
	AfterValidation(ctx *ValidationContext) error
}

// ValidationListener 验证监听器接口
// 职责：监听验证过程中的事件（观察者模式）
type ValidationListener interface {
	// OnValidationStart 验证开始
	OnValidationStart(ctx *ValidationContext)
	// OnValidationEnd 验证结束
	OnValidationEnd(ctx *ValidationContext)
	// OnError 发生错误
	OnError(ctx *ValidationContext, err *FieldError)
}

// ============================================================================
// 内部定义
// ============================================================================

type StrategyType int8

const (
	StrategyTypeRule StrategyType = iota + 1
	StrategyTypeNested
	StrategyTypeBusiness
)

// ValidationStrategy 验证策略接口
// 职责：定义具体的验证策略
type ValidationStrategy interface {
	// Type 策略类型
	Type() StrategyType
	// Priority 优先级（数字越小优先级越高）
	Priority() int8
	// Validate 执行验证
	Validate(target any, ctx *ValidationContext) error
}
