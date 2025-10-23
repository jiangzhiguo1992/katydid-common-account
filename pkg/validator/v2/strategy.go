package v2

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// 验证策略实现 - 策略模式
// ============================================================================

// RuleValidationStrategy 规则验证策略 - 执行基于规则的字段验证
// 设计原则：
//   - 单一职责：只负责字段规则验证
//   - 开放封闭：可以被替换或扩展，不影响其他策略
type RuleValidationStrategy struct {
	typeCache TypeCache
	validate  *validator.Validate
}

// NewRuleValidationStrategy 创建规则验证策略 - 工厂方法
func NewRuleValidationStrategy(typeCache TypeCache, validate *validator.Validate) *RuleValidationStrategy {
	return &RuleValidationStrategy{
		typeCache: typeCache,
		validate:  validate,
	}
}

// Execute 执行规则验证 - 实现 ValidationStrategy 接口
func (s *RuleValidationStrategy) Execute(obj any, scene Scene, collector ErrorCollector) bool {
	if obj == nil {
		return true // 继续执行后续策略
	}

	// 获取类型信息
	info := s.typeCache.Get(obj)
	if !info.IsRuleValidator {
		// 没有规则，尝试使用 struct tag
		if err := s.validate.Struct(obj); err != nil {
			s.collectValidationErrors(err, collector)
		}
		return true
	}

	// 匹配当前场景的规则
	rules := s.matchSceneRules(info.Rules, scene)
	if len(rules) == 0 {
		return true
	}

	// 验证每个字段
	s.validateFields(obj, rules, collector)

	return true // 继续执行后续策略
}

// matchSceneRules 匹配场景规则 - 私有方法
func (s *RuleValidationStrategy) matchSceneRules(allRules map[Scene]FieldRules, scene Scene) FieldRules {
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
func (s *RuleValidationStrategy) validateFields(obj any, rules FieldRules, collector ErrorCollector) {
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

// findFieldByJSONTag 通过 JSON tag 查找字段
func (s *RuleValidationStrategy) findFieldByJSONTag(val reflect.Value, typ reflect.Type, jsonTag string) reflect.Value {
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
func (s *RuleValidationStrategy) collectValidationErrors(err error, collector ErrorCollector) {
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

// ============================================================================
// CustomValidationStrategy 自定义验证策略 - 执行自定义业务逻辑验证
// ============================================================================

// CustomValidationStrategy 自定义验证策略
// 设计原则：
//   - 单一职责：只负责自定义验证逻辑的执行
//   - 开放封闭：不修改策略本身，通过接口扩展
type CustomValidationStrategy struct {
	typeCache TypeCache
}

// NewCustomValidationStrategy 创建自定义验证策略 - 工厂方法
func NewCustomValidationStrategy(typeCache TypeCache) *CustomValidationStrategy {
	return &CustomValidationStrategy{
		typeCache: typeCache,
	}
}

// Execute 执行自定义验证 - 实现 ValidationStrategy 接口
func (s *CustomValidationStrategy) Execute(obj any, scene Scene, collector ErrorCollector) bool {
	if obj == nil {
		return true
	}

	// 获取类型信息
	info := s.typeCache.Get(obj)
	if !info.IsCustomValidator {
		return true // 没有自定义验证，继续执行后续策略
	}

	// 类型断言
	customValidator, ok := obj.(CustomValidator)
	if !ok {
		return true
	}

	// 执行自定义验证（带 panic 恢复）
	defer func() {
		if r := recover(); r != nil {
			collector.Add(NewFieldError("", "", "validation_panic", "").
				WithMessage(fmt.Sprintf("custom validation panicked: %v", r)))
		}
	}()

	customValidator.ValidateCustom(scene, collector)

	return true // 继续执行后续策略
}

// ============================================================================
// NestedValidationStrategy 嵌套验证策略 - 递归验证嵌套结构
// ============================================================================

// NestedValidationStrategy 嵌套验证策略
// 设计原则：
//   - 单一职责：只负责嵌套结构的递归验证
//   - 防御性编程：防止无限递归和栈溢出
type NestedValidationStrategy struct {
	validator    Validator
	maxDepth     int
	currentDepth int
}

// NewNestedValidationStrategy 创建嵌套验证策略 - 工厂方法
func NewNestedValidationStrategy(validator Validator, maxDepth int) *NestedValidationStrategy {
	if maxDepth <= 0 {
		maxDepth = 100 // 默认最大深度
	}

	return &NestedValidationStrategy{
		validator: validator,
		maxDepth:  maxDepth,
	}
}

// Execute 执行嵌套验证 - 实现 ValidationStrategy 接口
func (s *NestedValidationStrategy) Execute(obj any, scene Scene, collector ErrorCollector) bool {
	if obj == nil {
		return true
	}

	// 防止无限递归
	if s.currentDepth >= s.maxDepth {
		collector.Add(NewFieldError("", "", "max_depth", fmt.Sprintf("%d", s.maxDepth)).
			WithMessage(fmt.Sprintf("nested validation depth exceeds maximum limit %d", s.maxDepth)))
		return false // 停止后续策略
	}

	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return true
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return true
	}

	// 遍历字段
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过不可访问的字段
		if !field.CanInterface() || !field.IsValid() {
			continue
		}

		// 只验证嵌入的结构体字段
		if !fieldType.Anonymous {
			continue
		}

		fieldKind := field.Kind()
		if fieldKind == reflect.Ptr && !field.IsNil() {
			fieldKind = field.Elem().Kind()
		}

		if fieldKind != reflect.Struct {
			continue
		}

		// 递归验证嵌套结构
		s.currentDepth++
		result := s.validator.Validate(field.Interface(), scene)
		s.currentDepth--

		if !result.IsValid() {
			collector.AddAll(result.Errors())
		}
	}

	return true // 继续执行后续策略
}
