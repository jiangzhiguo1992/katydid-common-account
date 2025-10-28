package core

// ============================================================================
// 外部需要实现的接口
// ============================================================================

// IRuleValidation 规则验证器接口
// 职责：提供字段级别的验证规则（required, min, max等）
// 设计原则：单一职责 - 只负责提供规则，不执行验证
type IRuleValidation interface {
	// ValidateRules 获取指定场景的验证规则
	// 返回格式：map[场景]map[字段名]规则字符串
	ValidateRules() map[Scene]map[string]string
}

// IBusinessValidation 自定义验证器接口
// 职责：执行复杂的业务逻辑验证（跨字段、数据库检查等）
// 设计原则：单一职责 - 只负责业务逻辑验证
type IBusinessValidation interface {
	// ValidateBusiness 执行业务验证
	// 通过 ctx.AddError 添加错误
	ValidateBusiness(scene Scene, ctx IValidationContext) error
}

// ILifecycleHooks 生命周期钩子接口
// 职责：在验证前后执行自定义逻辑
// 设计原则：开放封闭 - 通过钩子扩展功能
type ILifecycleHooks interface {
	// BeforeValidation 验证前执行
	BeforeValidation(ctx IValidationContext) error
	// AfterValidation 验证后执行
	AfterValidation(ctx IValidationContext) error
}

// IValidationListener 验证监听器接口
// 职责：监听验证过程中的事件（观察者模式）
type IValidationListener interface {
	// OnValidationStart 验证开始
	OnValidationStart(ctx IValidationContext)
	// OnValidationEnd 验证结束
	OnValidationEnd(ctx IValidationContext)
	// OnError 发生错误
	OnError(ctx IValidationContext, err IFieldError)
}

// ============================================================================
// 内部定义的接口
// ============================================================================

const (
	// ErrorMessageEstimatedLength 预估的错误消息平均长度，用于优化字符串构建时的内存分配
	ErrorMessageEstimatedLength = 80
)

// IValidationContext 验证上下文接口
// 职责：管理验证过程中的状态和错误信息
type IValidationContext interface {
	// AddError 添加单个字段错误
	AddError(IFieldError) bool
	// AddErrors 添加多个字段错误
	AddErrors([]IFieldError) bool
	// GetErrors 获取所有字段错误
	GetErrors() []IFieldError
	// HasErrors 检查是否存在错误
	HasErrors() bool
	// ErrorCount 获取错误数量
	ErrorCount() int
	// GetMetadata 获取上下文元数据
	GetMetadata(key string) (any, bool)
}

// IFieldError 字段错误接口
// 职责：封装字段级别的验证错误信息
type IFieldError interface {
	// Namespace 字段的完整命名空间路径（如 User.Profile.Email）
	// 用于嵌套结构体的错误定位，支持复杂对象的精确错误追踪
	Namespace() string
	// Tag 验证标签，描述验证规则类型（如 required, email, min, max 等）
	Tag() string
	// Param 验证参数，提供验证规则的具体配置值
	// 例如：min=3 中的 "3"，len=11 中的 "11"
	Param() string
	// Value 字段的实际值（用于 sl.ReportError 的 value 参数）
	// 用于调试和详细错误信息，可能包含敏感信息，谨慎使用
	Value() any
	// Message 用户友好的错误消息（可选，用于直接显示给终端用户）
	// 支持国际化，建议使用本地化后的错误消息
	Message() string
}

// IErrorFormatter 错误格式化器接口
// 职责：格式化错误信息
type IErrorFormatter interface {
	// Format 格式化单个错误
	Format(err IFieldError) string
	// FormatAll 格式化所有错误
	FormatAll(errs []IFieldError) string
}

// IValidationError 验证错误接口
// 职责：封装多个字段错误
type IValidationError interface {
	// HasErrors 是否有错误
	HasErrors() bool
	// Formatter 格式化所有错误
	Formatter() []string
}

// ISceneMatcher 场景匹配器接口
type ISceneMatcher interface {
	// Match 判断场景是否匹配
	Match(current, target Scene) bool
	// MatchRules 匹配并合并规则
	MatchRules(current Scene, rules map[Scene]map[string]string) map[string]string
}

// StrategyType 验证策略类型枚举
type StrategyType int8

// 验证策略类型枚举值
const (
	StrategyTypeRule StrategyType = iota + 1
	StrategyTypeNested
	StrategyTypeBusiness
)

// IValidationStrategy 验证策略接口
// 职责：定义具体的验证策略
type IValidationStrategy interface {
	// Type 策略类型
	Type() StrategyType
	// Priority 优先级（数字越小优先级越高）
	Priority() int8
	// Validate 执行验证
	Validate(target any, ctx IValidationContext) error
}
