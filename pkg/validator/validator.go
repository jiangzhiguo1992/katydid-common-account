package validator

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// ValidateScene 验证场景，用于区分不同的验证规则集，在外部定义
// 例如：创建场景、更新场景、删除场景等
type ValidateScene string

// 验证器配置常量
const (
	// 默认字符串构建器初始容量
	defaultBuilderCapacity = 128
	// 单个错误消息预估长度
	errorMessageEstimateLen = 50
	// 最大嵌套验证深度，防止无限递归
	maxNestedDepth = 100
)

// Validatable 可验证的接口，模型需要实现这个接口来定义验证规则
// 通过场景化的验证规则，可以针对不同的业务场景使用不同的验证逻辑
type Validatable interface {
	// ValidateRules 返回验证规则
	// 返回的 map 第一层 key 是验证场景，第二层 key 是字段名，value 是验证规则
	// 验证规则遵循 go-playground/validator/v10 的标签格式
	ValidateRules() map[ValidateScene]map[string]string
}

// CustomValidatable 自定义验证接口，用于复杂的业务验证逻辑
// 当标准的验证标签无法满足需求时，可以实现此接口进行自定义验证
type CustomValidatable interface {
	// CustomValidate 自定义验证方法
	// scene: 验证场景，可以根据不同场景执行不同的验证逻辑
	// 返回 error 表示验证失败，返回 nil 表示验证成功
	CustomValidate(scene ValidateScene) error
}

// NestedValidatable 嵌套验证接口，用于验证嵌套的复杂对象（如 Extras、自定义类型等）
// 此接口用于处理包含复杂嵌套结构的字段验证
type NestedValidatable interface {
	// ValidateNested 验证嵌套对象
	// scene: 验证场景
	// 返回 error 表示验证失败，返回 nil 表示验证成功
	ValidateNested(scene ValidateScene) error
}

// ErrorMessageProvider 错误信息提供者接口，允许模型自定义验证错误消息
// 实现此接口可以提供更友好、更具体的错误提示信息
type ErrorMessageProvider interface {
	// GetErrorMessage 获取字段验证失败的错误信息
	// fieldName: 字段名（通常是 json tag 中定义的名称）
	// tag: 验证标签（如 required, email, min 等）
	// param: 验证参数（如 min=3 中的 3）
	// 返回空字符串表示使用默认错误消息
	GetErrorMessage(fieldName, tag, param string) string
}

// Validator 验证器，提供结构体字段验证功能
// 支持场景化验证、嵌套验证、自定义验证等多种验证方式
// 线程安全，可在多个 goroutine 中并发使用
type Validator struct {
	validate  *validator.Validate // 底层验证器实例
	typeCache sync.Map            // 类型信息缓存，key: reflect.Type, value: *typeCache
	mu        sync.RWMutex        // 保护注册自定义验证函数的互斥锁
}

// typeCache 类型信息缓存结构，用于避免重复的类型断言
// 缓存类型实现的接口信息，提升性能
type typeCache struct {
	isValidatable       bool // 是否实现了 Validatable 接口
	isCustomValidatable bool // 是否实现了 CustomValidatable 接口
	isErrorProvider     bool // 是否实现了 ErrorMessageProvider 接口
	isNestedValidatable bool // 是否实现了 NestedValidatable 接口
}

var (
	// defaultValidator 默认验证器实例，全局单例
	defaultValidator *Validator
	// once 确保默认验证器只初始化一次
	once sync.Once
)

// Default 获取默认验证器实例（单例模式）
// 线程安全，可在多个 goroutine 中并发调用
func Default() *Validator {
	once.Do(func() {
		defaultValidator = New()
	})
	return defaultValidator
}

