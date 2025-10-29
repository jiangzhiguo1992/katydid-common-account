package core

import "context"

// ============================================================================
// 业务层接口 - 由业务模型实现
// ============================================================================

// IRuleValidator 规则提供者接口
// 职责：提供字段级别的验证规则
// 设计原则：单一职责 - 只提供规则，不执行验证
type IRuleValidator interface {
	// ValidateRules 获取指定场景的验证规则
	// 返回格式：map[字段名]规则字符串
	// 如果场景不匹配，返回 nil
	ValidateRules(scene Scene) map[string]string
}

// IBusinessValidator 业务验证器接口
// 职责：执行复杂的业务逻辑验证（跨字段、数据库检查等）
// 设计原则：单一职责 - 只负责业务逻辑验证
type IBusinessValidator interface {
	// ValidateBusiness 执行业务验证
	// 通过 collector.Collect() 添加错误
	ValidateBusiness(scene Scene, collector IErrorCollector)
}

// ILifecycleHooks 生命周期钩子接口
// 职责：在验证前后执行自定义逻辑
// 设计原则：开放封闭 - 通过钩子扩展功能
type ILifecycleHooks interface {
	// BeforeValidation 验证前执行
	BeforeValidation(ctx IContext) error
	// AfterValidation 验证后执行
	AfterValidation(ctx IContext) error
}

// ============================================================================
// 框架层接口 - 验证器核心接口
// ============================================================================

// IValidator 验证器核心接口
// 职责：提供验证功能
// 设计原则：接口隔离 - 只包含验证相关方法
type IValidator interface {
	// Validate 执行完整验证
	Validate(target any, scene Scene) IValidationError

	// ValidateWithContext 使用自定义上下文执行验证
	ValidateWithContext(target any, ctx IContext) error
}

// IFieldValidator 字段级验证器接口
// 职责：提供字段级别的验证功能
// 设计原则：接口隔离 - 分离字段验证职责
// TODO:GG 没人实现？
type IFieldValidator interface {
	// ValidateFields 验证指定字段
	ValidateFields(target any, scene Scene, fields ...string) IValidationError

	// ValidateFieldsExcept 验证除指定字段外的所有字段
	ValidateFieldsExcept(target any, scene Scene, excludeFields ...string) IValidationError
}

// IStrategyManager 策略管理器接口
// 职责：管理验证策略
// 设计原则：接口隔离 - 分离策略管理职责
// TODO:GG 没人实现？
type IStrategyManager interface {
	// RegisterStrategy 注册验证策略
	RegisterStrategy(strategy IValidationStrategy)

	// UnregisterStrategy 注销验证策略
	UnregisterStrategy(strategyType StrategyType)

	// GetStrategies 获取所有策略
	GetStrategies() []IValidationStrategy
}

// IConfigurableValidator 可配置验证器接口
// 职责：提供配置管理功能
// 设计原则：接口隔离 - 分离配置职责
type IConfigurableValidator interface {
	// RegisterAlias 注册规则别名
	RegisterAlias(alias, tags string)

	// RegisterValidation 注册自定义验证函数
	RegisterValidation(tag string, fn ValidationFunc) error
}

// ============================================================================
// 上下文接口
// ============================================================================

// IContext 验证上下文接口
// 职责：携带验证过程中的上下文信息（不包含错误）
// 设计原则：单一职责 - 只管理上下文，不管理错误
type IContext interface {
	// GoContext 获取 Go 标准上下文
	GoContext() context.Context

	// Scene 获取当前验证场景
	Scene() Scene

	// Depth 获取当前验证深度（用于嵌套结构体）
	Depth() int

	// Metadata 获取元数据
	Metadata() IMetadata

	// WithDepth 创建新的上下文，增加深度
	WithDepth(depth int) IContext

	// Release 释放上下文资源
	Release()
}

// IMetadata 元数据接口
// 职责：管理键值对元数据
// 设计原则：单一职责
type IMetadata interface {
	// Get 获取元数据
	Get(key string) (any, bool)

	// Set 设置元数据
	Set(key string, value any)

	// Has 检查是否存在
	Has(key string) bool

	// Delete 删除元数据
	Delete(key string)

	// Clear 清空所有元数据
	Clear()

	// All 获取所有元数据
	All() map[string]any
}

