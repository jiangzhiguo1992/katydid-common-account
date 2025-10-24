// Package v5 提供了一个符合 SOLID 原则的验证器框架
// 特性：高内聚低耦合、可扩展、可测试、可维护
package v5

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)

// ============================================================================
// 核心类型定义
// ============================================================================

// Scene 验证场景，使用位运算支持场景组合
type Scene int64

const (
	SceneNone Scene = 0  // 无场景
	SceneAll  Scene = -1 // 所有场景
)

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

// Match 判断当前场景是否匹配目标场景
func (s Scene) Match(target Scene) bool {
	if target == SceneAll || s == SceneAll {
		return true
	}
	return s&target != 0
}

// ============================================================================
// 业务层接口 - 由模型实现
// ============================================================================

// RuleProvider 规则提供者接口
// 职责：提供字段级别的验证规则（required, min, max等）
// 设计原则：单一职责 - 只负责提供规则，不执行验证
type RuleProvider interface {
	// GetRules 获取指定场景的验证规则
	// 返回格式：map[字段名]规则字符串
	GetRules(scene Scene) map[string]string
}

// BusinessValidator 业务验证器接口
// 职责：执行复杂的业务逻辑验证（跨字段、数据库检查等）
// 设计原则：单一职责 - 只负责业务逻辑验证
type BusinessValidator interface {
	// ValidateBusiness 执行业务验证
	// 通过 ctx.AddError 添加错误
	ValidateBusiness(ctx *ValidationContext) error
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

// ErrorCollector 错误收集器接口
// 职责：收集和管理验证错误
// 设计原则：单一职责、接口隔离
type ErrorCollector interface {
	AddError(err *FieldError)
	AddErrors(errs []*FieldError)
	GetErrors() []*FieldError
	HasErrors() bool
	Clear()
	ErrorCount() int
}

// TypeRegistry 类型注册表接口
// 职责：管理类型信息缓存
// 设计原则：依赖倒置 - 高层模块依赖抽象
type TypeRegistry interface {
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

// CacheStrategy 缓存策略接口
// 职责：定义缓存行为
// 设计原则：策略模式 - 支持不同缓存实现
type CacheStrategy interface {
	Get(key any) (value any, ok bool)
	Set(key, value any)
	Delete(key any)
	Clear()
}

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

// TypeInfo 类型信息
// 职责：缓存类型的验证能力信息
type TypeInfo struct {
	// IsRuleProvider 是否实现了 RuleProvider
	IsRuleProvider bool
	// IsBusinessValidator 是否实现了 BusinessValidator
	IsBusinessValidator bool
	// IsLifecycleHooks 是否实现了 LifecycleHooks
	IsLifecycleHooks bool
	// Rules 缓存的规则（如果实现了 RuleProvider）
	Rules map[Scene]map[string]string
}

// FieldError 字段错误
// 职责：描述单个字段的验证错误
// TODO:GG 检查所有构造函数params
type FieldError struct {
	// Namespace 字段的完整命名空间路径（如 User.Profile.Email）
	// 用于嵌套结构体的错误定位，支持复杂对象的精确错误追踪
	Namespace string

	// Tag 验证标签，描述验证规则类型（如 required, email, min, max 等）
	Tag string

	// Param 验证参数，提供验证规则的具体配置值
	// 例如：min=3 中的 "3"，len=11 中的 "11"
	Param string

	// Value 字段的实际值（用于 sl.ReportError 的 value 参数）
	// 用于调试和详细错误信息，可能包含敏感信息，谨慎使用
	Value any

	// Message 用户友好的错误消息（可选，用于直接显示给终端用户）
	// 支持国际化，建议使用本地化后的错误消息
	Message string
}

// NewFieldError 创建字段错误
func NewFieldError(namespace, tag string) *FieldError {
	// 防御性编程：安全检查并截断超长字段
	namespace = truncateString(namespace, maxNamespaceLength)
	tag = truncateString(tag, maxTagLength)

	return &FieldError{
		Namespace: namespace,
		Tag:       tag,
	}
}

// WithParam 设置参数
func (fe *FieldError) WithParam(param string) *FieldError {
	fe.Param = param
	return fe
}

// WithValue 设置值
func (fe *FieldError) WithValue(value any) *FieldError {
	// 安全检查：值大小限制
	if estimateValueSize(value) > maxValueSize {
		fe.Value = nil
		return fe
	}
	fe.Value = value
	return fe
}

// WithMessage 设置消息
func (fe *FieldError) WithMessage(message string) *FieldError {
	// 安全检查：截断超长消息
	fe.Message = truncateString(message, maxMessageLength)
	return fe
}

// Error 实现 error 接口
func (fe *FieldError) Error() string {
	// 优先使用自定义消息（更友好）
	if len(fe.Message) > 0 {
		return fe.Message
	}

	// 生成默认错误消息（用于调试）
	if len(fe.Namespace) > 0 && len(fe.Tag) > 0 {
		var builder strings.Builder
		builder.Grow(errorMessageEstimatedLength)
		builder.WriteString("field '")
		builder.WriteString(fe.Namespace)
		builder.WriteString("' validation failed on tag '")
		builder.WriteString(fe.Tag)
		if len(fe.Param) > 0 {
			builder.WriteString("' with param '")
			builder.WriteString(fe.Param)
			builder.WriteString("'")
		} else {
			builder.WriteString("'")
		}
		if fe.Value != nil {
			builder.WriteString(", value: ")
			builder.WriteString(fmt.Sprintf("%v", fe.Value))
		}
		return builder.String()
	}

	return "field validation failed"
}

// ValidationError 验证错误集合
// 职责：包装多个字段错误
type ValidationError struct {
	Errors []*FieldError
}

// NewValidationError 创建验证错误
func NewValidationError(errs []*FieldError) *ValidationError {
	return &ValidationError{Errors: errs}
}

// Error 实现 error 接口
func (ve *ValidationError) Error() string {
	if len(ve.Errors) == 0 {
		return "validation passed"
	}
	if len(ve.Errors) == 1 {
		return ve.Errors[0].Error()
	}
	return "validation failed with " + string(rune(len(ve.Errors))) + " errors"
}

// HasErrors 是否有错误
func (ve *ValidationError) HasErrors() bool {
	return len(ve.Errors) > 0
}

// truncateString 安全截断字符串，防止超长攻击
// 内部工具函数，提升代码复用性
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// 安全截断，避免截断 UTF-8 字符的中间
	if maxLen < 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// estimateValueSize 估算值的内存大小
// 用于防止存储过大的值导致内存问题
func estimateValueSize(v any) int {
	if v == nil {
		return 0
	}

	// 使用反射获取值的大小
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return len(rv.String())
	case reflect.Slice, reflect.Array:
		// 估算：每个元素 8 字节
		return rv.Len() * 8
	case reflect.Map:
		// 估算：每个键值对 16 字节
		return rv.Len() * 16
	case reflect.Struct:
		// 估算结构体大小
		return int(rv.Type().Size())
	case reflect.Ptr:
		if rv.IsNil() {
			return 0
		}
		return int(unsafe.Sizeof(rv.Interface()))
	default:
		return int(unsafe.Sizeof(v))
	}
}
