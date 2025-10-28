package core

// ================================
// 核心接口定义
// 设计原则：接口隔离、依赖倒置
// ================================

// Validator 验证器接口（门面接口）
// 职责：提供统一的验证入口
// 设计模式：门面模式
type Validator interface {
	// Validate 验证对象
	Validate(target any, scene Scene) error

	// ValidateWithRequest 使用请求对象验证
	ValidateWithRequest(req *ValidationRequest) (*ValidationResult, error)
}

// ValidationOrchestrator 验证编排器接口
// 职责：编排验证流程，协调各组件
// 设计原则：单一职责 - 只负责编排，不执行具体验证
type ValidationOrchestrator interface {
	// Orchestrate 编排验证流程
	Orchestrate(req *ValidationRequest) (*ValidationResult, error)
}

// StrategyExecutor 策略执行器接口
// 职责：执行验证策略，处理异常恢复
// 设计原则：单一职责 - 只负责执行策略
type StrategyExecutor interface {
	// Execute 执行单个策略
	Execute(strategy ValidationStrategy, req *ValidationRequest, ctx ValidationContext) error

	// ExecuteAll 执行所有策略
	ExecuteAll(strategies []ValidationStrategy, req *ValidationRequest, ctx ValidationContext) error
}

// ValidationStrategy 验证策略接口
// 职责：定义具体的验证策略
// 设计模式：策略模式
type ValidationStrategy interface {
	// Name 策略名称
	Name() string

	// Type 策略类型
	Type() StrategyType

	// Priority 优先级（数字越小优先级越高）
	Priority() int

	// Validate 执行验证
	Validate(req *ValidationRequest, ctx ValidationContext) error
}

// StrategyType 策略类型
type StrategyType int

const (
	StrategyTypeRule StrategyType = iota + 1
	StrategyTypeBusiness
	StrategyTypeNested
	StrategyTypeCustom
)

// ErrorCollector 错误收集器接口
// 职责：收集和管理验证错误
// 设计原则：单一职责 - 只负责错误收集
type ErrorCollector interface {
	// Add 添加错误
	Add(err *FieldError) bool

	// AddAll 批量添加错误
	AddAll(errs []*FieldError)

	// GetAll 获取所有错误
	GetAll() []*FieldError

	// HasErrors 是否有错误
	HasErrors() bool

	// Count 错误数量
	Count() int

	// Clear 清空错误
	Clear()

	// SetMaxErrors 设置最大错误数
	SetMaxErrors(max int)
}

// ValidationContext 验证上下文接口
// 职责：携带验证过程中的上下文信息
// 设计原则：上下文对象模式
type ValidationContext interface {
	// Request 获取验证请求
	Request() *ValidationRequest

	// ErrorCollector 获取错误收集器
	ErrorCollector() ErrorCollector

	// Depth 当前嵌套深度
	Depth() int

	// IncreaseDepth 增加深度
	IncreaseDepth() int

	// DecreaseDepth 减少深度
	DecreaseDepth() int

	// Set 设置上下文值
	Set(key string, value any)

	// Get 获取上下文值
	Get(key string) (any, bool)

	// Clone 克隆上下文（用于嵌套验证）
	Clone() ValidationContext

	// Release 释放资源（对象池）
	Release()
}

// EventDispatcher 事件分发器接口
// 职责：分发验证事件给监听器
// 设计模式：观察者模式
type EventDispatcher interface {
	// Dispatch 分发事件
	Dispatch(event ValidationEvent)

	// Subscribe 订阅事件
	Subscribe(listener ValidationListener)

	// Unsubscribe 取消订阅
	Unsubscribe(listener ValidationListener)
}

// ValidationEvent 验证事件接口
type ValidationEvent interface {
	// Type 事件类型
	Type() EventType

	// Context 获取上下文
	Context() ValidationContext

	// Timestamp 时间戳
	Timestamp() int64
}

// EventType 事件类型
type EventType int

const (
	EventTypeValidationStart EventType = iota + 1
	EventTypeValidationEnd
	EventTypeStrategyStart
	EventTypeStrategyEnd
	EventTypeError
)

// ValidationListener 验证监听器接口
// 职责：监听验证过程中的事件
// 设计模式：观察者模式
type ValidationListener interface {
	// OnEvent 事件处理
	OnEvent(event ValidationEvent)
}

// ErrorFormatter 错误格式化器接口
// 职责：格式化错误信息
// 设计原则：单一职责 - 只负责格式化
type ErrorFormatter interface {
	// Format 格式化单个错误
	Format(err *FieldError) string

	// FormatAll 格式化所有错误
	FormatAll(errs []*FieldError) string
}

// SceneMatcher 场景匹配器接口
// 职责：匹配验证场景
// 设计原则：单一职责 - 只负责场景匹配
type SceneMatcher interface {
	// Match 判断场景是否匹配
	Match(target, current Scene) bool

	// MatchRules 匹配并合并规则
	MatchRules(scene Scene, rules map[Scene]map[string]string) map[string]string
}

// TypeRegistry 类型注册表接口
// 职责：注册和缓存类型信息
// 设计原则：单一职责 - 只负责类型管理
type TypeRegistry interface {
	// Register 注册类型
	Register(target any) TypeInfo

	// Get 获取类型信息
	Get(target any) (TypeInfo, bool)

	// Clear 清除缓存
	Clear()
}

// TypeInfo 类型信息接口
type TypeInfo interface {
	// HasRuleValidation 是否实现了规则验证
	HasRuleValidation() bool

	// HasBusinessValidation 是否实现了业务验证
	HasBusinessValidation() bool

	// HasLifecycleHooks 是否实现了生命周期钩子
	HasLifecycleHooks() bool

	// GetRules 获取验证规则
	GetRules() map[Scene]map[string]string

	// GetFieldAccessor 获取字段访问器
	GetFieldAccessor(fieldName string) FieldAccessor
}

// FieldAccessor 字段访问器函数类型
// 设计思想：使用索引访问字段，优化性能
type FieldAccessor func(target any) (value any, exists bool)

// ================================
// 用户实现的接口
// ================================

// RuleProvider 规则提供者接口
// 职责：提供验证规则
// 设计原则：单一职责 - 只提供规则，不执行验证
type RuleProvider interface {
	// GetRules 获取验证规则
	// 返回格式：map[场景]map[字段名]规则字符串
	GetRules() map[Scene]map[string]string
}

// BusinessValidator 业务验证器接口
// 职责：执行业务逻辑验证
// 设计原则：单一职责 - 只负责业务验证
type BusinessValidator interface {
	// ValidateBusiness 执行业务验证
	ValidateBusiness(scene Scene, ctx ValidationContext) error
}

// LifecycleHook 生命周期钩子接口
// 职责：在验证前后执行自定义逻辑
// 设计原则：开放封闭 - 通过钩子扩展功能
type LifecycleHook interface {
	// BeforeValidation 验证前
	BeforeValidation(ctx ValidationContext) error

	// AfterValidation 验证后
	AfterValidation(ctx ValidationContext) error
}

// Plugin 插件接口
// 职责：扩展验证器功能
// 设计模式：插件模式
type Plugin interface {
	// Name 插件名称
	Name() string

	// Init 初始化插件
	Init(config map[string]any) error

	// BeforeValidate 验证前钩子
	BeforeValidate(ctx ValidationContext) error

	// AfterValidate 验证后钩子
	AfterValidate(ctx ValidationContext) error

	// Enabled 是否启用
	Enabled() bool
}
