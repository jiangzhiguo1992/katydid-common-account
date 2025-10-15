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

const (
	SceneCreate ValidateScene = "create" // 创建场景
	SceneUpdate ValidateScene = "update" // 更新场景
	SceneDelete ValidateScene = "delete" // 删除场景
	SceneQuery  ValidateScene = "query"  // 查询场景
)

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

// Validator 验证器
type Validator struct {
	validate *validator.Validate
	mu       sync.RWMutex
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
		if name == "-" {
			return ""
		}
		if name == "" {
			return fld.Name
		}
		return name
	})

	return &Validator{
		validate: v,
	}
}

// Validate 验证模型，支持指定场景和嵌套验证
func (v *Validator) Validate(obj interface{}, scene ValidateScene) error {
	if obj == nil {
		return fmt.Errorf("validation object cannot be nil")
	}

	// 1. 先执行结构体标签验证（如果实现了 Validatable 接口）
	if validatable, ok := obj.(Validatable); ok {
		if err := v.validateByRules(obj, validatable.ValidateRules(), scene); err != nil {
			return err
		}
	} else {
		// 如果没有实现 Validatable 接口，使用默认的 validator 验证所有字段
		if err := v.validate.Struct(obj); err != nil {
			return v.formatError(err)
		}
	}

	// 2. 递归验证嵌套的结构体字段（包括嵌入的 BaseModel 等）
	if err := v.validateNestedStructs(obj, scene); err != nil {
		return err
	}

	// 3. 执行自定义验证逻辑
	if customValidatable, ok := obj.(CustomValidatable); ok {
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
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过未导出的字段
		if !field.CanInterface() {
			continue
		}

		// 跳过基本类型和指针为 nil 的字段
		if field.Kind() == reflect.Ptr && field.IsNil() {
			continue
		}

		fieldValue := field.Interface()

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
		if fieldKind == reflect.Ptr {
			if !field.IsNil() {
				fieldKind = field.Elem().Kind()
			}
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
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

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
			return v.formatFieldError(fieldName, rule, err)
		}
	}

	return nil
}

// formatError 格式化验证错误
func (v *Validator) formatError(err error) error {
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	var errMsgs []string
	for _, e := range validationErrors {
		errMsgs = append(errMsgs, fmt.Sprintf("字段 '%s' 验证失败: %s", e.Field(), v.getErrorMsg(e)))
	}

	return fmt.Errorf("%s", strings.Join(errMsgs, "; "))
}

// formatFieldError 格式化字段错误
func (v *Validator) formatFieldError(fieldName, rule string, err error) error {
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return fmt.Errorf("字段 '%s' 验证失败: %v", fieldName, err)
	}

	for _, e := range validationErrors {
		return fmt.Errorf("字段 '%s' 验证失败: %s", fieldName, v.getErrorMsg(e))
	}

	return fmt.Errorf("字段 '%s' 验证失败", fieldName)
}

// getErrorMsg 获取错误信息
func (v *Validator) getErrorMsg(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "必填项"
	case "email":
		return "必须是有效的邮箱地址"
	case "min":
		return fmt.Sprintf("最小值/长度为 %s", e.Param())
	case "max":
		return fmt.Sprintf("最大值/长度为 %s", e.Param())
	case "len":
		return fmt.Sprintf("长度必须为 %s", e.Param())
	case "gt":
		return fmt.Sprintf("必须大于 %s", e.Param())
	case "gte":
		return fmt.Sprintf("必须大于等于 %s", e.Param())
	case "lt":
		return fmt.Sprintf("必须小于 %s", e.Param())
	case "lte":
		return fmt.Sprintf("必须小于等于 %s", e.Param())
	case "alphanum":
		return "只能包含字母和数字"
	case "alpha":
		return "只能包含字母"
	case "numeric":
		return "只能包含数字"
	case "url":
		return "必须是有效的URL"
	case "oneof":
		return fmt.Sprintf("必须是以下值之一: %s", e.Param())
	default:
		return fmt.Sprintf("不符合规则 '%s'", e.Tag())
	}
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
