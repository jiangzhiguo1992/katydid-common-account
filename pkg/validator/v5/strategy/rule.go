package strategy

import (
	"katydid-common-account/pkg/validator/v5/core"
	"katydid-common-account/pkg/validator/v5/err"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// RuleStrategy 规则验证策略
// 职责：执行基于规则的字段验证（required, min, max等）
type RuleStrategy struct {
	validator    *validator.Validate
	sceneMatcher core.ISceneMatcher
	registry     core.ITypeRegistry
}

// NewRuleStrategy 创建规则验证策略
func NewRuleStrategy(validator *validator.Validate, sceneMatcher core.ISceneMatcher, registry core.ITypeRegistry) core.IValidationStrategy {
	// 注册自定义标签名函数，使用 json tag 作为字段名
	validator.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})

	return &RuleStrategy{
		validator:    validator,
		sceneMatcher: sceneMatcher,
		registry:     registry,
	}
}

// Type 策略类型
func (s *RuleStrategy) Type() core.StrategyType {
	return core.StrategyTypeRule
}

// Priority 优先级（最高）
func (s *RuleStrategy) Priority() int8 {
	return 10
}

// Validate 执行规则验证
func (s *RuleStrategy) Validate(target any, ctx core.IValidationContext) {
	// 获取场景规则
	var sceneRules map[core.Scene]map[string]string
	if s.registry != nil {
		// 从缓存中获取类型信息，直接使用
		if typeInfo := s.registry.Register(target); typeInfo != nil {
			sceneRules = typeInfo.Rules()
		}
	} else {
		// 回退到传统方式：检查是否实现了 IRuleValidation 接口
		if provider, ok := target.(core.IRuleValidation); ok {
			sceneRules = provider.ValidateRules()
		}
	}

	// 匹配当前场景的规则
	rules := s.sceneMatcher.MatchRules(ctx.Scene(), sceneRules)
	if len(rules) == 0 {
		return
	}

	// 检查是否是部分字段验证
	if fields, ok := ctx.GetMetadata(core.MetadataKeyValidateFields); ok {
		if fieldList, ok := fields.([]string); ok && (len(fieldList) > 0) {
			rules = s.filterRulesByFields(rules, fieldList)
		}
	}

	// 检查是否需要排除字段
	if excludeFields, ok := ctx.GetMetadata(core.MetadataKeyExcludeFields); ok {
		if fieldList, ok := excludeFields.([]string); ok && (len(fieldList) > 0) {
			rules = s.excludeRulesFields(rules, fieldList)
		}
	}

	// 没必要validateByTags了，直接走 validateByRules(性能更好)
	s.validateByRules(target, rules, ctx)
}

// validateByRules 使用 IRuleValidation 提供的规则验证
func (s *RuleStrategy) validateByRules(target any, rules map[string]string, ctx core.IValidationContext) {
	if len(rules) == 0 {
		return
	}

	// 检查是否是部分字段验证
	if fields, ok := ctx.GetMetadata(core.MetadataKeyValidateFields); ok {
		if fieldList, ok := fields.([]string); ok {
			rules = s.filterRulesByFields(rules, fieldList)
		}
	}

	// 检查是否需要排除字段
	if excludeFields, ok := ctx.GetMetadata(core.MetadataKeyExcludeFields); ok {
		if fieldList, ok := excludeFields.([]string); ok {
			rules = s.excludeRulesFields(rules, fieldList)
		}
	}

	// 获取对象的反射值
	val := reflect.ValueOf(target)
	if !val.IsValid() {
		return
	}

	// 处理指针类型
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}

	// 只处理结构体类型
	if val.Kind() != reflect.Struct {
		return
	}

	// 优化：获取类型信息（包含字段访问器缓存）
	typeInfo := s.registry.Register(target)

	// 逐个字段验证
	for fieldName, rule := range rules {
		if len(fieldName) == 0 || len(rule) == 0 {
			continue
		}

		var field reflect.Value

		// 优化：优先使用缓存的访问器（O(1) 访问）
		var accessor core.FieldAccessor
		if typeInfo != nil {
			accessor = typeInfo.FieldAccessor(fieldName)
		}
		if accessor != nil {
			field = accessor(val)
		} else {
			// 回退到传统方式：通过字段名查找（O(n) 访问）
			field = val.FieldByName(fieldName)
			if !field.IsValid() {
				// 尝试通过 JSON tag 查找
				field = s.findFieldByJSONTag(val, fieldName)
			}
		}

		if !field.IsValid() || !field.CanInterface() {
			continue
		}

		// 验证字段
		if e := s.validator.Var(field.Interface(), rule); e != nil {
			if !s.addValidationErrors(e, ctx) {
				break
			}
		}
	}

	return
}

// validateByTags 使用 struct tag 验证
func (s *RuleStrategy) validateByTags(target any, rules map[string]string, ctx core.IValidationContext) {
	if len(rules) == 0 {
		// 没有排除字段，执行完整验证
		// Struct()内部本质还是validator.Var()
		if e := s.validator.Struct(target); e != nil {
			s.addValidationErrors(e, ctx)
		}
		return
	}

	// 使用底层验证器的 StructPartial 方法
	partialFields := make([]string, 0, len(rules))
	for fieldName := range rules {
		partialFields = append(partialFields, fieldName)
	}

	// StructPartial()内部本质还是validator.Var()
	if e := s.validator.StructPartial(target, partialFields...); e != nil {
		s.addValidationErrors(e, ctx)
	}

	return
}

// addValidationErrors 添加验证错误
func (s *RuleStrategy) addValidationErrors(error error, ctx core.IValidationContext) bool {
	validationErrors, ok := error.(validator.ValidationErrors)
	if !ok {
		return ctx.AddError(err.NewFieldErrorWithMessage(error.Error()))
	}

	for _, e := range validationErrors {
		if !ctx.AddError(err.NewFieldError(
			e.Namespace(), e.Tag(),
			err.WithParam(e.Param()),
			err.WithValue(e.Value()),
			err.WithMessage(e.Error()),
		)) {
			return false
		}
	}
	return true
}

// filterRulesByFields 过滤规则，只保留指定字段
func (s *RuleStrategy) filterRulesByFields(rules map[string]string, fields []string) map[string]string {
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

// excludeRulesFields 排除指定字段
func (s *RuleStrategy) excludeRulesFields(rules map[string]string, excludeFields []string) map[string]string {
	filtered := make(map[string]string)
	excludeSet := make(map[string]bool)
	for _, f := range excludeFields {
		excludeSet[f] = true
	}

	for field, rule := range rules {
		if !excludeSet[field] {
			filtered[field] = rule
		}
	}
	return filtered
}

// findFieldByJSONTag 通过 JSON tag 查找字段
func (s *RuleStrategy) findFieldByJSONTag(val reflect.Value, jsonTag string) reflect.Value {
	typ := val.Type()
	if typ == nil {
		return reflect.Value{}
	}

	numField := typ.NumField()
	for i := 0; i < numField; i++ {
		fieldType := typ.Field(i)
		// 提取 json tag 的第一部分（逗号前）
		tag := strings.SplitN(fieldType.Tag.Get("json"), ",", 2)[0]
		if tag == jsonTag {
			field := val.Field(i)
			// 确保字段可访问
			if field.CanInterface() {
				return field
			}
		}
	}

	// 未找到，返回零值
	return reflect.Value{}
}
