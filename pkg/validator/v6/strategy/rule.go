package strategy

import (
	"reflect"
	"strings"

	"katydid-common-account/pkg/validator/v6/core"

	"github.com/go-playground/validator/v10"
)

// RuleStrategy 规则验证策略
// 职责：执行基于规则的字段验证（required, min, max等）
// 设计原则：单一职责 - 只负责规则验证
type RuleStrategy struct {
	validator    *validator.Validate
	sceneMatcher core.SceneMatcher
	registry     core.TypeRegistry
}

// NewRuleStrategy 创建规则验证策略
func NewRuleStrategy(sceneMatcher core.SceneMatcher, registry core.TypeRegistry) *RuleStrategy {
	v := validator.New()

	// 注册自定义标签名函数，使用 json tag 作为字段名
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})

	return &RuleStrategy{
		validator:    v,
		sceneMatcher: sceneMatcher,
		registry:     registry,
	}
}

// Name 策略名称
func (s *RuleStrategy) Name() string {
	return "RuleStrategy"
}

// Type 策略类型
func (s *RuleStrategy) Type() core.StrategyType {
	return core.StrategyTypeRule
}

// Priority 优先级（最高）
func (s *RuleStrategy) Priority() int {
	return 10
}

// Validate 执行规则验证
func (s *RuleStrategy) Validate(req *core.ValidationRequest, ctx core.ValidationContext) error {
	// 1. 注册类型信息
	typeInfo := s.registry.Register(req.Target)
	if typeInfo == nil || !typeInfo.HasRuleValidation() {
		// 没有规则验证，跳过
		return nil
	}

	// 2. 获取规则
	allRules := typeInfo.GetRules()
	if len(allRules) == 0 {
		return nil
	}

	// 3. 匹配当前场景的规则
	rules := s.sceneMatcher.MatchRules(req.Scene, allRules)
	if len(rules) == 0 {
		return nil
	}

	// 4. 过滤规则（如果指定了字段）
	rules = s.filterRules(rules, req.Fields, req.ExcludeFields)

	// 5. 执行验证
	return s.validateByRules(req.Target, rules, ctx)
}

// validateByRules 使用规则验证
func (s *RuleStrategy) validateByRules(target any, rules map[string]string, ctx core.ValidationContext) error {
	if len(rules) == 0 {
		return nil
	}

	// 获取对象的反射值
	val := reflect.ValueOf(target)
	if !val.IsValid() {
		return nil
	}

	// 处理指针类型
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	// 只处理结构体类型
	if val.Kind() != reflect.Struct {
		return nil
	}

	// 获取类型信息
	typeInfo := s.registry.Register(target)

	// 逐个字段验证
	for fieldName, rule := range rules {
		if len(fieldName) == 0 || len(rule) == 0 {
			continue
		}

		// 使用字段访问器获取字段值（性能优化）
		var fieldValue any
		var exists bool
		if accessor := typeInfo.GetFieldAccessor(fieldName); accessor != nil {
			fieldValue, exists = accessor(target)
		} else {
			// 回退到反射方式
			fieldValue, exists = s.getFieldValue(val, fieldName)
		}

		if !exists {
			continue
		}

		// 验证字段
		err := s.validator.Var(fieldValue, rule)
		if err != nil {
			// 转换为 FieldError
			if validationErrs, ok := err.(validator.ValidationErrors); ok {
				for _, e := range validationErrs {
					fieldErr := core.NewFieldError(fieldName, e.Tag()).
						WithField(fieldName).
						WithParam(e.Param()).
						WithValue(e.Value())

					// 添加到错误收集器
					if !ctx.ErrorCollector().Add(fieldErr) {
						// 达到最大错误数，停止验证
						return nil
					}
				}
			}
		}
	}

	return nil
}

// getFieldValue 获取字段值（回退方案）
func (s *RuleStrategy) getFieldValue(val reflect.Value, fieldName string) (any, bool) {
	// 首先尝试直接字段名
	fieldVal := val.FieldByName(fieldName)
	if fieldVal.IsValid() {
		return fieldVal.Interface(), true
	}

	// 尝试通过 JSON tag 查找
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			tagName := strings.Split(jsonTag, ",")[0]
			if tagName == fieldName {
				fieldVal := val.Field(i)
				if fieldVal.IsValid() {
					return fieldVal.Interface(), true
				}
			}
		}
	}

	return nil, false
}

// filterRules 过滤规则
func (s *RuleStrategy) filterRules(rules map[string]string, includeFields, excludeFields []string) map[string]string {
	// 如果指定了包含字段
	if len(includeFields) > 0 {
		filtered := make(map[string]string)
		includeSet := makeSet(includeFields)
		for field, rule := range rules {
			if includeSet[field] {
				filtered[field] = rule
			}
		}
		rules = filtered
	}

	// 如果指定了排除字段
	if len(excludeFields) > 0 {
		excludeSet := makeSet(excludeFields)
		for field := range excludeSet {
			delete(rules, field)
		}
	}

	return rules
}

// makeSet 创建集合
func makeSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}
