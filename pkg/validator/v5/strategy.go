package v5

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// 验证策略实现
// ============================================================================

// RuleStrategy 规则验证策略
// 职责：执行基于规则的字段验证（required, min, max等）
// 设计原则：单一职责 - 只负责规则验证
type RuleStrategy struct {
	sceneMatcher SceneMatcher
	typeRegistry TypeRegistry
}

// NewRuleStrategy 创建规则验证策略
func NewRuleStrategy(sceneMatcher SceneMatcher, typeRegistry TypeRegistry) *RuleStrategy {
	// 注册自定义标签名函数，使用 json tag 作为字段名
	typeRegistry.GetValidator().RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})

	return &RuleStrategy{

		sceneMatcher: sceneMatcher,
		typeRegistry: typeRegistry,
	}
}

// Name 策略名称
func (s *RuleStrategy) Name() string {
	return "rule"
}

// Priority 优先级（最高）
func (s *RuleStrategy) Priority() int {
	return 10
}

// Validate 执行规则验证
func (s *RuleStrategy) Validate(target any, ctx *ValidationContext) error {
	if target == nil || ctx == nil {
		return nil
	}

	// 获取场景规则
	var sceneRules map[Scene]map[string]string
	if s.typeRegistry != nil {
		// 从缓存中获取类型信息，直接使用
		if typeInfo := s.typeRegistry.Register(target); typeInfo != nil {
			sceneRules = typeInfo.Rules
		}
	} else {
		// 回退到传统方式：检查是否实现了 RuleValidation 接口
		if provider, ok := target.(RuleValidation); ok {
			sceneRules = provider.ValidateRules()
		}
	}

	// 匹配当前场景的规则
	rules := s.sceneMatcher.MatchRules(ctx.Scene, sceneRules)
	if len(rules) == 0 {
		return nil
	}

	// 是否需要特定规则
	var partial bool

	// 检查是否是部分字段验证
	if fields, ok := ctx.GetMetadata("validate_fields"); ok {
		if fieldList, ok := fields.([]string); ok && (len(fieldList) > 0) {
			partial = true
			rules = s.filterRulesByFields(rules, fieldList)
		}
	}

	// 检查是否需要排除字段
	if excludeFields, ok := ctx.GetMetadata("exclude_fields"); ok {
		if fieldList, ok := excludeFields.([]string); ok && (len(fieldList) > 0) {
			partial = true
			rules = s.excludeRulesFields(rules, fieldList)
		}
	}

	if len(rules) > 0 {
		return s.validateByRules(target, rules, ctx)
	}

	// 没有实现接口，使用 struct tag 验证
	if partial {
		return s.validateByTags(target, rules, ctx)
	} else {
		return s.validateByTags(target, nil, ctx)
	}
}

