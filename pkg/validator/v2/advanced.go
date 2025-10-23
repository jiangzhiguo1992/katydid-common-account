package v2

import (
	"fmt"
	"reflect"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// 高级验证功能 - 单一职责：提供高级验证能力
// 设计原则：开放封闭原则，通过接口扩展功能
// ============================================================================

// AdvancedValidator 高级验证器接口
// 扩展基础验证器，提供更多高级功能
type AdvancedValidator interface {
	Validator

	// ValidateWithContext 使用上下文验证
	ValidateWithContext(ctx *ValidationContext, data interface{}) error

	// ValidateNested 验证嵌套结构
	ValidateNested(data interface{}, scene Scene, maxDepth int) error

	// ValidateStruct 验证结构体（底层方法）
	ValidateStruct(data interface{}) error

	// ValidateVar 验证单个变量
	ValidateVar(field interface{}, rule string) error

	// RegisterCustomValidation 注册自定义验证函数
	RegisterCustomValidation(tag string, fn validator.Func) error

	// RegisterAlias 注册验证规则别名
	RegisterAlias(alias string, tags string)

	// ClearTypeCache 清除类型缓存
	ClearTypeCache()

	// GetTypeCacheStats 获取类型缓存统计
	GetTypeCacheStats() TypeCacheStats
}

// advancedValidator 高级验证器实现
type advancedValidator struct {
	*defaultValidator
}

// NewAdvancedValidator 创建高级验证器
func NewAdvancedValidator(opts ...ValidatorOption) (AdvancedValidator, error) {
	base := &defaultValidator{
		validate:  validator.New(),
		typeCache: NewTypeCacheManager(),
		tagName:   "validate",
		maxDepth:  100,
	}

	// 应用选项
	for _, opt := range opts {
		opt(base)
	}

	return &advancedValidator{
		defaultValidator: base,
	}, nil
}

// ValidateWithContext 使用上下文验证
func (v *advancedValidator) ValidateWithContext(ctx *ValidationContext, data interface{}) error {
	if data == nil {
		return fmt.Errorf("验证数据不能为nil")
	}

	if ctx == nil {
		ctx = NewValidationContext(SceneCreate, nil)
		defer ctx.Release()
	}

	// 检查递归深度
	if !ctx.IncrementDepth() {
		return fmt.Errorf("超过最大嵌套深度 %d", ctx.MaxDepth)
	}
	defer ctx.DecrementDepth()

	// 检查循环引用
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		ptr := val.Pointer()
		if ctx.IsVisited(ptr) {
			return nil // 已访问过，跳过
		}
		ctx.MarkVisited(ptr)
	}

	// 获取类型缓存
	typeCache := v.getOrCacheTypeInfo(data)

	// 执行基础验证
	var rules map[string]string
	if typeCache.IsRuleProvider {
		if provider, ok := data.(RuleProvider); ok {
			rules = provider.GetRules(ctx.Scene)
		}
	}

	var baseErr error
	if v.usePool && v.pool != nil {
		validate := v.pool.Get()
		defer v.pool.Put(validate)
		baseErr = v.executeValidation(validate, data, rules)
	} else {
		baseErr = v.executeValidation(v.validate, data, rules)
	}

	// 处理基础验证错误
	if baseErr != nil {
		if errs, ok := baseErr.(validator.ValidationErrors); ok {
			v.processValidationErrors(errs, data, ctx.Errors)
		} else {
			return baseErr
		}
	}

	// 执行自定义验证
	if typeCache.IsCustomValidator {
		if customValidator, ok := data.(CustomValidator); ok {
			customValidator.CustomValidate(ctx.Scene, ctx.Errors)
		}
	}

	// 快速失败检查
	if ctx.ShouldStop() {
		return ctx.GetErrors()
	}

	// 返回错误
	return ctx.GetErrors()
}

// ValidateNested 验证嵌套结构
func (v *advancedValidator) ValidateNested(data interface{}, scene Scene, maxDepth int) error {
	if data == nil {
		return nil
	}

	if maxDepth <= 0 {
		maxDepth = v.GetMaxDepth()
	}

	// 创建嵌套验证器
	nestedValidator := NewNestedValidator(v, maxDepth)
	return nestedValidator.ValidateNested(data, scene, maxDepth)
}

// ValidateStruct 验证结构体（底层方法）
func (v *advancedValidator) ValidateStruct(data interface{}) error {
	if data == nil {
		return fmt.Errorf("验证数据不能为nil")
	}

	var validate *validator.Validate
	if v.usePool && v.pool != nil {
		validate = v.pool.Get()
		defer v.pool.Put(validate)
	} else {
		validate = v.validate
	}

	return validate.Struct(data)
}

// ValidateVar 验证单个变量
func (v *advancedValidator) ValidateVar(field interface{}, rule string) error {
	if rule == "" {
		return nil
	}

	var validate *validator.Validate
	if v.usePool && v.pool != nil {
		validate = v.pool.Get()
		defer v.pool.Put(validate)
	} else {
		validate = v.validate
	}

	return validate.Var(field, rule)
}

// RegisterCustomValidation 注册自定义验证函数
func (v *advancedValidator) RegisterCustomValidation(tag string, fn validator.Func) error {
	if tag == "" || fn == nil {
		return fmt.Errorf("标签和验证函数不能为空")
	}
	return v.validate.RegisterValidation(tag, fn)
}

// RegisterAlias 注册验证规则别名
func (v *advancedValidator) RegisterAlias(alias string, tags string) {
	if alias == "" || tags == "" {
		return
	}
	v.validate.RegisterAlias(alias, tags)
}

// ============================================================================
// 批量验证功能
// ============================================================================

// ValidateBatch 批量验证多个对象
func ValidateBatch(items []interface{}, scene Scene) []error {
	if len(items) == 0 {
		return nil
	}

	errors := make([]error, len(items))
	for i, item := range items {
		errors[i] = Validate(item, scene)
	}

	return errors
}

// ValidateBatchParallel 并行批量验证
func ValidateBatchParallel(items []interface{}, scene Scene) []error {
	if len(items) == 0 {
		return nil
	}

	errors := make([]error, len(items))
	done := make(chan bool, len(items))

	for i, item := range items {
		go func(index int, data interface{}) {
			errors[index] = Validate(data, scene)
			done <- true
		}(i, item)
	}

	// 等待所有验证完成
	for i := 0; i < len(items); i++ {
		<-done
	}

	return errors
}

// ============================================================================
// 条件验证功能
// ============================================================================

// ConditionalValidator 条件验证器
type ConditionalValidator struct {
	validator Validator
}

// NewConditionalValidator 创建条件验证器
func NewConditionalValidator(v Validator) *ConditionalValidator {
	if v == nil {
		v = defaultGlobalValidator
	}
	return &ConditionalValidator{validator: v}
}

// ValidateIf 条件验证
func (cv *ConditionalValidator) ValidateIf(condition bool, data interface{}, scene Scene) error {
	if !condition {
		return nil
	}
	return cv.validator.Validate(data, scene)
}

// ValidateUnless 反向条件验证
func (cv *ConditionalValidator) ValidateUnless(condition bool, data interface{}, scene Scene) error {
	if condition {
		return nil
	}
	return cv.validator.Validate(data, scene)
}

// ValidateIfNotNil 非空验证
func (cv *ConditionalValidator) ValidateIfNotNil(data interface{}, scene Scene) error {
	if data == nil {
		return nil
	}
	return cv.validator.Validate(data, scene)
}
