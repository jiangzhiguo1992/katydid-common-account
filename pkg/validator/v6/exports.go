package v6

import (
	"katydid-common-account/pkg/validator/v6/core"
	"katydid-common-account/pkg/validator/v6/errors"
	"katydid-common-account/pkg/validator/v6/orchestration"
)

// ============================================================================
// 导出错误相关函数
// ============================================================================

// NewFieldError 创建字段错误
func NewFieldError(namespace, field, tag string, opts ...errors.FieldErrorOption) core.IFieldError {
	return errors.NewFieldError(namespace, field, tag, opts...)
}

// WithParam 设置验证参数
func WithParam(param string) errors.FieldErrorOption {
	return errors.WithParam(param)
}

// WithValue 设置字段值
func WithValue(value any) errors.FieldErrorOption {
	return errors.WithValue(value)
}

// WithMessage 设置自定义消息
func WithMessage(message string) errors.FieldErrorOption {
	return errors.WithMessage(message)
}

// NewListErrorCollector 创建列表错误收集器
func NewListErrorCollector(maxErrors int) core.IErrorCollector {
	return errors.NewListErrorCollector(maxErrors)
}

// NewMapErrorCollector 创建 Map 错误收集器
func NewMapErrorCollector(maxErrors int) core.IErrorCollector {
	return errors.NewMapErrorCollector(maxErrors)
}

// NewDefaultFormatter 创建默认格式化器
func NewDefaultFormatter() core.IErrorFormatter {
	return errors.NewDefaultFormatter()
}

// NewJSONFormatter 创建 JSON 格式化器
func NewJSONFormatter() core.IErrorFormatter {
	return errors.NewJSONFormatter()
}

// NewDetailedFormatter 创建详细格式化器
func NewDetailedFormatter() core.IErrorFormatter {
	return errors.NewDetailedFormatter()
}

// ============================================================================
// 导出拦截器相关类型
// ============================================================================

// InterceptorFunc 拦截器函数类型
type InterceptorFunc = orchestration.InterceptorFunc

// ============================================================================
// 导出常量
// ============================================================================

// 重新导出场景常量
const (
	SceneNone = core.SceneNone
	SceneAll  = core.SceneAll
)

// 重新导出策略类型
const (
	StrategyTypeRule     = core.StrategyTypeRule
	StrategyTypeBusiness = core.StrategyTypeBusiness
	StrategyTypeNested   = core.StrategyTypeNested
	StrategyTypeCustom   = core.StrategyTypeCustom
)

// 重新导出执行模式
const (
	ExecutionModeSequential = core.ExecutionModeSequential
	ExecutionModeParallel   = core.ExecutionModeParallel
)

// ============================================================================
// 类型别名
// ============================================================================

// Scene 场景类型别名
type Scene = core.Scene

// StrategyType 策略类型别名
type StrategyType = core.StrategyType

// ExecutionMode 执行模式别名
type ExecutionMode = core.ExecutionMode

// Validator 验证器接口别名
type Validator = core.IValidator

// ValidationError 验证错误接口别名
type ValidationError = core.IValidationError

// FieldError 字段错误接口别名
type FieldError = core.IFieldError

// ErrorCollector 错误收集器接口别名
type ErrorCollector = core.IErrorCollector

// Context 上下文接口别名
type Context = core.IContext

// ValidationStrategy 验证策略接口别名
type ValidationStrategy = core.IValidationStrategy

// Interceptor 拦截器接口别名
type Interceptor = core.IInterceptor
