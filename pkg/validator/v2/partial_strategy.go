package v2

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// PartialValidationStrategy 部分验证策略 - 支持字段级别的选择性验证
// 设计模式：策略模式（Strategy Pattern）
// 设计原则：
//   - 单一职责原则（SRP）：只负责部分字段的验证逻辑
//   - 开放封闭原则（OCP）：可扩展新的字段过滤逻辑
// ============================================================================

// PartialValidationStrategy 部分验证策略
// 灵活性：支持包含模式和排除模式
type PartialValidationStrategy struct {
	typeCache TypeCache
	validate  *validator.Validate
	fieldSet  map[string]bool // 字段集合
	isExclude bool            // true: 排除模式，false: 包含模式
}

// NewPartialValidationStrategy 创建部分验证策略 - 工厂方法
// 参数：
//   - typeCache: 类型缓存
//   - validate: 底层验证器
//   - fieldSet: 字段集合
//   - isExclude: true 表示排除这些字段，false 表示只验证这些字段
func NewPartialValidationStrategy(
	typeCache TypeCache,
	validate *validator.Validate,
	fieldSet map[string]bool,
	isExclude bool,
) *PartialValidationStrategy {
	return &PartialValidationStrategy{
		typeCache: typeCache,
		validate:  validate,
		fieldSet:  fieldSet,
		isExclude: isExclude,
	}
}

// Execute 执行部分验证 - 实现 ValidationStrategy 接口
func (s *PartialValidationStrategy) Execute(obj any, scene Scene, collector ErrorCollector) bool {
	if obj == nil {
		return true
	}

	// 获取类型信息
	info := s.typeCache.Get(obj)
	if !info.IsRuleValidator {
		// 没有规则，尝试使用 struct tag（部分验证）
		s.validateStructPartial(obj, collector)
		return true
	}

	// 匹配当前场景的规则
	rules := s.matchSceneRules(info.Rules, scene)
	if len(rules) == 0 {
		return true
	}

	// 根据模式过滤规则
	filteredRules := s.filterRules(rules)

	// 验证过滤后的字段
	s.validateFields(obj, filteredRules, collector)

	return true // 继续执行后续策略
}

// filterRules 根据包含/排除模式过滤规则 - 私有方法
func (s *PartialValidationStrategy) filterRules(rules FieldRules) FieldRules {
	if len(s.fieldSet) == 0 {
		return rules
	}

	result := make(FieldRules)

	if s.isExclude {
		// 排除模式：保留不在集合中的字段
		for field, rule := range rules {
			if !s.fieldSet[field] {
				result[field] = rule
			}
		}
	} else {
		// 包含模式：只保留在集合中的字段
		for field, rule := range rules {
			if s.fieldSet[field] {
				result[field] = rule
			}
		}
	}

	return result
}

// validateStructPartial 部分验证结构体（使用 struct tag） - 私有方法
func (s *PartialValidationStrategy) validateStructPartial(obj any, collector ErrorCollector) {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()

	// 遍历字段
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过不可访问的字段
		if !field.CanInterface() || !field.IsValid() {
			continue
		}

		// 获取字段名（优先使用 JSON tag）
		fieldName := s.getFieldName(fieldType)

		// 根据模式判断是否需要验证
		shouldValidate := s.shouldValidateField(fieldName)
		if !shouldValidate {
			continue
		}

		// 获取验证标签
		validateTag := fieldType.Tag.Get("validate")
		if validateTag == "" || validateTag == "-" {
			continue
		}

		// 验证字段
		if err := s.validate.Var(field.Interface(), validateTag); err != nil {
			s.collectValidationErrors(err, collector)
		}
	}
}

// shouldValidateField 判断字段是否应该被验证 - 私有方法
func (s *PartialValidationStrategy) shouldValidateField(fieldName string) bool {
	if len(s.fieldSet) == 0 {
		return true
	}

	inSet := s.fieldSet[fieldName]

	if s.isExclude {
		// 排除模式：不在集合中的才验证
		return !inSet
	}

	// 包含模式：在集合中的才验证
	return inSet
}

// getFieldName 获取字段名（优先使用 JSON tag） - 私有方法
func (s *PartialValidationStrategy) getFieldName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" && jsonTag != "-" {
		// 提取 tag 名称（忽略选项）
		tagName := strings.SplitN(jsonTag, ",", 2)[0]
		if tagName != "" {
			return tagName
		}
	}
	return field.Name
}

// matchSceneRules 匹配场景规则 - 私有方法
func (s *PartialValidationStrategy) matchSceneRules(allRules map[Scene]FieldRules, scene Scene) FieldRules {
	result := make(FieldRules)

	// 直接匹配场景
	if rules, ok := allRules[scene]; ok {
		for field, rule := range rules {
			result[field] = rule
		}
	}

	return result
}

// validateFields 验证字段 - 私有方法
func (s *PartialValidationStrategy) validateFields(obj any, rules FieldRules, collector ErrorCollector) {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
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

		// 先尝试通过字段名查找
		field := val.FieldByName(fieldName)

		// 如果找不到，尝试通过 JSON tag 查找
		if !field.IsValid() {
			field = s.findFieldByJSONTag(val, typ, fieldName)
		}

		if !field.IsValid() || !field.CanInterface() {
			continue
		}

		// 验证字段
		if err := s.validate.Var(field.Interface(), rule); err != nil {
			s.collectValidationErrors(err, collector)
		}
	}
}

// findFieldByJSONTag 通过 JSON tag 查找字段 - 私有方法
func (s *PartialValidationStrategy) findFieldByJSONTag(val reflect.Value, typ reflect.Type, jsonTag string) reflect.Value {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" {
			continue
		}

		// 提取 tag 名称（忽略选项）
		tagName := strings.SplitN(tag, ",", 2)[0]
		if tagName == jsonTag {
			return val.Field(i)
		}
	}

	return reflect.Value{}
}

// collectValidationErrors 收集验证错误 - 私有方法
func (s *PartialValidationStrategy) collectValidationErrors(err error, collector ErrorCollector) {
	if err == nil {
		return
	}

	// 转换为 ValidationErrors
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			fieldErr := NewFieldError(
				e.Namespace(),
				e.Field(),
				e.Tag(),
				e.Param(),
			).WithValue(e.Value())

			collector.Add(fieldErr)
		}
	} else {
		// 其他类型的错误
		collector.Add(NewFieldError("", "", "validation_error", "").
			WithMessage(err.Error()))
	}
}