// ============================================================================
// 错误相关接口
// ============================================================================

// IFieldError 字段错误接口
// 职责：封装单个字段的验证错误信息
// 设计原则：值对象模式，不可变
type IFieldError interface {
	// Namespace 字段的完整命名空间路径（如 User.Profile.Email）
	Namespace() string

	// Field 字段名（如 Email）
	Field() string

	// Tag 验证标签（如 required, email, min）
	Tag() string

	// Param 验证参数（如 min=3 中的 "3"）
	Param() string

	// Value 字段的实际值
	Value() any

	// Message 用户友好的错误消息
	Message() string

	// Error 实现 error 接口
	Error() string
}

// IErrorCollector 错误收集器接口
// 职责：收集和管理验证错误
// 设计原则：单一职责 - 只负责错误收集
type IErrorCollector interface {
	// Collect 收集单个错误
	// 返回 false 表示已达到最大错误数，停止收集
	Collect(err IFieldError) bool

	// CollectAll 批量收集错误
	CollectAll(errs []IFieldError) bool

	// Errors 获取所有错误
	Errors() []IFieldError

	// HasErrors 是否有错误
	HasErrors() bool

	// Count 错误数量
	Count() int

	// Clear 清空所有错误
	Clear()

	// MaxErrors 最大错误数限制
	MaxErrors() int
}

// IErrorFormatter 错误格式化器接口
// 职责：格式化错误信息
// 设计原则：单一职责
type IErrorFormatter interface {
	// Format 格式化单个错误
	Format(err IFieldError) string

	// FormatAll 格式化所有错误
	FormatAll(errs []IFieldError) []string
}

// IValidationError 验证错误接口
// 职责：封装验证结果和错误列表
// 设计原则：值对象模式
type IValidationError interface {
	// Error 实现 error 接口
	Error() string

	// HasErrors 是否有错误
	HasErrors() bool

	// Errors 获取所有格式化的错误消息
	Errors() []string

	// FieldErrors 获取原始字段错误
	FieldErrors() []IFieldError

	// First 获取第一个错误
	First() string
}

// ============================================================================
// 策略相关接口
// ============================================================================

// StrategyType 验证策略类型
type StrategyType string

const (
	StrategyTypeRule     StrategyType = "rule"     // 规则验证
	StrategyTypeBusiness StrategyType = "business" // 业务验证
	StrategyTypeNested   StrategyType = "nested"   // 嵌套验证
	StrategyTypeCustom   StrategyType = "custom"   // 自定义验证
)

// IValidationStrategy 验证策略接口
// 职责：定义具体的验证策略
// 设计原则：策略模式 - 策略之间完全独立，可自由替换
type IValidationStrategy interface {
	// Type 策略类型
	Type() StrategyType

	// Name 策略名称
	Name() string

	// Validate 执行验证
	// 注意：策略不应该关心优先级，由 Orchestrator 决定执行顺序
	Validate(target any, ctx IContext, collector IErrorCollector) error
}

// IStrategyOrchestrator 策略编排器接口
// 职责：管理和编排验证策略的执行顺序
// 设计原则：责任链模式 + 策略模式
type IStrategyOrchestrator interface {
	// Register 注册策略
	Register(strategy IValidationStrategy, priority int)

	// Unregister 注销策略
	Unregister(strategyType StrategyType)

	// Execute 执行所有策略
	Execute(target any, ctx IContext, collector IErrorCollector) error

	// SetExecutionMode 设置执行模式（串行/并行）
	SetExecutionMode(mode ExecutionMode)
}

// ExecutionMode 策略执行模式
type ExecutionMode int

const (
	ExecutionModeSequential ExecutionMode = iota // 串行执行
	ExecutionModeParallel                        // 并行执行
)

// ============================================================================
// 拦截器接口
// ============================================================================

// IInterceptor 拦截器接口
// 职责：在验证前后执行自定义逻辑
// 设计原则：责任链模式
type IInterceptor interface {
	// Intercept 拦截验证过程
	// next 是下一个拦截器或实际的验证逻辑
	Intercept(ctx IContext, target any, next func() error) error
}

