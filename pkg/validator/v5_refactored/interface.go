package v5_refactored

import "reflect"

// ============================================================================
// 核心验证器接口
// ============================================================================

// Validator 验证器核心接口
// 职责：提供验证功能的统一入口
// 设计原则：接口隔离 - 只暴露必要的方法
type Validator interface {
	// Validate 执行完整验证
	Validate(target any, scene Scene) *ValidationError

	// ValidateFields 验证指定字段
	ValidateFields(target any, scene Scene, fields ...string) *ValidationError

	// ValidateFieldsExcept 验证除指定字段外的所有字段
	ValidateFieldsExcept(target any, scene Scene, fields ...string) *ValidationError
}

// ============================================================================
// 业务验证接口 (用户实现)
// ============================================================================

// RuleProvider 规则提供者接口
// 职责：提供字段级别的验证规则
// 设计原则：单一职责 - 只负责提供规则，不执行验证
type RuleProvider interface {
	// GetRules 获取指定场景的验证规则
	// 返回格式：map[字段名]规则字符串
	GetRules(scene Scene) map[string]string
}

// BusinessValidator 自定义业务验证器接口
// 职责：执行复杂的业务逻辑验证（跨字段、数据库检查等）
// 设计原则：单一职责 - 只负责业务逻辑验证
type BusinessValidator interface {
	// ValidateBusiness 执行业务验证
	// 通过 collector.Add 添加错误
	ValidateBusiness(scene Scene, ctx *ValidationContext, collector ErrorCollector) error
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

// ============================================================================
// 策略接口
// ============================================================================

// ValidationStrategy 验证策略接口
// 职责：定义具体的验证策略
// 设计原则：策略模式 - 支持不同的验证策略
type ValidationStrategy interface {
	// Type 策略类型
	Type() StrategyType

	// Priority 优先级（数字越小优先级越高）
	Priority() int8

	// Validate 执行验证
	Validate(target any, ctx *ValidationContext, collector ErrorCollector) error

	// Name 策略名称（用于调试和日志）
	Name() string
}

// StrategyType 策略类型
type StrategyType int8

const (
	StrategyTypeRule StrategyType = iota + 1
	StrategyTypeBusiness
	StrategyTypeNested
	StrategyTypeCustom
)

// ============================================================================
// 管道执行器接口
// ============================================================================

// PipelineExecutor 管道执行器接口
// 职责：编排和执行验证策略
// 设计原则：单一职责 - 只负责策略的编排和执行
type PipelineExecutor interface {
	// Execute 执行验证管道
	Execute(target any, ctx *ValidationContext, collector ErrorCollector) error

	// AddStrategy 添加策略
	AddStrategy(strategy ValidationStrategy)

	// RemoveStrategy 移除策略
	RemoveStrategy(strategyType StrategyType)

	// GetStrategies 获取所有策略
	GetStrategies() []ValidationStrategy
}

// ============================================================================
// 事件总线接口
// ============================================================================

// EventBus 事件总线接口
// 职责：事件发布订阅，解耦组件
// 设计原则：观察者模式 - 支持事件驱动
type EventBus interface {
	// Subscribe 订阅事件
	Subscribe(listener EventListener)

	// Unsubscribe 取消订阅
	Unsubscribe(listener EventListener)

	// Publish 发布事件
	Publish(event Event)

	// Clear 清空所有监听器
	Clear()
}

// EventListener 事件监听器接口
// 职责：监听和处理事件
type EventListener interface {
	// OnEvent 事件处理
	OnEvent(event Event)

	// EventTypes 感兴趣的事件类型（空表示所有事件）
	EventTypes() []EventType
}

// Event 事件接口
type Event interface {
	// Type 事件类型
	Type() EventType

	// Context 获取验证上下文
	Context() *ValidationContext

	// Timestamp 事件时间戳
	Timestamp() int64

	// Data 事件数据
	Data() map[string]any
}

// EventType 事件类型
type EventType int8

const (
	EventValidationStart EventType = iota + 1
	EventValidationEnd
	EventStrategyStart
	EventStrategyEnd
	EventFieldValidated
	EventErrorOccurred
	EventHookBefore
	EventHookAfter
)

// ============================================================================
// 钩子管理器接口
// ============================================================================

// HookManager 钩子管理器接口
// 职责：管理生命周期钩子
// 设计原则：单一职责 - 只负责钩子的管理和执行
type HookManager interface {
	// ExecuteBefore 执行前置钩子
	ExecuteBefore(target any, ctx *ValidationContext) error

	// ExecuteAfter 执行后置钩子
	ExecuteAfter(target any, ctx *ValidationContext) error

	// RegisterHook 注册钩子（可选，用于动态注册）
	RegisterHook(target any, hooks LifecycleHooks)

	// UnregisterHook 取消注册钩子（可选）
	UnregisterHook(target any)
}

// ============================================================================
// 错误收集器接口
// ============================================================================

// ErrorCollector 错误收集器接口
// 职责：收集和管理验证错误
// 设计原则：单一职责 - 只负责错误收集
type ErrorCollector interface {
	// Add 添加错误，返回 false 表示已达到最大错误数
	Add(err *FieldError) bool

	// GetAll 获取所有错误
	GetAll() []*FieldError

	// GetByField 按字段获取错误
	GetByField(field string) []*FieldError

	// HasErrors 是否有错误
	HasErrors() bool

	// Count 错误数量
	Count() int

	// Clear 清空错误
	Clear()

	// IsFull 是否已满（达到最大错误数）
	IsFull() bool
}

// ============================================================================
// 类型注册表接口
// ============================================================================

// TypeInfoReader 类型信息读取器
// 职责：读取类型信息
// 设计原则：接口隔离 - 只读操作
type TypeInfoReader interface {
	// Get 获取类型信息
	Get(typ reflect.Type) (*TypeInfo, bool)
}

// TypeInfoWriter 类型信息写入器
// 职责：写入类型信息
// 设计原则：接口隔离 - 只写操作
type TypeInfoWriter interface {
	// Set 设置类型信息
	Set(typ reflect.Type, info *TypeInfo)
}

// TypeInfoCache 类型信息缓存
// 职责：缓存类型信息
// 设计原则：组合 Reader 和 Writer
type TypeInfoCache interface {
	TypeInfoReader
	TypeInfoWriter

	// Clear 清空缓存
	Clear()

	// Stats 获取统计信息
	Stats() CacheStats
}

// TypeAnalyzer 类型分析器
// 职责：分析类型信息
// 设计原则：单一职责 - 只负责分析
type TypeAnalyzer interface {
	// Analyze 分析类型
	Analyze(target any) *TypeInfo
}

// TypeRegistry 类型注册表
// 职责：组合类型缓存和分析能力
// 设计原则：组合复用
type TypeRegistry interface {
	TypeInfoCache
	TypeAnalyzer

	// Register 注册并缓存类型信息
	Register(target any) *TypeInfo
}

// CacheStats 缓存统计
type CacheStats struct {
	// HitCount 命中次数
	HitCount int64
	// MissCount 未命中次数
	MissCount int64
	// Size 缓存大小
	Size int
}

// ============================================================================
// 场景匹配器接口
// ============================================================================

// SceneMatcher 场景匹配器接口
// 职责：场景匹配逻辑
// 设计原则：策略模式 - 支持不同的匹配策略
type SceneMatcher interface {
	// Match 判断场景是否匹配
	Match(current, target Scene) bool

	// MatchRules 匹配并合并规则
	MatchRules(current Scene, rules map[Scene]map[string]string) map[string]string
}

// ============================================================================
// 错误格式化器接口
// ============================================================================

// ErrorFormatter 错误格式化器接口
// 职责：格式化错误信息
// 设计原则：策略模式 - 支持不同的格式化策略
type ErrorFormatter interface {
	// Format 格式化单个错误
	Format(err *FieldError) string

	// FormatAll 格式化所有错误
	FormatAll(errs []*FieldError) string
}

// ============================================================================
// 工厂接口
// ============================================================================

// ValidatorFactory 验证器工厂接口
// 职责：创建验证器实例
// 设计原则：工厂模式 - 封装创建逻辑
type ValidatorFactory interface {
	// Create 创建验证器
	Create(opts ...EngineOption) Validator

	// CreateDefault 创建默认验证器
	CreateDefault() Validator
}

// ============================================================================
// 建造者接口
// ============================================================================

// ValidatorBuilder 验证器建造者接口
// 职责：提供流畅的 API 构建验证器
// 设计原则：建造者模式 - 支持复杂配置
type ValidatorBuilder interface {
	// WithPipeline 设置管道执行器
	WithPipeline(pipeline PipelineExecutor) ValidatorBuilder

	// WithEventBus 设置事件总线
	WithEventBus(bus EventBus) ValidatorBuilder

	// WithHookManager 设置钩子管理器
	WithHookManager(manager HookManager) ValidatorBuilder

	// WithRegistry 设置类型注册表
	WithRegistry(registry TypeRegistry) ValidatorBuilder

	// WithErrorCollectorFactory 设置错误收集器工厂
	WithErrorCollectorFactory(factory ErrorCollectorFactory) ValidatorBuilder

	// WithErrorFormatter 设置错误格式化器
	WithErrorFormatter(formatter ErrorFormatter) ValidatorBuilder

	// WithMaxErrors 设置最大错误数
	WithMaxErrors(max int) ValidatorBuilder

	// WithMaxDepth 设置最大嵌套深度
	WithMaxDepth(depth int) ValidatorBuilder

	// Build 构建验证器
	Build() Validator
}

// ErrorCollectorFactory 错误收集器工厂
// 职责：创建错误收集器实例
type ErrorCollectorFactory interface {
	// Create 创建错误收集器
	Create(maxErrors int) ErrorCollector
}