// New 创建新的验证器实例
// 返回一个已配置好的验证器，可独立使用
func New() *Validator {
	v := validator.New()

	// 注册自定义标签名函数，使用 json tag 作为字段名
	// 这样验证错误消息中会显示 json 字段名而不是结构体字段名
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
// 通过缓存避免重复的类型断言，提升性能
// 参数 obj: 待验证的对象
// 返回该对象类型的缓存信息
func (v *Validator) getOrCacheTypeInfo(obj interface{}) *typeCache {
	// 安全检查：防止 nil 对象导致 panic
	if obj == nil {
		return &typeCache{}
	}

	typ := reflect.TypeOf(obj)

	// 安全检查：防止反射类型为 nil
	if typ == nil {
		return &typeCache{}
	}

	// 尝试从缓存获取
	if cached, ok := v.typeCache.Load(typ); ok {
		return cached.(*typeCache)
	}

	// 创建新的缓存项
	cache := &typeCache{}
	_, cache.isValidatable = obj.(Validatable)
	_, cache.isCustomValidatable = obj.(CustomValidatable)
	_, cache.isErrorProvider = obj.(ErrorMessageProvider)
	_, cache.isNestedValidatable = obj.(NestedValidatable)

	// 存入缓存（使用 LoadOrStore 避免并发时的重复存储）
	actual, _ := v.typeCache.LoadOrStore(typ, cache)
	return actual.(*typeCache)
}

// Validate 验证模型，支持指定场景和嵌套验证
// 验证流程：
// 1. 执行结构体标签验证（基于 Validatable 接口的规则）
// 2. 递归验证嵌套的结构体字段
// 3. 执行自定义验证逻辑（CustomValidatable 接口）
// 4. 验证实现了 NestedValidatable 接口的嵌套字段
// 参数：
//
//	obj: 待验证的对象
//	scene: 验证场景
//
// 返回：验证错误，nil 表示验证成功
func (v *Validator) Validate(obj interface{}, scene ValidateScene) error {
	// 安全检查：防止 nil 对象
	if obj == nil {
		return fmt.Errorf("validation failed: object cannot be nil")
	}

	// 获取类型缓存
	cache := v.getOrCacheTypeInfo(obj)

	// 1. 先执行结构体标签验证（如果实现了 Validatable 接口）
	if cache.isValidatable {
		validatable := obj.(Validatable)
		rules := validatable.ValidateRules()
		// 安全检查：防止 ValidateRules 返回 nil
		if rules != nil {
			if err := v.validateByRules(obj, rules, scene); err != nil {
				return err
			}
		}
	} else {
		// 如果没有实现 Validatable 接口，使用默认的 validator 验证所有字段
		if err := v.validate.Struct(obj); err != nil {
			return v.formatError(obj, err, cache.isErrorProvider)
		}
	}

	// 2. 递归验证嵌套的结构体字段（包括嵌入的 BaseModel 等）
	if err := v.validateNestedStructs(obj, scene, 0); err != nil {
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
	if err := v.validateNestedValidatables(obj, scene, 0); err != nil {
		return err
	}

	return nil
}

// validateNestedStructs 递归验证嵌套的结构体
// 遍历结构体的所有字段，对实现了验证接口的字段进行验证
// 参数：
//
//	obj: 待验证的对象
//	scene: 验证场景
//	depth: 当前递归深度，防止无限递归
//
// 返回：验证错误，nil 表示验证成功
func (v *Validator) validateNestedStructs(obj interface{}, scene ValidateScene, depth int) error {
	// 防止无限递归导致栈溢出
	if depth > maxNestedDepth {
		return fmt.Errorf("validation failed: nested depth exceeds maximum limit %d", maxNestedDepth)
	}

	val := reflect.ValueOf(obj)

	// 安全检查：处理无效的反射值
	if !val.IsValid() {
		return nil
	}

	if val.Kind() == reflect.Ptr {
		// 安全检查：跳过 nil 指针
		if val.IsNil() {
			return nil
		}
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

		// 安全检查：防止无效的字段值
		if !field.IsValid() {
			continue
		}

		fieldValue := field.Interface()
		fieldType := typ.Field(i)

		// 检查字段是否实现了 Validatable 接口
		if validatable, ok := fieldValue.(Validatable); ok {
			rules := validatable.ValidateRules()
			// 安全检查：防止 ValidateRules 返回 nil
			if rules != nil {
				if err := v.validateByRules(fieldValue, rules, scene); err != nil {
					return fmt.Errorf("field '%s' validation failed: %w", fieldType.Name, err)
				}
			}
		}

		// 检查字段是否实现了 CustomValidatable 接口
		if customValidatable, ok := fieldValue.(CustomValidatable); ok {
			if err := customValidatable.CustomValidate(scene); err != nil {
				return fmt.Errorf("field '%s' custom validation failed: %w", fieldType.Name, err)
			}
		}

		// 如果是结构体或结构体指针，递归验证
		fieldKind := field.Kind()
		if fieldKind == reflect.Ptr && !field.IsNil() {
			fieldKind = field.Elem().Kind()
		}

		if fieldKind == reflect.Struct {
			// 递归验证，深度加 1
			if err := v.validateNestedStructs(fieldValue, scene, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateNestedValidatables 验证实现了 NestedValidatable 接口的字段
// 处理实现了 NestedValidatable 接口的嵌套对象
// 参数：
//
//	obj: 待验证的对象
//	scene: 验证场景
//	depth: 当前递归深度，防止无限递归
//
// 返回：验证错误，nil 表示验证成功
func (v *Validator) validateNestedValidatables(obj interface{}, scene ValidateScene, depth int) error {
	// 防止无限递归导致栈溢出
	if depth > maxNestedDepth {
		return fmt.Errorf("validation failed: nested validatable depth exceeds maximum limit %d", maxNestedDepth)
	}

	val := reflect.ValueOf(obj)

	// 安全检查：处理无效的反射值
	if !val.IsValid() {
		return nil
	}

	if val.Kind() == reflect.Ptr {
		// 安全检查：跳过 nil 指针
		if val.IsNil() {
			return nil
		}
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

		// 安全检查：防止无效的字段值
		if !field.IsValid() {
			continue
		}

		fieldValue := field.Interface()

		// 检查是否实现了 NestedValidatable 接口
		if nestedValidatable, ok := fieldValue.(NestedValidatable); ok {
			fieldType := typ.Field(i)
			if err := nestedValidatable.ValidateNested(scene); err != nil {
				return fmt.Errorf("field '%s' nested validation failed: %w", fieldType.Name, err)
			}
		}
	}

	return nil
}

// validateByRules 根据规则验证
// 根据指定场景的验证规则验证对象的字段
// 参数：
//
//	obj: 待验证的对象
//	rules: 验证规则集合
//	scene: 验证场景
//
// 返回：验证错误，nil 表示验证成功
func (v *Validator) validateByRules(obj interface{}, rules map[ValidateScene]map[string]string, scene ValidateScene) error {
	// 安全检查：防止 rules 为 nil
	if rules == nil {
		return nil
	}

	sceneRules, exists := rules[scene]
	if !exists {
		// 如果场景不存在，不进行验证
		return nil
	}

	// 安全检查：防止 sceneRules 为 nil
	if sceneRules == nil {
		return nil
	}

	val := reflect.ValueOf(obj)

	// 安全检查：处理无效的反射值
	if !val.IsValid() {
		return fmt.Errorf("validation failed: invalid object")
	}

	if val.Kind() == reflect.Ptr {
		// 安全检查：防止 nil 指针
		if val.IsNil() {
			return fmt.Errorf("validation failed: object pointer is nil")
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("validation failed: object must be a struct, got %s", val.Kind())
	}

	// 逐个字段验证
	for fieldName, rule := range sceneRules {
		if rule == "" {
			continue
		}

		field := val.FieldByName(fieldName)
		if !field.IsValid() {
			return fmt.Errorf("validation failed: field '%s' not found in struct", fieldName)
		}

		// 安全检查：确保字段可以被访问
		if !field.CanInterface() {
			return fmt.Errorf("validation failed: field '%s' cannot be accessed", fieldName)
		}

		// 使用 validator 验证字段
		if err := v.validate.Var(field.Interface(), rule); err != nil {
			return v.formatFieldError(obj, fieldName, rule, err)
		}
	}

	return nil
}

// formatError 格式化验证错误
// 将验证器返回的错误格式化为更友好的错误消息
// 参数：
//
//	obj: 验证对象
//	err: 原始错误
//	isErrorProvider: 是否实现了 ErrorMessageProvider 接口
//
// 返回：格式化后的错误
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
	// 防止内存溢出：限制最大容量
	capacity := errCount * errorMessageEstimateLen
	if capacity > 10000 { // 限制最大容量为 10KB
		capacity = 10000
	}
	var builder strings.Builder
	builder.Grow(capacity)

	for i, e := range validationErrors {
		if i > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString("field '")
		builder.WriteString(e.Field())
		builder.WriteString("' validation failed: ")
		builder.WriteString(v.getErrorMsg(obj, e, isErrorProvider))
	}

	return fmt.Errorf("%s", builder.String())
}

// formatFieldError 格式化字段错误
// 格式化单个字段的验证错误消息
// 参数：
//
//	obj: 验证对象
//	fieldName: 字段名
//	rule: 验证规则
//	err: 原始错误
//
// 返回：格式化后的错误
func (v *Validator) formatFieldError(obj interface{}, fieldName, rule string, err error) error {
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return fmt.Errorf("field '%s' validation failed: %v", fieldName, err)
	}

	// 获取类型缓存以避免重复类型断言
	cache := v.getOrCacheTypeInfo(obj)

	for _, e := range validationErrors {
		return fmt.Errorf("field '%s' validation failed: %s", fieldName, v.getErrorMsg(obj, e, cache.isErrorProvider))
	}

	return fmt.Errorf("field '%s' validation failed", fieldName)
}

// getErrorMsg 获取错误信息，优先使用模型自定义的错误消息
// 参数：
//
//	obj: 验证对象
//	e: 字段错误
//	isErrorProvider: 是否实现了 ErrorMessageProvider 接口
//
// 返回：错误消息字符串
func (v *Validator) getErrorMsg(obj interface{}, e validator.FieldError, isErrorProvider bool) string {
	// 尝试从对象获取自定义错误消息
	if isErrorProvider && obj != nil {
		provider := obj.(ErrorMessageProvider)
		if msg := provider.GetErrorMessage(e.Field(), e.Tag(), e.Param()); msg != "" {
			return msg
		}
	}

	// 使用默认错误消息，优化字符串拼接
	var builder strings.Builder
	builder.Grow(64) // 预分配容量
	builder.WriteString("validation rule error: field='")
	builder.WriteString(e.Field())
	builder.WriteString("', tag='")
	builder.WriteString(e.Tag())
	builder.WriteString("'")

	// 如果有参数，添加参数信息
	if e.Param() != "" {
		builder.WriteString(", param='")
		builder.WriteString(e.Param())
		builder.WriteString("'")
	}

	return builder.String()
}

// RegisterValidation 注册自定义验证规则
// 允许扩展验证器，添加自定义的验证标签
// 参数：
//
//	tag: 验证标签名称
//	fn: 验证函数
//
// 返回：注册错误，nil 表示成功
// 线程安全
func (v *Validator) RegisterValidation(tag string, fn validator.Func) error {
	// 安全检查：防止空标签名
	if tag == "" {
		return fmt.Errorf("validation tag cannot be empty")
	}

	// 安全检查：防止 nil 函数
	if fn == nil {
		return fmt.Errorf("validation function cannot be nil")
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	return v.validate.RegisterValidation(tag, fn)
}

// ValidateStruct 简单的结构体验证（不区分场景）
// 使用 go-playground/validator 的默认验证逻辑
// 参数：
//
//	obj: 待验证的对象
//
// 返回：验证错误，nil 表示验证成功
func (v *Validator) ValidateStruct(obj interface{}) error {
	// 安全检查：防止 nil 对象
	if obj == nil {
		return fmt.Errorf("validation failed: object cannot be nil")
	}

	return v.validate.Struct(obj)
}

// ClearTypeCache 清除类型缓存
// 用于测试或需要重新加载类型信息时
// 注意：此方法会清空所有缓存的类型信息，可能影响性能
// 线程安全
func (v *Validator) ClearTypeCache() {
	v.typeCache = sync.Map{}
}

// 便捷函数

// Validate 使用默认验证器验证
// 参数：
//
//	obj: 待验证的对象
//	scene: 验证场景
//
// 返回：验证错误，nil 表示验证成功
func Validate(obj interface{}, scene ValidateScene) error {
	return Default().Validate(obj, scene)
}

// ValidateStruct 使用默认验证器验证结构体
// 参数：
//
//	obj: 待验证的对象
//
// 返回：验证错误，nil 表示验证成功
func ValidateStruct(obj interface{}) error {
	return Default().ValidateStruct(obj)
}

// RegisterValidation 注册自定义验证规则到默认验证器
// 参数：
//
//	tag: 验证标签名称
//	fn: 验证函数
//
// 返回：注册错误，nil 表示成功
func RegisterValidation(tag string, fn validator.Func) error {
	return Default().RegisterValidation(tag, fn)
}

// ClearTypeCache 清除默认验证器的类型缓存
// 用于测试或需要重新加载类型信息时
func ClearTypeCache() {
	Default().ClearTypeCache()
}
