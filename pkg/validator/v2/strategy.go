package v2

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// 验证策略实现 - 策略模式 + 开放封闭原则（OCP）
// ============================================================================

// ruleStrategy 规则验证策略
// 职责：基于 RuleProvider 提供的规则进行验证
type ruleStrategy struct {
	validate *validator.Validate
}

// NewRuleStrategy 创建规则验证策略（工厂方法）
func NewRuleStrategy(v *validator.Validate) ValidationStrategy {
	return &ruleStrategy{validate: v}
}

// Execute 执行规则验证
func (s *ruleStrategy) Execute(obj any, scene ValidateScene, collector ErrorCollector) {
	// 类型检查
	ruleProvider, ok := obj.(RuleProvider)
	if !ok {
		// 不实现 RuleProvider 接口，跳过
		return
	}

	// 获取验证规则
	rules := ruleProvider.GetRules()
	if len(rules) == 0 {
		return
	}

	// 匹配当前场景的规则
	matchedRules := s.matchRules(rules, scene)
	if len(matchedRules) == 0 {
		return
	}

	// 执行验证
	s.validateFields(obj, matchedRules, collector)
}

// matchRules 匹配场景规则
func (s *ruleStrategy) matchRules(rules map[ValidateScene]map[string]string, scene ValidateScene) map[string]string {
	matched := make(map[string]string)

	for ruleScene, fieldRules := range rules {
		// 使用位运算匹配场景
		if scene&ruleScene != 0 {
			for field, rule := range fieldRules {
				matched[field] = rule
			}
		}
	}

	return matched
}

// validateFields 验证字段
func (s *ruleStrategy) validateFields(obj any, rules map[string]string, collector ErrorCollector) {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()

	for fieldName, rule := range rules {
		if rule == "" {
			continue
		}

		// 查找字段
		field := s.findField(val, typ, fieldName)
		if !field.IsValid() || !field.CanInterface() {
			continue
		}

		// 执行验证
		err := s.validate.Var(field.Interface(), rule)
		if err != nil {
			s.addValidationErrors(err, fieldName, collector)
		}
	}
}

// findField 查找字段（支持 JSON tag）
func (s *ruleStrategy) findField(val reflect.Value, typ reflect.Type, fieldName string) reflect.Value {
	// 先按字段名查找
	field := val.FieldByName(fieldName)
	if field.IsValid() {
		return field
	}

	// 按 JSON tag 查找
	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		jsonTag := strings.SplitN(structField.Tag.Get("json"), ",", 2)[0]
		if jsonTag == fieldName {
			return val.Field(i)
		}
	}

	return reflect.Value{}
}

// addValidationErrors 添加验证错误
func (s *ruleStrategy) addValidationErrors(err error, fieldName string, collector ErrorCollector) {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			collector.Add(NewFieldError(
				fieldName,
				e.Tag(),
				e.Error(),
			))
		}
	} else {
		collector.Add(NewFieldError(fieldName, "validation", err.Error()))
	}
}

// businessStrategy 业务验证策略
// 职责：执行 BusinessValidator 的业务逻辑验证
type businessStrategy struct{}

// NewBusinessStrategy 创建业务验证策略（工厂方法）
func NewBusinessStrategy() ValidationStrategy {
	return &businessStrategy{}
}

// Execute 执行业务验证
func (s *businessStrategy) Execute(obj any, scene ValidateScene, collector ErrorCollector) {
	// 类型检查
	businessValidator, ok := obj.(BusinessValidator)
	if !ok {
		// 不实现 BusinessValidator 接口，跳过
		return
	}

	// 执行业务验证
	errors := businessValidator.ValidateBusiness(scene)
	if len(errors) > 0 {
		collector.AddAll(errors)
	}
}

// compositeStrategy 组合策略
// 职责：组合多个验证策略
// 设计模式：组合模式
type compositeStrategy struct {
	strategies []ValidationStrategy
}

// NewCompositeStrategy 创建组合策略（工厂方法）
func NewCompositeStrategy(strategies ...ValidationStrategy) ValidationStrategy {
	return &compositeStrategy{
		strategies: strategies,
	}
}

// Execute 执行所有策略
func (s *compositeStrategy) Execute(obj any, scene ValidateScene, collector ErrorCollector) {
	for _, strategy := range s.strategies {
		if strategy != nil {
			strategy.Execute(obj, scene, collector)
		}
	}
}
