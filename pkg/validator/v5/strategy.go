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
	validate     *validator.Validate
	sceneMatcher SceneMatcher
}

// NewRuleStrategy 创建规则验证策略
func NewRuleStrategy(sceneMatcher SceneMatcher) *RuleStrategy {
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
		validate:     v,
		sceneMatcher: sceneMatcher,
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

	// 检查是否实现了 RuleValidator 接口
	provider, ok := target.(RuleValidator)
	if !ok {
		// 没有实现接口，使用 struct tag 验证
		return s.validateByTags(target, ctx)
	}

	// 使用 RuleValidator 提供的规则验证
	return s.validateByRules(target, provider, ctx)
}

// validateByRules 使用 RuleValidator 提供的规则验证
func (s *RuleStrategy) validateByRules(target any, provider RuleValidator, ctx *ValidationContext) error {
	// 获取场景规则
	sceneRules := provider.ValidateRule()
	if len(sceneRules) == 0 {
		return nil
	}

	// 匹配当前场景的规则
	rules := s.sceneMatcher.MatchRules(ctx.Scene, sceneRules)
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
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	// 逐个字段验证
	for fieldName, rule := range rules {
		if rule == "" {
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
		if err := s.validate.Var(field.Interface(), rule); err != nil {
			s.addValidationErrors(err, ctx)
		}
	}

	return nil
}

// validateByTags 使用 struct tag 验证
func (s *RuleStrategy) validateByTags(target any, ctx *ValidationContext) error {
	if err := s.validate.Struct(target); err != nil {
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
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if tag == jsonTag {
			return val.Field(i)
		}
	}
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

	// 检查是否实现了 BusinessValidator 接口
	validator, ok := target.(BusinessValidator)
	if !ok {
		return nil
	}

	// 执行业务验证
	return validator.ValidateBusiness(ctx)
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
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过不可访问的字段
		if !field.CanInterface() {
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

		if fieldKind == reflect.Struct && fieldType.Anonymous {
			// 创建子上下文
			subCtx := NewValidationContext(ctx.Scene, field.Interface())
			subCtx.Depth = ctx.Depth + 1
			subCtx.Context = ctx.Context

			// 递归验证
			if err := s.engine.Validate(field.Interface(), ctx.Scene); err != nil {
				if ve, ok := err.(*ValidationError); ok {
					ctx.AddErrors(ve.Errors)
				}
			}
		}
	}

	return nil
}

// ============================================================================
// ValidationPipeline 验证管道
// ============================================================================

// ValidationPipeline 验证管道
// 职责：按顺序执行多个验证器
// 设计模式：责任链模式
type ValidationPipeline struct {
	validators []ValidationStrategy
}

// NewValidationPipeline 创建验证管道
func NewValidationPipeline() *ValidationPipeline {
	return &ValidationPipeline{
		validators: make([]ValidationStrategy, 0),
	}
}

// Add 添加验证器
func (p *ValidationPipeline) Add(validator ValidationStrategy) *ValidationPipeline {
	p.validators = append(p.validators, validator)
	return p
}

// Execute 执行管道
func (p *ValidationPipeline) Execute(target any, ctx *ValidationContext) error {
	for _, v := range p.validators {
		if err := v.Validate(target, ctx); err != nil {
			return err
		}
	}
	return nil
}
