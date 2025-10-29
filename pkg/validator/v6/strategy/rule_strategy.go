package strategy

import (
	"katydid-common-account/pkg/validator/v6/context"
	"katydid-common-account/pkg/validator/v6/core"
	"katydid-common-account/pkg/validator/v6/errors"
	"reflect"

	"github.com/go-playground/validator/v10"
)

// ruleStrategy 规则验证策略
// 职责：执行基于规则的字段验证
// 设计原则：单一职责 - 只负责规则验证
type ruleStrategy struct {
	name         string
	ruleEngine   core.IRuleEngine
	inspector    core.ITypeInspector
	sceneMatcher core.ISceneMatcher
}

// NewRuleStrategy 创建规则验证策略
func NewRuleStrategy(
	ruleEngine core.IRuleEngine,
	inspector core.ITypeInspector,
	sceneMatcher core.ISceneMatcher,
) core.IValidationStrategy {
	return &ruleStrategy{
		name:         "rule_strategy",
		ruleEngine:   ruleEngine,
		inspector:    inspector,
		sceneMatcher: sceneMatcher,
	}
}

// Type 策略类型
func (s *ruleStrategy) Type() core.StrategyType {
	return core.StrategyTypeRule
}

// Name 策略名称
func (s *ruleStrategy) Name() string {
	return s.name
}

// Validate 执行规则验证
func (s *ruleStrategy) Validate(target any, ctx core.IContext, collector core.IErrorCollector) error {
	// 检查类型信息
	typeInfo := s.inspector.Inspect(target)
	if typeInfo == nil {
		return nil
	}

	// 获取规则
	var rules map[string]string

	// 优先从类型信息缓存获取
	if typeInfo.IsRuleValidator() {
		if provider, ok := target.(core.IRuleValidator); ok {
			sceneRules := provider.ValidateRules(ctx.Scene())
			rules = sceneRules
		}
	}

	// 如果没有规则，直接返回
	if len(rules) == 0 {
		return nil
	}

	// 处理字段过滤
	rules = s.filterRules(rules, ctx)

	if len(rules) == 0 {
		return nil
	}

	// 执行字段级验证
	s.validateFields(target, rules, typeInfo, collector)

	return nil
}

// validateFields 验证字段
func (s *ruleStrategy) validateFields(
	target any,
	rules map[string]string,
	typeInfo core.ITypeInfo,
	collector core.IErrorCollector,
) {
	// 逐个字段验证
	for fieldName, rule := range rules {
		if len(fieldName) == 0 || len(rule) == 0 {
			continue
		}

		// 获取字段值
		fieldValue, ok := s.getFieldValue(target, fieldName, typeInfo)
		if !ok {
			continue
		}

		// 验证字段
		if err := s.ruleEngine.ValidateField(fieldValue, rule); err != nil {
			// 转换错误
			s.convertAndCollectErrors(err, collector)

			// 如果收集器已满，停止验证
			if collector.Count() >= collector.MaxErrors() {
				break
			}
		}
	}
}

// getFieldValue 获取字段值
func (s *ruleStrategy) getFieldValue(target any, fieldName string, typeInfo core.ITypeInfo) (any, bool) {
	// 优先使用缓存的访问器
	if accessor := typeInfo.FieldAccessor(fieldName); accessor != nil {
		return accessor(target)
	}

	// 回退到反射方式
	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, false
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, false
	}

	// 通过字段名查找
	field := val.FieldByName(fieldName)
	if !field.IsValid() || !field.CanInterface() {
		return nil, false
	}

	return field.Interface(), true
}

// convertAndCollectErrors 转换并收集错误
func (s *ruleStrategy) convertAndCollectErrors(err error, collector core.IErrorCollector) {
	// 检查是否是 validator.ValidationErrors
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			fieldErr := errors.NewFieldError(
				e.Namespace(),
				e.Field(),
				e.Tag(),
				errors.WithParam(e.Param()),
				errors.WithValue(e.Value()),
				errors.WithMessage(e.Error()),
			)

			// 收集错误
			if !collector.Collect(fieldErr) {
				break
			}
		}
	} else {
		// 其他类型的错误
		fieldErr := errors.NewFieldError(
			"",
			"",
			"error",
			errors.WithMessage(err.Error()),
		)
		collector.Collect(fieldErr)
	}
}

// filterRules 过滤规则
func (s *ruleStrategy) filterRules(rules map[string]string, ctx core.IContext) map[string]string {
	// 检查是否需要只验证指定字段
	if fields, ok := ctx.Metadata().Get(context.MetadataKeyValidateFields); ok {
		if fieldList, ok := fields.([]string); ok && len(fieldList) > 0 {
			return s.includeFields(rules, fieldList)
		}
	}

	// 检查是否需要排除字段
	if excludeFields, ok := ctx.Metadata().Get(context.MetadataKeyExcludeFields); ok {
		if fieldList, ok := excludeFields.([]string); ok && len(fieldList) > 0 {
			return s.excludeFields(rules, fieldList)
		}
	}

	return rules
}

// includeFields 只包含指定字段
func (s *ruleStrategy) includeFields(rules map[string]string, fields []string) map[string]string {
	filtered := make(map[string]string)
	fieldSet := make(map[string]bool)
	for _, f := range fields {
		fieldSet[f] = true
	}

	for field, rule := range rules {
		if fieldSet[field] {
			filtered[field] = rule
		}
	}

	return filtered
}

// excludeFields 排除指定字段
func (s *ruleStrategy) excludeFields(rules map[string]string, fields []string) map[string]string {
	filtered := make(map[string]string)
	excludeSet := make(map[string]bool)
	for _, f := range fields {
		excludeSet[f] = true
	}

	for field, rule := range rules {
		if !excludeSet[field] {
			filtered[field] = rule
		}
	}

	return filtered
}
