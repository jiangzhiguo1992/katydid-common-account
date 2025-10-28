package strategy

import (
	"katydid-common-account/pkg/validator/v6/core"
)

// BusinessStrategy 业务验证策略
// 职责：执行业务逻辑验证
// 设计原则：单一职责 - 只负责业务验证
type BusinessStrategy struct{}

// NewBusinessStrategy 创建业务验证策略
func NewBusinessStrategy() *BusinessStrategy {
	return &BusinessStrategy{}
}

// Name 策略名称
func (s *BusinessStrategy) Name() string {
	return "BusinessStrategy"
}

// Type 策略类型
func (s *BusinessStrategy) Type() core.StrategyType {
	return core.StrategyTypeBusiness
}

// Priority 优先级（中等）
func (s *BusinessStrategy) Priority() int {
	return 20
}

// Validate 执行业务验证
func (s *BusinessStrategy) Validate(req *core.ValidationRequest, ctx core.ValidationContext) error {
	// 检查是否实现了 BusinessValidator 接口
	validator, ok := req.Target.(core.BusinessValidator)
	if !ok {
		// 没有实现接口，跳过
		return nil
	}

	// 执行业务验证
	return validator.ValidateBusiness(req.Scene, ctx)
}
