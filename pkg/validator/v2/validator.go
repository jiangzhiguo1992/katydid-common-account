package v2

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// 核心验证器实现 - 组合各个组件，遵循依赖倒置原则
// ============================================================================

// defaultValidator 默认验证器实现
type defaultValidator struct {
	validate       *validator.Validate
	cache          CacheManager
	pool           ValidatorPool
	strategy       ValidationStrategy
	errorFormatter ErrorFormatter
	tagName        string
	useCache       bool
	usePool        bool
}

// Validate 执行验证
func (v *defaultValidator) Validate(data interface{}, scene Scene) error {
	if data == nil {
		return fmt.Errorf("验证数据不能为nil")
	}

	// 获取类型名称
	typeName := getTypeName(data)

	// 获取验证规则
	rules := v.getRules(data, scene, typeName)

	// 执行基础验证
	var baseErr error
	if v.usePool && v.pool != nil {
		validate := v.pool.Get()
		defer v.pool.Put(validate)
		baseErr = v.executeValidation(validate, data, rules)
	} else {
		baseErr = v.executeValidation(v.validate, data, rules)
	}

	// 收集错误
	collector := GetPooledErrorCollector()
	defer PutPooledErrorCollector(collector)

	// 处理基础验证错误
	if baseErr != nil {
		if errs, ok := baseErr.(validator.ValidationErrors); ok {
			v.processValidationErrors(errs, data, collector)
		} else {
			return baseErr
		}
	}

	// 执行自定义验证
	if customValidator, ok := data.(CustomValidator); ok {
		customValidator.CustomValidate(scene, collector)
	}

	// 返回错误
	if collector.HasErrors() {
		return collector.GetErrors()
	}

	return nil
}

// ValidatePartial 部分字段验证
func (v *defaultValidator) ValidatePartial(data interface{}, fields ...string) error {
	if data == nil {
		return fmt.Errorf("验证数据不能为nil")
	}

	if len(fields) == 0 {
		return nil
	}

	// 获取所有规则（使用默认 SceneCreate 场景）
	var allRules map[string]string
	if provider, ok := data.(RuleProvider); ok {
		allRules = provider.GetRules(SceneCreate)
	}

	// 如果没有规则，回退到 struct tag 验证
	if allRules == nil || len(allRules) == 0 {
		var err error
		if v.usePool && v.pool != nil {
			validate := v.pool.Get()
			defer v.pool.Put(validate)
			err = validate.StructPartial(data, fields...)
		} else {
			err = v.validate.StructPartial(data, fields...)
		}

		if err == nil {
			return nil
		}

		// 格式化错误
		collector := GetPooledErrorCollector()
		defer PutPooledErrorCollector(collector)

		if errs, ok := err.(validator.ValidationErrors); ok {
			v.processValidationErrors(errs, data, collector)
		} else {
			return err
		}

		if collector.HasErrors() {
			return collector.GetErrors()
		}

		return nil
	}

	// 使用动态规则，只验证指定字段
	partialRules := make(map[string]string)
	for _, field := range fields {
		if rule, ok := allRules[field]; ok {
			partialRules[field] = rule
		}
	}

	if len(partialRules) == 0 {
		return nil
	}

	// 执行验证
	var validate *validator.Validate
	if v.usePool && v.pool != nil {
		validate = v.pool.Get()
		defer v.pool.Put(validate)
	} else {
		validate = v.validate
	}

	return v.validateWithRules(validate, data, partialRules)
}

// getRules 获取验证规则（带缓存）
func (v *defaultValidator) getRules(data interface{}, scene Scene, typeName string) map[string]string {
	// 尝试从缓存获取
	if v.useCache && v.cache != nil {
		if rules, ok := v.cache.Get(typeName, scene); ok {
			return rules
		}
	}

	// 从数据对象获取规则
	var rules map[string]string
	if provider, ok := data.(RuleProvider); ok {
		rules = provider.GetRules(scene)
	}

	// 缓存规则
	if v.useCache && v.cache != nil && rules != nil {
		v.cache.Set(typeName, scene, rules)
	}

	return rules
}

// executeValidation 执行验证
func (v *defaultValidator) executeValidation(validate *validator.Validate, data interface{}, rules map[string]string) error {
	// 如果有规则，使用动态验证
	if rules != nil && len(rules) > 0 {
		return v.validateWithRules(validate, data, rules)
	}

	// 否则使用策略或默认验证
	if v.strategy != nil {
		return v.strategy.Execute(validate, data, rules)
	}
	return validate.Struct(data)
}

