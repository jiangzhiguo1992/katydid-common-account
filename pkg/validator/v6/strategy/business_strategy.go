package strategy

import (
	"katydid-common-account/pkg/validator/v6/core"
)

// businessStrategy 业务验证策略
// 职责：执行业务逻辑验证
// 设计原则：单一职责 - 只负责业务验证
type businessStrategy struct {
	name      string
	inspector core.TypeInspector
}

// NewBusinessStrategy 创建业务验证策略
func NewBusinessStrategy(inspector core.TypeInspector) core.ValidationStrategy {
	return &businessStrategy{
		name:      "business_strategy",
		inspector: inspector,
	}
}

// Type 策略类型
func (s *businessStrategy) Type() core.StrategyType {
	return core.StrategyTypeBusiness
}

// Name 策略名称
func (s *businessStrategy) Name() string {
	return s.name
}

// Validate 执行业务验证
func (s *businessStrategy) Validate(target any, ctx core.Context, collector core.ErrorCollector) error {
	// 检查类型信息
	typeInfo := s.inspector.Inspect(target)
	if typeInfo == nil || !typeInfo.IsBusinessValidator() {
		return nil
	}

	// 执行业务验证
	if validator, ok := target.(core.BusinessValidator); ok {
		validator.ValidateBusiness(ctx.Scene(), collector)
	}

	return nil
}
