// Package v5 提供了一个符合 SOLID 原则的验证器框架
// 特性：高内聚低耦合、可扩展、可测试、可维护
package v5

import "github.com/go-playground/validator/v10"

// 预估的错误消息平均长度，用于优化字符串构建时的内存分配
// 通过预分配减少内存重新分配次数，提升性能
const (
	// errorMessageEstimatedLength 单个错误消息的预估长度
	errorMessageEstimatedLength = 80

	// namespaceEstimatedLength 命名空间的预估长度
	namespaceEstimatedLength = 50

	// maxErrorsCapacity 错误列表的最大容量，防止恶意数据导致内存溢出
	maxErrorsCapacity = 1000

	// maxNamespaceLength 最大命名空间长度，防止超长命名空间攻击
	maxNamespaceLength = 512

	// maxTagLength 最大标签长度，防止超长标签攻击
	maxTagLength = 128

	// maxParamLength 最大参数长度，防止超长参数攻击
	maxParamLength = 256

	// maxMessageLength 最大错误消息长度，防止超长消息攻击
	maxMessageLength = 2048

	// maxValueSize 最大值大小（字节），防止存储过大的值导致内存问题
	maxValueSize = 4096
)

// ============================================================================
// 业务层接口 - 由模型实现
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

// ============================================================================
// 框架层接口 - 由框架实现
// ============================================================================

// ValidationStrategy 验证策略接口
// 职责：定义具体的验证策略
// 设计原则：策略模式 - 支持不同的验证策略
type ValidationStrategy interface {
	// Name 策略名称
	Name() string
	// Validate 执行验证
	Validate(target any, ctx *ValidationContext) error
	// Priority 优先级（数字越小优先级越高）
	Priority() int
}

// ErrorHandler 错误收集器接口
// 职责：收集和管理验证错误
// 设计原则：单一职责、接口隔离
type ErrorHandler interface {
	AddError(err *FieldError)
	AddErrors(errs []*FieldError)
	GetErrors() []*FieldError
	HasErrors() bool
	Clear()
	ErrorCount() int
}

// TypeInfo 类型信息
// 职责：缓存类型的验证能力信息
type TypeInfo struct {
	// IsRuleValidator 是否实现了 RuleValidation
	IsRuleValidator bool
	// IsBusinessValidator 是否实现了 BusinessValidation
	IsBusinessValidator bool
	// IsLifecycleHooks 是否实现了 LifecycleHooks
	IsLifecycleHooks bool
	// Rules 缓存的规则（如果实现了 RuleValidation）
	Rules map[Scene]map[string]string
}

// Registry 类型注册表接口
// 职责：管理类型信息缓存
// 设计原则：依赖倒置 - 高层模块依赖抽象
type Registry interface {
	// GetValidator 获取原生validator
	GetValidator() *validator.Validate
	// Register 注册类型信息
	Register(target any) *TypeInfo
	// Get 获取类型信息
	Get(target any) (*TypeInfo, bool)
	// Clear 清除缓存
	Clear()
	// Stats 获取统计信息
	Stats() (count int)
}

// SceneMatcher 场景匹配器接口
// 职责：处理场景匹配逻辑
// 设计原则：单一职责
type SceneMatcher interface {
	// Match 判断场景是否匹配
	Match(current, target Scene) bool
	// MatchRules 匹配并合并规则
	MatchRules(current Scene, rules map[Scene]map[string]string) map[string]string
}

// ErrorFormatter 错误格式化器接口
// 职责：格式化错误信息
// 设计原则：开放封闭 - 支持自定义格式化
type ErrorFormatter interface {
	// Format 格式化单个错误
	Format(err *FieldError) string
	// FormatAll 格式化所有错误
	FormatAll(errs []*FieldError) string
}

// ValidationListener 验证监听器接口
// 职责：监听验证过程中的事件
// 设计原则：观察者模式
type ValidationListener interface {
	// OnValidationStart 验证开始
	OnValidationStart(ctx *ValidationContext)
	// OnValidationEnd 验证结束
	OnValidationEnd(ctx *ValidationContext)
	// OnError 发生错误
	OnError(ctx *ValidationContext, err *FieldError)
}
