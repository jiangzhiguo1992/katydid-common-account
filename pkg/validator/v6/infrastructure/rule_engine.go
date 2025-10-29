package infrastructure

import (
	"katydid-common-account/pkg/validator/v6/core"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// Playground 规则引擎适配器
// ============================================================================

// playgroundRuleEngine 基于 go-playground/validator 的规则引擎
// 设计模式：适配器模式 - 适配第三方验证库
type playgroundRuleEngine struct {
	validator *validator.Validate
}

// NewPlaygroundRuleEngine 创建 Playground 规则引擎
func NewPlaygroundRuleEngine() core.RuleEngine {
	v := validator.New()

	// 注册 JSON tag 作为字段名
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})

	return &playgroundRuleEngine{
		validator: v,
	}
}

// ValidateField 验证单个字段
func (e *playgroundRuleEngine) ValidateField(value any, rule string) error {
	return e.validator.Var(value, rule)
}

// ValidateStruct 验证整个结构体
func (e *playgroundRuleEngine) ValidateStruct(target any) error {
	return e.validator.Struct(target)
}

// RegisterAlias 注册别名
func (e *playgroundRuleEngine) RegisterAlias(alias, tags string) {
	e.validator.RegisterAlias(alias, tags)
}

// RegisterValidation 注册自定义验证函数
func (e *playgroundRuleEngine) RegisterValidation(tag string, fn core.ValidationFunc) error {
	return e.validator.RegisterValidation(tag, func(fl validator.FieldLevel) bool {
		return fn(fl.Field().Interface(), fl.Param())
	})
}

// GetValidator 获取底层 validator 实例（用于高级用法）
func (e *playgroundRuleEngine) GetValidator() *validator.Validate {
	return e.validator
}
