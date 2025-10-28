package facade

import (
	"katydid-common-account/pkg/validator/v6/core"
)

// ValidatorFacade 验证器门面实现
// 职责：提供统一的验证入口，隐藏内部复杂性
// 设计模式：门面模式
type ValidatorFacade struct {
	orchestrator core.ValidationOrchestrator
}

// NewValidatorFacade 创建验证器门面
func NewValidatorFacade(orchestrator core.ValidationOrchestrator) core.Validator {
	return &ValidatorFacade{
		orchestrator: orchestrator,
	}
}

// Validate 验证对象
func (f *ValidatorFacade) Validate(target any, scene core.Scene) error {
	// 创建验证请求
	req := core.NewValidationRequest(target, scene)

	// 执行验证
	result, err := f.orchestrator.Orchestrate(req)
	if err != nil {
		return err
	}

	// 转换为 error
	return result.ToError()
}

// ValidateWithRequest 使用请求对象验证
func (f *ValidatorFacade) ValidateWithRequest(req *core.ValidationRequest) (*core.ValidationResult, error) {
	return f.orchestrator.Orchestrate(req)
}