// validateByRules 使用 RuleValidation 提供的规则验证
func (s *RuleStrategy) validateByRules(target any, rules map[string]string, ctx *ValidationContext) error {
	if len(rules) == 0 {
		return nil
	}

	// 检查是否是部分字段验证
	if fields, ok := ctx.GetMetadata("validate_fields"); ok {
		if fieldList, ok := fields.([]string); ok {
			rules = s.filterRulesByFields(rules, fieldList)
		}
	}

	// 检查是否需要排除字段
	if excludeFields, ok := ctx.GetMetadata("exclude_fields"); ok {
		if fieldList, ok := excludeFields.([]string); ok {
			rules = s.excludeRulesFields(rules, fieldList)
		}
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

	// 逐个字段验证
	for fieldName, rule := range rules {
		if len(fieldName) == 0 || len(rule) == 0 {
			continue
		}

		// 获取字段值
		field := val.FieldByName(fieldName)
		if !field.IsValid() {
			// 尝试通过 JSON tag 查找
			field = s.findFieldByJSONTag(val, fieldName)
		}

		if !field.IsValid() || !field.CanInterface() {
			continue
		}

		// 验证字段
		if err := s.typeRegistry.GetValidator().Var(field.Interface(), rule); err != nil {
			s.addValidationErrors(err, ctx)
		}
	}

	return nil
}

// validateByTags 使用 struct tag 验证
func (s *RuleStrategy) validateByTags(target any, rules map[string]string, ctx *ValidationContext) error {
	if len(rules) == 0 {
		// 没有排除字段，执行完整验证
		if err := s.typeRegistry.GetValidator().Struct(target); err != nil {
			s.addValidationErrors(err, ctx)
		}
		return nil
	}

	// 使用底层验证器的 StructPartial 方法
	partialFields := make([]string, 0, len(rules))
	for fieldName := range rules {
		partialFields = append(partialFields, fieldName)
	}

	if err := s.typeRegistry.GetValidator().StructPartial(target, partialFields...); err != nil {
		s.addValidationErrors(err, ctx)
	}

	return nil
}

// addValidationErrors 添加验证错误
func (s *RuleStrategy) addValidationErrors(err error, ctx *ValidationContext) {
	if err == nil {
		return
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		ctx.AddError(NewFieldError("", "validation_error").
			WithMessage(err.Error()))
		return
	}

	for _, e := range validationErrors {
		ctx.AddError(NewFieldError(e.Namespace(), e.Tag()).WithParam(e.Param()).WithValue(e.Value()))
	}
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

// ============================================================================
// BusinessStrategy 业务验证策略
// ============================================================================

// BusinessStrategy 业务验证策略
// 职责：执行业务逻辑验证
// 设计原则：单一职责
type BusinessStrategy struct{}

// NewBusinessStrategy 创建业务验证策略
func NewBusinessStrategy() *BusinessStrategy {
	return &BusinessStrategy{}
}

// Name 策略名称
func (s *BusinessStrategy) Name() string {
	return "business"
}

// Priority 优先级
func (s *BusinessStrategy) Priority() int {
	return 30
}

// Validate 执行业务验证
func (s *BusinessStrategy) Validate(target any, ctx *ValidationContext) error {
	if target == nil || ctx == nil {
		return nil
	}

	// 检查是否实现了 BusinessValidation 接口
	valid, ok := target.(BusinessValidation)
	if !ok {
		return nil
	}

	// 执行业务验证 (外部利用ctx来AddError)
	return valid.ValidateBusiness(ctx.Scene, ctx)
}

// ============================================================================
// NestedStrategy 嵌套验证策略
// ============================================================================

// NestedStrategy 嵌套验证策略
// 职责：递归验证嵌套的结构体
// 设计原则：单一职责、递归处理
type NestedStrategy struct {
	engine   *ValidatorEngine
	maxDepth int
}

// NewNestedStrategy 创建嵌套验证策略
func NewNestedStrategy(engine *ValidatorEngine, maxDepth int) *NestedStrategy {
	return &NestedStrategy{
		engine:   engine,
		maxDepth: maxDepth,
	}
}

// Name 策略名称
func (s *NestedStrategy) Name() string {
	return "nested"
}

// Priority 优先级
func (s *NestedStrategy) Priority() int {
	return 20
}

// Validate 执行嵌套验证
func (s *NestedStrategy) Validate(target any, ctx *ValidationContext) error {
	if target == nil || ctx == nil {
		return nil
	}

	// 检查嵌套深度
	if ctx.Depth >= s.maxDepth {
		ctx.AddError(NewFieldError("", "max_depth").
			WithMessage(fmt.Sprintf("nested validation depth exceeds maximum limit %d", s.maxDepth)))
		return nil
	}

	// 获取反射值
	val := reflect.ValueOf(target)
	if !val.IsValid() {
		return nil
	}

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	// 遍历所有字段
	typ := val.Type()
	numField := val.NumField()

	for i := 0; i < numField; i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过不可访问的字段
		if !field.CanInterface() || !field.IsValid() {
			continue
		}

		// 跳过 nil 指针
		if field.Kind() == reflect.Ptr && field.IsNil() {
			continue
		}

		// 只处理匿名（嵌入）结构体字段
		fieldKind := field.Kind()
		if fieldKind == reflect.Ptr && !field.IsNil() {
			fieldKind = field.Elem().Kind()
		}

		// 只处理匿名（嵌入）的结构体字段
		if fieldKind == reflect.Struct && fieldType.Anonymous {
			fieldValue := field.Interface()

			// 创建子上下文，保持深度和上下文信息
			subCtx := NewValidationContext(ctx.Scene, fieldValue)
			subCtx.Depth = ctx.Depth + 1
			subCtx.Context = ctx.Context
			// 复制元数据
			if ctx.Metadata != nil {
				subCtx.Metadata = make(map[string]any)
				for k, v := range ctx.Metadata {
					subCtx.Metadata[k] = v
				}
			}

			// 使用子上下文进行递归验证
			if err := s.engine.validateWithContext(fieldValue, subCtx); err != nil {
				// 如果返回错误，直接中断
				return err
			}

			// 将子上下文的错误添加到父上下文
			if subCtx.HasErrors() {
				ctx.AddErrors(subCtx.GetErrors())
			}
		}
	}

	return nil
}