// validateWithRules 使用动态规则验证
func (v *defaultValidator) validateWithRules(validate *validator.Validate, data interface{}, rules map[string]string) error {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("data must be a struct")
	}

	collector := NewErrorCollector()
	typ := val.Type()

	// 遍历规则并验证每个字段
	for fieldName, rule := range rules {
		field := val.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}

		// 获取字段的 JSON tag 名称（用于错误消息）
		structField, found := typ.FieldByName(fieldName)
		if !found {
			continue
		}

		displayName := fieldName
		if jsonTag := structField.Tag.Get("json"); jsonTag != "" {
			if idx := strings.Index(jsonTag, ","); idx > 0 {
				displayName = jsonTag[:idx]
			} else {
				displayName = jsonTag
			}
		}

		// 检查是否为 required 字段
		isRequired := strings.Contains(rule, "required")
		fieldValue := field.Interface()

		// 检查零值
		isZero := false
		switch field.Kind() {
		case reflect.String:
			isZero = field.String() == ""
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			isZero = field.Int() == 0
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			isZero = field.Uint() == 0
		case reflect.Float32, reflect.Float64:
			isZero = field.Float() == 0
		case reflect.Bool:
			isZero = !field.Bool()
		case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
			isZero = field.IsNil()
		}

		// 如果是必填字段且为零值，添加 required 错误
		if isRequired && isZero {
			collector.AddFieldError(displayName, "required", "", "")
			continue
		}

		// 如果不是必填字段且为零值，跳过验证（omitempty 语义）
		if !isRequired && isZero {
			continue
		}

		// 验证字段
		err := validate.Var(fieldValue, rule)
		if err != nil {
			if errs, ok := err.(validator.ValidationErrors); ok {
				for _, e := range errs {
					collector.AddFieldError(displayName, e.Tag(), e.Param(), "")
				}
			}
		}
	}

	if collector.HasErrors() {
		return collector.(interface{ GetErrors() ValidationErrors }).GetErrors()
	}

	return nil
}

// processValidationErrors 处理验证错误
func (v *defaultValidator) processValidationErrors(errs validator.ValidationErrors, data interface{}, collector ErrorCollector) {
	var msgProvider ErrorMessageProvider
	if provider, ok := data.(ErrorMessageProvider); ok {
		msgProvider = provider
	}

	for _, err := range errs {
		field := err.Field()
		tag := err.Tag()
		param := err.Param()

		// 获取自定义消息
		var message string
		if msgProvider != nil {
			message = msgProvider.GetErrorMessage(field, tag, param)
		}

		collector.AddFieldError(field, tag, param, message)
	}
}

// ============================================================================
// 辅助函数
// ============================================================================

// getTypeName 获取类型名称
func getTypeName(data interface{}) string {
	t := reflect.TypeOf(data)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// ============================================================================
// 多场景验证器 - 支持map数据验证
// ============================================================================

// MapValidator Map数据验证器
type MapValidator struct {
	validate       *validator.Validate
	cache          CacheManager
	errorFormatter ErrorFormatter
}

// NewMapValidator 创建Map验证器
func NewMapValidator(opts ...ValidatorOption) *MapValidator {
	mv := &MapValidator{
		validate: validator.New(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(mv)
	}

	return mv
}

// ValidateMap 验证Map数据
func (v *MapValidator) ValidateMap(data map[string]interface{}, rules map[string]string) error {
	collector := GetPooledErrorCollector()
	defer PutPooledErrorCollector(collector)

	for field, rule := range rules {
		value, exists := data[field]

		// 检查必填字段
		if strings.Contains(rule, "required") && !exists {
			collector.AddError(field, "required")
			continue
		}

		if !exists {
			continue
		}

		// 验证字段
		if err := v.validate.Var(value, rule); err != nil {
			if errs, ok := err.(validator.ValidationErrors); ok {
				for _, e := range errs {
					collector.AddFieldError(field, e.Tag(), e.Param(), "")
				}
			}
		}
	}

	if collector.HasErrors() {
		return collector.GetErrors()
	}

	return nil
}

// ============================================================================
// 验证器选项 - 函数选项模式
// ============================================================================

// ValidatorOption 验证器选项函数
type ValidatorOption func(interface{})

// WithValidatorCache 设置缓存
func WithValidatorCache(cache CacheManager) ValidatorOption {
	return func(v interface{}) {
		switch val := v.(type) {
		case *defaultValidator:
			val.cache = cache
			val.useCache = true
		case *MapValidator:
			val.cache = cache
		}
	}
}

// WithValidatorPool 设置对象池
func WithValidatorPool(pool ValidatorPool) ValidatorOption {
	return func(v interface{}) {
		if val, ok := v.(*defaultValidator); ok {
			val.pool = pool
			val.usePool = true
		}
	}
}

// WithValidatorStrategy 设置验证策略
func WithValidatorStrategy(strategy ValidationStrategy) ValidatorOption {
	return func(v interface{}) {
		if val, ok := v.(*defaultValidator); ok {
			val.strategy = strategy
		}
	}
}

// WithErrorFormatter 设置错误格式化器
func WithErrorFormatter(formatter ErrorFormatter) ValidatorOption {
	return func(v interface{}) {
		switch val := v.(type) {
		case *defaultValidator:
			val.errorFormatter = formatter
		case *MapValidator:
			val.errorFormatter = formatter
		}
	}
}
