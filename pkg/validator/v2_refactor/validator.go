package v2

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// 验证器实现
// ============================================================================

// Validator 验证器结构
type Validator struct {
	validate        *validator.Validate
	typeCache       *typeCacheManager
	registeredTypes sync.Map // 已注册的自定义验证类型
}

var (
	// 默认验证器实例
	defaultValidator *Validator
	once             sync.Once
)

// Default 获取默认验证器实例（单例模式）
func Default() *Validator {
	once.Do(func() {
		defaultValidator = New()
	})
	return defaultValidator
}

// New 创建新的验证器实例
func New() *Validator {
	v := validator.New()

	// 注册自定义标签名函数，使用 json tag 作为字段名
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})

	return &Validator{
		validate:  v,
		typeCache: newTypeCacheManager(),
	}
}

// ============================================================================
// 全局便捷函数
// ============================================================================

// Validate 使用默认验证器验证对象
func Validate(obj interface{}, scene Scene) ValidationErrors {
	return Default().Validate(obj, scene)
}

// ValidateFields 使用默认验证器验证指定字段
func ValidateFields(obj interface{}, scene Scene, fields ...string) ValidationErrors {
	return Default().ValidateFields(obj, scene, fields...)
}

// ValidateExcept 使用默认验证器验证排除字段外的所有字段
func ValidateExcept(obj interface{}, scene Scene, excludeFields ...string) ValidationErrors {
	return Default().ValidateExcept(obj, scene, excludeFields...)
}

// RegisterAlias 在默认验证器上注册别名
func RegisterAlias(alias, tags string) {
	Default().RegisterAlias(alias, tags)
}

// ClearTypeCache 清除默认验证器的类型缓存
func ClearTypeCache() {
	Default().ClearTypeCache()
}

// ============================================================================
// 验证器方法
// ============================================================================

// Validate 验证对象
func (v *Validator) Validate(obj interface{}, scene Scene) ValidationErrors {
	// 防御性检查
	if obj == nil {
		return ValidationErrors{
			NewFieldError("struct", "required", "").
				WithMessage("validation target cannot be nil"),
		}
	}

	// 获取类型缓存
	cache := v.typeCache.getOrCacheTypeInfo(obj)

	// 注册自定义验证器（仅首次）
	if cache.isCustomValidator {
		v.registerStructValidator(obj)
	}

	// 创建错误收集器
	collector := getErrorCollector()
	defer putErrorCollector(collector)

	// 1. 执行字段规则验证
	if cache.isRuleProvider {
		v.validateFieldsByRules(obj, cache.validationRules, scene, collector)
	} else {
		v.validateFieldsByTags(obj, collector)
	}

	// 2. 递归验证嵌套结构体
	v.validateNestedStructs(obj, scene, collector, 0)

	// 3. 执行自定义验证
	if cache.isCustomValidator {
		if customValidator, ok := obj.(CustomValidator); ok {
			customValidator.CustomValidation(scene, collector.Report)
		}
	}

	// 返回验证结果
	if collector.HasErrors() {
		return collector.GetErrors()
	}

	return nil
}

// ValidateFields 验证指定字段
func (v *Validator) ValidateFields(obj interface{}, scene Scene, fields ...string) ValidationErrors {
	if obj == nil || len(fields) == 0 {
		return nil
	}

	// 获取类型缓存
	cache := v.typeCache.getOrCacheTypeInfo(obj)

	// 创建错误收集器
	collector := getErrorCollector()
	defer putErrorCollector(collector)

	// 获取验证规则
	var rules map[string]string
	if cache.isRuleProvider && cache.validationRules != nil {
		rules = v.getRulesForScene(cache.validationRules, scene)
	}

	// 过滤出需要验证的字段规则
	fieldRules := make(map[string]string)
	for _, field := range fields {
		if rule, ok := rules[field]; ok {
			fieldRules[field] = rule
		}
	}

	// 如果有规则，执行验证
	if len(fieldRules) > 0 {
		if err := v.validateWithRules(obj, fieldRules); err != nil {
			v.processValidationErrors(err, obj, collector)
		}
	} else {
		// 回退到 struct tag 验证
		if err := v.validate.StructPartial(obj, fields...); err != nil {
			v.processValidationErrors(err, obj, collector)
		}
	}

	if collector.HasErrors() {
		return collector.GetErrors()
	}

	return nil
}

// ValidateExcept 验证排除字段外的所有字段
func (v *Validator) ValidateExcept(obj interface{}, scene Scene, excludeFields ...string) ValidationErrors {
	if obj == nil {
		return nil
	}

	if len(excludeFields) == 0 {
		return v.Validate(obj, scene)
	}

	// 获取类型缓存
	cache := v.typeCache.getOrCacheTypeInfo(obj)

	// 创建错误收集器
	collector := getErrorCollector()
	defer putErrorCollector(collector)

	// 获取验证规则
	var rules map[string]string
	if cache.isRuleProvider && cache.validationRules != nil {
		rules = v.getRulesForScene(cache.validationRules, scene)
	}

	// 创建排除字段集合
	excludeSet := make(map[string]bool, len(excludeFields))
	for _, field := range excludeFields {
		excludeSet[field] = true
	}

	// 过滤规则
	filteredRules := make(map[string]string)
	for field, rule := range rules {
		if !excludeSet[field] {
			filteredRules[field] = rule
		}
	}

	// 执行验证
	if len(filteredRules) > 0 {
		if err := v.validateWithRules(obj, filteredRules); err != nil {
			v.processValidationErrors(err, obj, collector)
		}
	}

	// 执行自定义验证
	if cache.isCustomValidator {
		if customValidator, ok := obj.(CustomValidator); ok {
			customValidator.CustomValidation(scene, collector.Report)
		}
	}

	if collector.HasErrors() {
		return collector.GetErrors()
	}

	return nil
}

