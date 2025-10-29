package strategy

import "katydid-common-account/pkg/validator/v5/core"

// BusinessStrategy 业务验证策略
// 职责：执行业务逻辑验证
type BusinessStrategy struct{}

// NewBusinessStrategy 创建业务验证策略
func NewBusinessStrategy() core.IValidationStrategy {
	return &BusinessStrategy{}
}

// Type 策略类型
func (s *BusinessStrategy) Type() core.StrategyType {
	return core.StrategyTypeBusiness
}

// Priority 优先级
func (s *BusinessStrategy) Priority() int8 {
	return 30
}

// Validate 执行业务验证
func (s *BusinessStrategy) Validate(target any, ctx core.IValidationContext) {
	// 检查是否实现了 IBusinessValidation 接口
	valid, ok := target.(core.IBusinessValidation)
	if !ok {
		return
	}

	// 执行业务验证 (外部利用ctx来AddError)
	valid.ValidateBusiness(ctx.Scene(), ctx)
}