// IInterceptorChain 拦截器链接口
// 职责：管理拦截器链
type IInterceptorChain interface {
	// Add 添加拦截器
	Add(interceptor IInterceptor)

	// Execute 执行拦截器链
	Execute(ctx IContext, target any, validator func() error) error

	// Clear 清空拦截器链
	Clear()
}

// ============================================================================
// 基础设施接口
// ============================================================================

// ITypeInspector 类型检查器接口
// 职责：检查类型信息并缓存
// 设计原则：缓存代理模式
type ITypeInspector interface {
	// Inspect 检查类型信息
	Inspect(target any) ITypeInfo

	// ClearCache 清除缓存
	ClearCache()

	// Stats 获取统计信息
	Stats() CacheStats
}

// ITypeInfo 类型信息接口
// 职责：封装类型的验证能力信息
// 设计原则：值对象模式
type ITypeInfo interface {
	// IsRuleValidator 是否实现了 IRuleValidator
	IsRuleValidator() bool

	// IsBusinessValidator 是否实现了 IBusinessValidator
	IsBusinessValidator() bool

	// IsLifecycleHooks 是否实现了 ILifecycleHooks
	IsLifecycleHooks() bool

	// ValidateRules 获取规则（如果实现了 IRuleValidator）
	ValidateRules(scene Scene) map[string]string

	// FieldAccessor 获取字段访问器
	FieldAccessor(fieldName string) FieldAccessor

	// TypeName 类型名称
	TypeName() string
}

// FieldAccessor 字段访问器类型
// 通过预编译的访问器避免运行时 FieldByName 查找
type FieldAccessor func(value any) (fieldValue any, ok bool)

// ISceneMatcher 场景匹配器接口
// 职责：场景匹配和规则合并
// 设计原则：策略模式
type ISceneMatcher interface {
	// Match 判断场景是否匹配
	Match(target, current Scene) bool

	// MergeRules 合并多个场景的规则
	MergeRules(current Scene, rules map[Scene]map[string]string) map[string]string
}

// IPlaygroundEngine Playground引擎接口
// 职责：抽象的规则验证引擎
// 设计原则：适配器模式 - 适配不同的底层验证库
type IPlaygroundEngine interface {
	// ValidateField 验证单个字段
	ValidateField(value any, rule string) error

	// ValidateStruct 验证整个结构体
	ValidateStruct(target any) error

	// ValidateMap 验证 Map 数据
	ValidateMap(data map[string]interface{}, rules map[string]interface{})

	// RegisterAlias 注册别名
	RegisterAlias(alias, tags string)

	// RegisterValidation 注册自定义验证函数
	RegisterValidation(tag string, fn ValidationFunc) error
}

// ValidationFunc 自定义验证函数类型
type ValidationFunc func(value any, param string) bool

// ============================================================================
// 缓存相关接口
// ============================================================================

// ICacheManager 缓存管理器接口
// 职责：管理各种缓存
// 设计原则：单一职责
type ICacheManager interface {
	// Get 获取缓存
	Get(key any) (value any, ok bool)

	// Set 设置缓存
	Set(key, value any)

	// Delete 删除缓存
	Delete(key any)

	// Clear 清空缓存
	Clear()

	// Stats 获取缓存统计
	Stats() CacheStats
}

// CacheStats 缓存统计信息
type CacheStats struct {
	Hits   int64 // 命中次数
	Misses int64 // 未命中次数
	Size   int   // 当前大小
}

// ICachePolicy 缓存策略接口
// 职责：定义缓存淘汰策略
// TODO:GG 没人实现？
type ICachePolicy interface {
	// ShouldEvict 是否应该淘汰
	ShouldEvict(size, maxSize int) bool

	// SelectVictim 选择淘汰对象
	SelectVictim(keys []any) any
}

// ============================================================================
// 验证管道接口
// ============================================================================

// IValidationPipeline 验证管道接口
// 职责：组合多个验证器
// 设计原则：组合模式
// TODO:GG 没人实现？
type IValidationPipeline interface {
	// Add 添加验证器
	Add(validator IValidator) IValidationPipeline

	// Validate 执行管道验证
	Validate(target any, scene Scene) IValidationError

	// Clear 清空管道
	Clear()
}
