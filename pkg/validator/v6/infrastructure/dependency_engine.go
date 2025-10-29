package infrastructure

import (
	"katydid-common-account/pkg/validator/v6/core"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// 依赖库 规则引擎适配器
// ============================================================================

// dependencyEngine 基于 go-playground/validator 的规则引擎
// 设计模式：适配器模式 - 适配第三方验证库
type dependencyEngine struct {
	validator *validator.Validate
}

// NewDependencyEngine 创建 依赖库 规则引擎
func NewDependencyEngine() core.IDependencyEngine {
	v := validator.New()

	// 注册 JSON tag 作为字段名
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})

	return &dependencyEngine{
		validator: v,
	}
}

// ValidateField 验证单个字段
func (e *dependencyEngine) ValidateField(value any, rule string) error {
	return e.validator.Var(value, rule)
}

// ValidateStruct 验证整个结构体
func (e *dependencyEngine) ValidateStruct(target any) error {
	return e.validator.Struct(target)
}

// ValidateMap 验证 Map 数据
// TODO:GG 外部怎么调用这个方法？
// TODO:GG 这个怎么搞场景化
func (e *dependencyEngine) ValidateMap(data map[string]interface{}, rules map[string]interface{}) {
	e.validator.ValidateMap(data, rules)
}

// RegisterAlias 注册别名
// TODO:GG 外部怎么调用这个方法？
func (e *dependencyEngine) RegisterAlias(alias, tags string) {
	e.validator.RegisterAlias(alias, tags)
}

// RegisterValidation 注册自定义验证函数
// TODO:GG 外部怎么调用这个方法？
func (e *dependencyEngine) RegisterValidation(tag string, fn core.ValidationFunc) error {
	return e.validator.RegisterValidation(tag, func(fl validator.FieldLevel) bool {
		return fn(fl.Field().Interface(), fl.Param())
	})
}

// GetValidator 获取底层 validator 实例（用于高级用法）
func (e *dependencyEngine) GetValidator() *validator.Validate {
	return e.validator
}
