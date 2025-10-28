package strategy

import v5 "katydid-common-account/pkg/validator/v5"

// BusinessStrategy 业务验证策略
// 职责：执行业务逻辑验证
type BusinessStrategy struct{}

// NewBusinessStrategy 创建业务验证策略
func NewBusinessStrategy() *BusinessStrategy {
	return &BusinessStrategy{}
}

// Type 策略类型
func (s *BusinessStrategy) Type() v5.StrategyType {
	return v5.StrategyTypeBusiness
}

// Priority 优先级
func (s *BusinessStrategy) Priority() int8 {
	return 30
}

// Validate 执行业务验证
func (s *BusinessStrategy) Validate(target any, ctx *v5.ValidationContext) error {
	// 检查是否实现了 BusinessValidation 接口
	valid, ok := target.(v5.BusinessValidation)
	if !ok {
		return nil
	}

	// 执行业务验证 (外部利用ctx来AddError)
	return valid.ValidateBusiness(ctx.Scene, ctx)
}
