package v2

import "github.com/go-playground/validator/v10"

// ============================================================================
// 核心接口定义 - 接口隔离原则(ISP)：小而精的接口，职责明确
// ============================================================================

// Validator 验证器核心接口 - 定义验证的基本能力
type Validator interface {
	// Validate 执行验证
	Validate(data interface{}, scene Scene) error
	// ValidatePartial 部分字段验证
	ValidatePartial(data interface{}, fields ...string) error
}

// RuleProvider 规则提供者接口 - 单一职责：只负责提供验证规则
type RuleProvider interface {
	// GetRules 获取指定场景的验证规则
	GetRules(scene Scene) map[string]string
}

// CustomValidator 自定义验证器接口 - 单一职责：只负责自定义验证逻辑
type CustomValidator interface {
	// CustomValidate 执行自定义验证逻辑
	CustomValidate(scene Scene, collector ErrorCollector)
}

// ErrorCollector 错误收集器接口 - 单一职责：只负责收集验证错误
type ErrorCollector interface {
	// AddError 添加单个错误
	AddError(field, tag string, params ...interface{})
	// AddFieldError 添加字段错误（更详细）
	AddFieldError(field, tag, param, message string)
	// HasErrors 是否有错误
	HasErrors() bool
	// GetErrors 获取所有错误
	GetErrors() ValidationErrors
	// Clear 清空错误
	Clear()
}

// ErrorMessageProvider 错误消息提供者接口 - 单一职责：只负责提供自定义错误消息
type ErrorMessageProvider interface {
	// GetErrorMessage 获取自定义错误消息
	GetErrorMessage(field, tag, param string) string
}

// ValidationStrategy 验证策略接口 - 策略模式：支持不同的验证策略
type ValidationStrategy interface {
	// Execute 执行验证策略
	Execute(validate *validator.Validate, data interface{}, rules map[string]string) error
}

// CacheManager 缓存管理器接口 - 单一职责：只负责规则缓存
type CacheManager interface {
	// Get 获取缓存的规则
	Get(key string, scene Scene) (map[string]string, bool)
	// Set 设置缓存
	Set(key string, scene Scene, rules map[string]string)
	// Clear 清空缓存
	Clear()
}

// ValidatorPool 验证器池接口 - 单一职责：只负责对象复用
type ValidatorPool interface {
	// Get 获取验证器实例
	Get() *validator.Validate
	// Put 归还验证器实例
	Put(v *validator.Validate)
}

// ErrorFormatter 错误格式化器接口 - 单一职责：只负责错误格式化
type ErrorFormatter interface {
	// Format 格式化验证错误
	Format(err error, provider ErrorMessageProvider) ValidationErrors
}

// ============================================================================
// 组合接口 - 提供更高级的能力
// ============================================================================

// FullValidator 完整验证器接口 - 组合多个能力
type FullValidator interface {
	RuleProvider
	CustomValidator
	ErrorMessageProvider
}

// SceneValidator 场景验证器接口 - 支持多场景验证
type SceneValidator interface {
	RuleProvider
	// GetSupportedScenes 获取支持的场景列表
	GetSupportedScenes() []Scene
}

// ============================================================================
// 配置接口
// ============================================================================

// ValidatorConfig 验证器配置接口 - 单一职责：配置管理
type ValidatorConfig interface {
	// EnableCache 是否启用缓存
	EnableCache() bool
	// EnablePool 是否启用对象池
	EnablePool() bool
	// GetStrategy 获取验证策略
	GetStrategy() ValidationStrategy
	// GetTagName 获取标签名称
	GetTagName() string
}

// ============================================================================
// 构建器接口 - 建造者模式
// ============================================================================

// ValidatorBuilder 验证器构建器接口 - 流式API构建复杂对象
type ValidatorBuilder interface {
	// WithCache 启用缓存
	WithCache(cache CacheManager) ValidatorBuilder
	// WithPool 启用对象池
	WithPool(pool ValidatorPool) ValidatorBuilder
	// WithStrategy 设置验证策略
	WithStrategy(strategy ValidationStrategy) ValidatorBuilder
	// WithErrorFormatter 设置错误格式化器
	WithErrorFormatter(formatter ErrorFormatter) ValidatorBuilder
	// WithTagName 设置标签名称
	WithTagName(tagName string) ValidatorBuilder
	// RegisterCustomValidation 注册自定义验证函数
	RegisterCustomValidation(tag string, fn validator.Func) ValidatorBuilder
	// Build 构建验证器
	Build() (Validator, error)
}
