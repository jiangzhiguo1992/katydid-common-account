package registry

import (
	"katydid-common-account/pkg/validator/v5/core"

	"github.com/go-playground/validator/v10"
)

// RuleRegister 规则注册器
type RuleRegister struct {
	validator *validator.Validate
}

// NewRuleRegister 创建规则注册器
func NewRuleRegister(validator *validator.Validate) core.IRuleRegister {
	return &RuleRegister{
		validator: validator,
	}
}

// RegisterAlias 注册别名（聚合Tag）
func (r RuleRegister) RegisterAlias(alias, tags string) {
	r.validator.RegisterAlias(alias, tags)
}

// RegisterValidation 注册自定义验证函数（tag:func）
func (r RuleRegister) RegisterValidation(tag string, fn func()) error {
	return r.validator.RegisterValidation(tag, func(fl validator.FieldLevel) bool {
		fn()
		return true
	})
}
