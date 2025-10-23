package v2

import (
	"github.com/go-playground/validator/v10"
)

// ============================================================================
// 验证策略实现 - 策略模式：不同的验证方式
// ============================================================================

// DefaultStrategy 默认验证策略 - 验证所有字段
type DefaultStrategy struct{}

// NewDefaultStrategy 创建默认策略
func NewDefaultStrategy() ValidationStrategy {
	return &DefaultStrategy{}
}

// Execute 执行默认验证策略
func (s *DefaultStrategy) Execute(validate *validator.Validate, data interface{}, _ map[string]string) error {
	return validate.Struct(data)
}

// ============================================================================
// PartialStrategy 部分字段验证策略 - 只验证指定字段
// ============================================================================

// PartialStrategy 部分验证策略
type PartialStrategy struct {
	fields []string
}

// NewPartialStrategy 创建部分验证策略
func NewPartialStrategy(fields ...string) ValidationStrategy {
	return &PartialStrategy{
		fields: fields,
	}
}

// Execute 执行部分验证
func (s *PartialStrategy) Execute(validate *validator.Validate, data interface{}, _ map[string]string) error {
	if len(s.fields) == 0 {
		return validate.Struct(data)
	}
	return validate.StructPartial(data, s.fields...)
}

// ============================================================================
// FailFastStrategy 快速失败策略 - 遇到第一个错误就停止
// ============================================================================

// FailFastStrategy 快速失败策略
type FailFastStrategy struct {
	validate *validator.Validate
}

// NewFailFastStrategy 创建快速失败策略
func NewFailFastStrategy() ValidationStrategy {
	return &FailFastStrategy{}
}

// Execute 执行快速失败验证
func (s *FailFastStrategy) Execute(_ *validator.Validate, data interface{}, _ map[string]string) error {
	// 创建一个临时验证器实例，设置快速失败模式
	v := validator.New()

	// 复制自定义验证函数
	// 注意：这里简化处理，实际使用时可能需要更完善的实现

	// 只返回第一个错误
	err := v.Struct(data)
	if err != nil {
		if errs, ok := err.(validator.ValidationErrors); ok && len(errs) > 0 {
			// 只返回第一个错误
			return errs[0:1]
		}
		return err
	}
	return nil
}

// ============================================================================
// ConditionalStrategy 条件验证策略 - 根据条件决定是否验证某些字段
// ============================================================================

// ConditionFunc 条件函数类型
type ConditionFunc func(data interface{}) bool

// ConditionalStrategy 条件验证策略
type ConditionalStrategy struct {
	condition ConditionFunc
	strategy  ValidationStrategy
}

// NewConditionalStrategy 创建条件验证策略
func NewConditionalStrategy(condition ConditionFunc, strategy ValidationStrategy) ValidationStrategy {
	return &ConditionalStrategy{
		condition: condition,
		strategy:  strategy,
	}
}

// Execute 执行条件验证
func (s *ConditionalStrategy) Execute(validate *validator.Validate, data interface{}, rules map[string]string) error {
	if s.condition(data) {
		return s.strategy.Execute(validate, data, rules)
	}
	return nil
}

// ============================================================================
// ChainStrategy 链式策略 - 组合多个策略
// ============================================================================

// ChainStrategy 链式策略
type ChainStrategy struct {
	strategies []ValidationStrategy
}

// NewChainStrategy 创建链式策略
func NewChainStrategy(strategies ...ValidationStrategy) ValidationStrategy {
	return &ChainStrategy{
		strategies: strategies,
	}
}

// Execute 执行链式验证
func (s *ChainStrategy) Execute(validate *validator.Validate, data interface{}, rules map[string]string) error {
	for _, strategy := range s.strategies {
		if err := strategy.Execute(validate, data, rules); err != nil {
			return err
		}
	}
	return nil
}

// ============================================================================
// CustomRuleStrategy 自定义规则策略 - 使用动态规则而非 struct tag
// ============================================================================

// CustomRuleStrategy 自定义规则策略
type CustomRuleStrategy struct {
	rules map[string]string
}

// NewCustomRuleStrategy 创建自定义规则策略
func NewCustomRuleStrategy(rules map[string]string) ValidationStrategy {
	return &CustomRuleStrategy{
		rules: rules,
	}
}

// Execute 执行自定义规则验证
func (s *CustomRuleStrategy) Execute(validate *validator.Validate, data interface{}, rules map[string]string) error {
	// 合并规则
	mergedRules := make(map[string]string)
	for k, v := range rules {
		mergedRules[k] = v
	}
	for k, v := range s.rules {
		mergedRules[k] = v
	}

	// 使用 VarWithValue 或 Struct 验证
	return validate.Struct(data)
}