// RegisterAlias 注册验证标签别名
func (v *Validator) RegisterAlias(alias, tags string) {
	if len(alias) == 0 || len(tags) == 0 {
		return
	}
	v.validate.RegisterAlias(alias, tags)
}

// ClearTypeCache 清除类型缓存
func (v *Validator) ClearTypeCache() {
	v.typeCache.Clear()
	v.registeredTypes = sync.Map{}
}

// ============================================================================
// 内部辅助方法
// ============================================================================

// getRulesForScene 获取指定场景的规则
func (v *Validator) getRulesForScene(allRules map[Scene]map[string]string, scene Scene) map[string]string {
	// 优先使用精确匹配的场景规则
	if rules, ok := allRules[scene]; ok {
		return rules
	}

	// 查找包含当前场景的组合场景
	for s, rules := range allRules {
		if s.Has(scene) {
			return rules
		}
	}

	// 使用 SceneAll 的规则
	if rules, ok := allRules[SceneAll]; ok {
		return rules
	}

	return nil
}

// validateFieldsByRules 使用动态规则验证字段
func (v *Validator) validateFieldsByRules(obj interface{}, allRules map[Scene]map[string]string, scene Scene, collector ErrorCollector) {
	rules := v.getRulesForScene(allRules, scene)
	if rules == nil || len(rules) == 0 {
		return
	}

	if err := v.validateWithRules(obj, rules); err != nil {
		v.processValidationErrors(err, obj, collector)
	}
}

// validateFieldsByTags 使用 struct tag 验证字段
func (v *Validator) validateFieldsByTags(obj interface{}, collector ErrorCollector) {
	if err := v.validate.Struct(obj); err != nil {
		v.processValidationErrors(err, obj, collector)
	}
}

// validateWithRules 使用指定规则验证
func (v *Validator) validateWithRules(obj interface{}, rules map[string]string) error {
	objValue := reflect.ValueOf(obj)
	if objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
	}

	if objValue.Kind() != reflect.Struct {
		return fmt.Errorf("validation target must be a struct")
	}

	// 逐字段验证
	for fieldName, rule := range rules {
		field := objValue.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}

		// 执行验证
		if err := v.validate.Var(field.Interface(), rule); err != nil {
			return err
		}
	}

	return nil
}

// validateNestedStructs 递归验证嵌套结构体
func (v *Validator) validateNestedStructs(obj interface{}, scene Scene, collector ErrorCollector, depth int) {
	// 防止无限递归
	if depth >= MaxNestedDepth {
		return
	}

	objValue := reflect.ValueOf(obj)
	if objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
	}

	if objValue.Kind() != reflect.Struct {
		return
	}

	objType := objValue.Type()

	// 遍历所有字段
	for i := 0; i < objValue.NumField(); i++ {
		field := objValue.Field(i)
		fieldType := objType.Field(i)

		// 跳过未导出字段
		if !fieldType.IsExported() {
			continue
		}

		// 处理指针类型
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				continue
			}
			field = field.Elem()
		}

		// 只处理结构体
		if field.Kind() != reflect.Struct {
			continue
		}

		// 递归验证嵌套结构体
		nestedObj := field.Interface()
		nestedCache := v.typeCache.getOrCacheTypeInfo(nestedObj)

		if nestedCache.isRuleProvider {
			v.validateFieldsByRules(nestedObj, nestedCache.validationRules, scene, collector)
		}

		if nestedCache.isCustomValidator {
			if customValidator, ok := nestedObj.(CustomValidator); ok {
				customValidator.CustomValidation(scene, collector.Report)
			}
		}

		// 继续递归
		v.validateNestedStructs(nestedObj, scene, collector, depth+1)
	}
}

// registerStructValidator 注册结构体验证器
func (v *Validator) registerStructValidator(obj interface{}) {
	objType := reflect.TypeOf(obj)

	// 检查是否已注册
	if _, loaded := v.registeredTypes.LoadOrStore(objType, true); loaded {
		return
	}

	// 注册自定义验证函数
	v.validate.RegisterStructValidation(func(sl validator.StructLevel) {
		// 这里不执行任何操作，仅用于标记
		// 实际的自定义验证在 CustomValidation 中执行
	}, obj)
}

// processValidationErrors 处理验证错误
func (v *Validator) processValidationErrors(err error, obj interface{}, collector ErrorCollector) {
	if err == nil {
		return
	}

	// 处理 validator.ValidationErrors
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range validationErrors {
			// 从 go-playground/validator 获取完整的 namespace
			namespace := fieldErr.Namespace()
			field := fieldErr.Field()

			collector.AddError(&FieldError{
				Namespace: namespace,
				Field:     field,
				Tag:       fieldErr.Tag(),
				Param:     fieldErr.Param(),
				Value:     fieldErr.Value(),
			})
		}
	}
}
