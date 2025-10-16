package validator

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// ValidateScene 验证场景
type ValidateScene string

// Validatable 可验证的接口，模型需要实现这个接口来定义验证规则
type Validatable interface {
	// ValidateRules 返回验证规则
	// 返回的 map key 是字段名，value 是验证规则
	ValidateRules() map[ValidateScene]map[string]string
}

// CustomValidatable 自定义验证接口，用于复杂的业务验证逻辑
type CustomValidatable interface {
	// CustomValidate 自定义验证方法
	CustomValidate(scene ValidateScene) error
}

// NestedValidatable 嵌套验证接口，用于验证嵌套的复杂对象（如 Extras）
type NestedValidatable interface {
	// ValidateNested 验证嵌套对象
	ValidateNested(scene ValidateScene) error
}

// ErrorMessageProvider 错误信息提供者接口，允许模型自定义验证错误消息
type ErrorMessageProvider interface {
	// GetErrorMessage 获取字段验证失败的错误信息
	// fieldName: 字段名
	// tag: 验证标签（如 required, email, min 等）
	// param: 验证参数（如 min=3 中的 3）
	GetErrorMessage(fieldName, tag, param string) string
}

// Validator 验证器
type Validator struct {
	validate  *validator.Validate
	typeCache sync.Map // key: reflect.Type, value: *typeCache
	mu        sync.RWMutex
}

// typeCache 类型信息缓存
type typeCache struct {
	isValidatable       bool
	isCustomValidatable bool
	isErrorProvider     bool
}

var (
	defaultValidator *Validator
	once             sync.Once
)

// Default 获取默认验证器实例
func Default() *Validator {
	once.Do(func() {
		defaultValidator = New()
	})
	return defaultValidator
}

// New 创建新的验证器
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
		validate: v,
	}
}

// getOrCacheTypeInfo 获取或缓存类型信息
func (v *Validator) getOrCacheTypeInfo(obj interface{}) *typeCache {
	typ := reflect.TypeOf(obj)

	// 尝试从缓存获取
	if cached, ok := v.typeCache.Load(typ); ok {
		return cached.(*typeCache)
	}

	// 创建新的缓存项
	cache := &typeCache{}
	_, cache.isValidatable = obj.(Validatable)
	_, cache.isCustomValidatable = obj.(CustomValidatable)
	_, cache.isErrorProvider = obj.(ErrorMessageProvider)

	// 存入缓存
	v.typeCache.Store(typ, cache)

	return cache
}

// Validate 验证模型，支持指定场景和嵌套验证
func (v *Validator) Validate(obj interface{}, scene ValidateScene) error {
	if obj == nil {
		return fmt.Errorf("validation object cannot be nil")
	}

	// 获取类型缓存
	cache := v.getOrCacheTypeInfo(obj)

	// 1. 先执行结构体标签验证（如果实现了 Validatable 接口）
	if cache.isValidatable {
		validatable := obj.(Validatable)
		if err := v.validateByRules(obj, validatable.ValidateRules(), scene); err != nil {
			return err
		}
	} else {
		// 如果没有实现 Validatable 接口，使用默认的 validator 验证所有字段
		if err := v.validate.Struct(obj); err != nil {
			return v.formatError(obj, err, cache.isErrorProvider)
		}
	}

	// 2. 递归验证嵌套的结构体字段（包括嵌入的 BaseModel 等）
	if err := v.validateNestedStructs(obj, scene); err != nil {
		return err
	}

	// 3. 执行自定义验证逻辑
	if cache.isCustomValidatable {
		customValidatable := obj.(CustomValidatable)
		if err := customValidatable.CustomValidate(scene); err != nil {
			return err
		}
	}

	// 4. 验证实现了 NestedValidatable 接口的嵌套字段
	if err := v.validateNestedValidatables(obj, scene); err != nil {
		return err
	}

	return nil
}

// validateNestedStructs 递归验证嵌套的结构体
func (v *Validator) validateNestedStructs(obj interface{}, scene ValidateScene) error {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	numField := val.NumField()

	for i := 0; i < numField; i++ {
		field := val.Field(i)

		// 跳过未导出的字段
		if !field.CanInterface() {
			continue
		}

		// 跳过基本类型和指针为 nil 的字段
		if field.Kind() == reflect.Ptr && field.IsNil() {
			continue
		}

		fieldValue := field.Interface()
		fieldType := typ.Field(i)

		// 检查字段是否实现了 Validatable 接口
		if validatable, ok := fieldValue.(Validatable); ok {
			if err := v.validateByRules(fieldValue, validatable.ValidateRules(), scene); err != nil {
				return fmt.Errorf("字段 '%s' 验证失败: %w", fieldType.Name, err)
			}
		}

		// 检查字段是否实现了 CustomValidatable 接口
		if customValidatable, ok := fieldValue.(CustomValidatable); ok {
			if err := customValidatable.CustomValidate(scene); err != nil {
				return fmt.Errorf("字段 '%s' 自定义验证失败: %w", fieldType.Name, err)
			}
		}

		// 如果是结构体或结构体指针，递归验证
		fieldKind := field.Kind()
		if fieldKind == reflect.Ptr && !field.IsNil() {
			fieldKind = field.Elem().Kind()
		}

		if fieldKind == reflect.Struct {
			if err := v.validateNestedStructs(fieldValue, scene); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateNestedValidatables 验证实现了 NestedValidatable 接口的字段
func (v *Validator) validateNestedValidatables(obj interface{}, scene ValidateScene) error {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	numField := val.NumField()

	for i := 0; i < numField; i++ {
		field := val.Field(i)

		// 跳过未导出的字段
		if !field.CanInterface() {
			continue
		}

		// 跳过指针为 nil 的字段
		if field.Kind() == reflect.Ptr && field.IsNil() {
			continue
		}

		fieldValue := field.Interface()

		// 检查是否实现了 NestedValidatable 接口
		if nestedValidatable, ok := fieldValue.(NestedValidatable); ok {
			fieldType := typ.Field(i)
			if err := nestedValidatable.ValidateNested(scene); err != nil {
				return fmt.Errorf("字段 '%s' 嵌套验证失败: %w", fieldType.Name, err)
			}
		}
	}

	return nil
}

// validateByRules 根据规则验证
func (v *Validator) validateByRules(obj interface{}, rules map[ValidateScene]map[string]string, scene ValidateScene) error {
	sceneRules, exists := rules[scene]
	if !exists {
		// 如果场景不存在，不进行验证
		return nil
	}

	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("validation object must be a struct")
	}

	// 逐个字段验证
	for fieldName, rule := range sceneRules {
		if rule == "" {
			continue
		}

		field := val.FieldByName(fieldName)
		if !field.IsValid() {
			return fmt.Errorf("field %s not found in struct", fieldName)
		}

		// 使用 validator 验证字段
		if err := v.validate.Var(field.Interface(), rule); err != nil {
			return v.formatFieldError(obj, fieldName, rule, err)
		}
	}

	return nil
}

// formatError 格式化验证错误
func (v *Validator) formatError(obj interface{}, err error, isErrorProvider bool) error {
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	errCount := len(validationErrors)
	if errCount == 0 {
		return nil
	}

	// 使用 strings.Builder 优化字符串拼接，预分配容量
	var builder strings.Builder
	builder.Grow(errCount * 50) // 预估每个错误约50字节

	for i, e := range validationErrors {
		if i > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString("字段 '")
		builder.WriteString(e.Field())
		builder.WriteString("' 验证失败: ")
		builder.WriteString(v.getErrorMsg(obj, e, isErrorProvider))
	}

	return fmt.Errorf("%s", builder.String())
}

// formatFieldError 格式化字段错误
func (v *Validator) formatFieldError(obj interface{}, fieldName, rule string, err error) error {
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return fmt.Errorf("字段 '%s' 验证失败: %v", fieldName, err)
	}

	// 获取类型缓存以避免重复类型断言
	cache := v.getOrCacheTypeInfo(obj)

	for _, e := range validationErrors {
		return fmt.Errorf("字段 '%s' 验证失败: %s", fieldName, v.getErrorMsg(obj, e, cache.isErrorProvider))
	}

	return fmt.Errorf("字段 '%s' 验证失败", fieldName)
}

// getErrorMsg 获取错误信息，优先使用模型自定义的错误消息
func (v *Validator) getErrorMsg(obj interface{}, e validator.FieldError, isErrorProvider bool) string {
	// 尝试从对象获取自定义错误消息
	if isErrorProvider {
		provider := obj.(ErrorMessageProvider)
		if msg := provider.GetErrorMessage(e.Field(), e.Tag(), e.Param()); msg != "" {
			return msg
		}
	}

	// 使用默认错误消息，优化字符串拼接
	var builder strings.Builder
	builder.Grow(32) // 预分配容量
	builder.WriteString("struct rule error '")
	builder.WriteString(e.Field())
	builder.WriteString(", ")
	builder.WriteString(e.Tag())
	builder.WriteString("'")
	return builder.String()
}

// RegisterValidation 注册自定义验证规则
func (v *Validator) RegisterValidation(tag string, fn validator.Func) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.validate.RegisterValidation(tag, fn)
}

// ValidateStruct 简单的结构体验证（不区分场景）
func (v *Validator) ValidateStruct(obj interface{}) error {
	return v.validate.Struct(obj)
}

// ClearTypeCache 清除类型缓存（用于测试或需要重新加载类型信息时）
func (v *Validator) ClearTypeCache() {
	v.typeCache = sync.Map{}
}

// 便捷函数

// Validate 使用默认验证器验证
func Validate(obj interface{}, scene ValidateScene) error {
	return Default().Validate(obj, scene)
}

// ValidateStruct 使用默认验证器验证结构体
func ValidateStruct(obj interface{}) error {
	return Default().ValidateStruct(obj)
}

// RegisterValidation 注册自定义验证规则到默认验证器
func RegisterValidation(tag string, fn validator.Func) error {
	return Default().RegisterValidation(tag, fn)
}

// ClearTypeCache 清除默认验证器的类型缓存
func ClearTypeCache() {
	Default().ClearTypeCache()
}
