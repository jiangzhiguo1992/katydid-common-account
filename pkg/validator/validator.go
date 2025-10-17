package validator

import (
	"errors"
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
	CustomValidate(scene ValidateScene) []*FieldError
}

// NestedValidatable 嵌套验证接口，用于验证嵌套的复杂对象（如 Extras、自定义类型等）
// 此接口用于处理包含复杂嵌套结构的字段验证
type NestedValidatable interface {
	// ValidateNested 验证嵌套对象
	// scene: 验证场景
	// 返回 error 表示验证失败，返回 nil 表示验证成功
	ValidateNested(scene ValidateScene) []*FieldError
}

// StructLevelValidatable 结构体级别验证接口（自动注册）
// 实现此接口的类型会在首次验证时自动注册到验证器
// 用于跨字段验证、复杂的业务逻辑验证
// 注意：不再暴露第三方库类型，使用封装的 StructLevel 接口
type StructLevelValidatable interface {
	// StructLevelValidation 结构体级别验证
	// sl: StructLevel 提供的验证上下文（已封装）
	// 可以通过 sl.ReportError() 报告验证错误
	StructLevelValidation(sl StructLevel)
}

// MapRulesValidatable Map 规则验证接口（自动注册）
// 实现此接口的类型会在首次验证时自动注册到验证器
// 用于简单的字段验证规则定义
type MapRulesValidatable interface {
	// ValidationMapRules 返回字段验证规则的 map
	// key: 字段名, value: 验证规则（遵循 go-playground/validator 标签格式）
	ValidationMapRules() map[string]string
}

// Validator 验证器，提供结构体字段验证功能
// 支持场景化验证、嵌套验证、自定义验证等多种验证方式
// 线程安全，可在多个 goroutine 中并发使用
type Validator struct {
	validate          *validator.Validate // 底层验证器实例
	typeCache         *sync.Map           // 类型信息缓存，key: reflect.Type, value: *typeCache
	registeredTags    *sync.Map           // 已注册的验证标签缓存，key: string(tag), value: bool
	registeredStructs *sync.Map           // 已注册的结构体验证缓存，key: reflect.Type, value: bool
	autoRegistered    *sync.Map           // 自动注册的类型缓存，key: reflect.Type, value: bool
	mu                sync.RWMutex        // 保护注册自定义验证函数的互斥锁
}

// typeCache 类型信息缓存结构，用于避免重复的类型断言
// 缓存类型实现的接口信息，提升性能
type typeCache struct {
	isValidatable   bool                                // 是否实现了 Validatable 接口
	validationRules map[ValidateScene]map[string]string // 缓存的验证规则

	isCustomValidatable      bool // 是否实现了 CustomValidatable 接口
	isNestedValidatable      bool // 是否实现了 NestedValidatable 接口
	isStructLevelValidatable bool // 是否实现了 StructLevelValidatable 接口（自动注册）
	isMapRulesValidatable    bool // 是否实现了 MapRulesValidatable 接口（自动注册）
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

// Validate 使用默认验证器验证
func Validate(obj any, scene ValidateScene) []*FieldError {
	return Default().Validate(obj, scene)
}

// ClearTypeCache 清除默认验证器的类型缓存
func ClearTypeCache() {
	Default().ClearTypeCache()
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
		validate:          v,
		typeCache:         &sync.Map{},
		registeredTags:    &sync.Map{},
		registeredStructs: &sync.Map{},
		autoRegistered:    &sync.Map{},
	}
}

// getOrCacheTypeInfo 获取或缓存类型信息
// 通过缓存避免重复的类型断言，提升性能
// 参数 obj: 待验证的对象
// 返回该对象类型的缓存信息
func (v *Validator) getOrCacheTypeInfo(obj any) *typeCache {
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

	// 检查接口实现
	if validatable, ok := obj.(Validatable); ok {
		cache.isValidatable = true
		cache.validationRules = validatable.ValidateRules()
	}
	_, cache.isCustomValidatable = obj.(CustomValidatable)
	_, cache.isNestedValidatable = obj.(NestedValidatable)
	_, cache.isStructLevelValidatable = obj.(StructLevelValidatable)
	_, cache.isMapRulesValidatable = obj.(MapRulesValidatable)

	// 存入缓存（使用 LoadOrStore 避免并发时的重复存储）
	actual, _ := v.typeCache.LoadOrStore(typ, cache)
	return actual.(*typeCache)
}

// Validate 验证模型，支持指定场景和嵌套验证
// 验证流程：
// 1. 自动注册实现了自动注册接口的类型
// 2. 执行结构体标签验证（基于 Validatable 接口的规则）
// 3. 递归验证嵌套的结构体字段（包括嵌入字段）
// 4. 执行自定义验证逻辑（CustomValidatable 接口）
// 5. 验证实现了 NestedValidatable 接口的嵌套字段
// 参数：
//
//	obj: 待验证的对象
//	scene: 验证场景
//
// 返回：验证错误，nil 表示验证成功
func (v *Validator) Validate(obj any, scene ValidateScene) []*FieldError {
	// 安全检查：防止 nil 对象
	if obj == nil {
		return []*FieldError{NewFieldError("struct", "required", nil, nil)}
	}

	// 获取类型缓存
	cache := v.getOrCacheTypeInfo(obj)

	// 0. 自动注册实现了自动注册接口的类型（懒加载，只注册一次）
	v.autoRegisterIfNeeded(obj, cache)

	// 创建验证上下文
	ctx := NewValidationContext(scene)

	// 1. 执行结构体标签验证
	if cache.isValidatable {
		if cache.validationRules != nil {
			v.collectValidationErrors(obj, cache.validationRules, ctx)
		}
	} else {
		if err := v.validate.Struct(obj); err != nil {
			v.addFieldErrors(obj, err, ctx)
		}
	}

	// 2. 递归验证嵌套的结构体字段
	v.collectNestedStructErrors(obj, ctx, 0)

	// 3. 执行自定义验证逻辑
	if cache.isCustomValidatable {
		customValidatable := obj.(CustomValidatable)
		if errs := customValidatable.CustomValidate(scene); errs != nil {
			// 如果自定义验证返回 ValidationError，合并错误
			ctx.AddErrors(errs)
		}
	}

	// 4. 验证实现了 NestedValidatable 接口的嵌套字段
	v.collectNestedValidatableErrors(obj, scene, ctx, 0)

	// 如果有错误，返回验证错误
	if ctx.HasErrors() {
		return ctx.Errors
	} else if len(ctx.Message) != 0 {
		return []*FieldError{NewFieldError("", ctx.Message, nil, nil)}
	}

	return nil
}

// autoRegisterIfNeeded 自动注册实现了自动注册接口的类型
// 这是懒加载机制，只在首次验证时注册一次
func (v *Validator) autoRegisterIfNeeded(obj any, cache *typeCache) {
	typ := reflect.TypeOf(obj)
	if typ == nil {
		return
	}

	// 检查是否已经自动注册过
	if _, registered := v.autoRegistered.Load(typ); registered {
		return
	}

	// 标记为已检查（即使不需要注册，也避免重复检查）
	v.autoRegistered.Store(typ, true)

	// 自动注册 StructLevelValidatable
	if cache.isStructLevelValidatable {
		structLevelValidatable := obj.(StructLevelValidatable)
		v.mu.Lock()
		if _, loaded := v.registeredStructs.LoadOrStore(typ, true); !loaded {
			// 包装调用对象的 StructLevelValidation 方法，使用封装的 StructLevel
			v.validate.RegisterStructValidation(func(sl validator.StructLevel) {
				// 通过类型断言获取当前验证的对象
				if current, ok := sl.Current().Interface().(StructLevelValidatable); ok {
					// 包装第三方库的 StructLevel，隐藏实现细节
					wrapper := &structLevelWrapper{sl: sl}
					current.StructLevelValidation(wrapper)
				}
			}, obj)
		}
		v.mu.Unlock()
		_ = structLevelValidatable // 避免 unused 警告
	}

	// 自动注册 MapRulesValidatable
	if cache.isMapRulesValidatable {
		mapRulesValidatable := obj.(MapRulesValidatable)
		rules := mapRulesValidatable.ValidationMapRules()
		if rules != nil && len(rules) > 0 {
			v.mu.Lock()
			if _, loaded := v.registeredStructs.LoadOrStore(typ, true); !loaded {
				v.validate.RegisterStructValidationMapRules(rules, obj)
			}
			v.mu.Unlock()
		}
	}
}

// collectValidationErrors 收集验证错误（不中断）
func (v *Validator) collectValidationErrors(obj any, rules map[ValidateScene]map[string]string, ctx *ValidationContext) {
	if rules == nil || ctx == nil {
		return
	}

	sceneRules, exists := rules[ctx.Scene]
	if !exists || sceneRules == nil {
		return
	}

	val := reflect.ValueOf(obj)
	if !val.IsValid() {
		return
	}

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return
	}

	// 验证所有字段，收集所有错误
	for fieldName, rule := range sceneRules {
		if rule == "" {
			continue
		}

		field := val.FieldByName(fieldName)
		if !field.IsValid() {
			typ := val.Type()
			field = v.findFieldByJSONTag(val, typ, fieldName)
		}

		if !field.IsValid() || !field.CanInterface() {
			continue
		}

		// 验证字段
		if err := v.validate.Var(field.Interface(), rule); err != nil {
			v.addFieldErrors(obj, err, ctx)
		}
	}
}

// collectNestedStructErrors 收集嵌套结构体错误
func (v *Validator) collectNestedStructErrors(obj any, ctx *ValidationContext, depth int) {
	if depth > maxNestedDepth {
		ctx.AddErrorByDetail("", fmt.Sprintf("nested depth exceeds maximum limit %d", maxNestedDepth), nil, nil)
		return
	}

	val := reflect.ValueOf(obj)
	if !val.IsValid() {
		return
	}

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
	numField := val.NumField()

	for i := 0; i < numField; i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if !field.CanInterface() || !field.IsValid() {
			continue
		}

		if field.Kind() == reflect.Ptr && field.IsNil() {
			continue
		}

		fieldValue := field.Interface()

		// 处理嵌入字段
		if fieldType.Anonymous {
			v.collectNestedStructErrors(fieldValue, ctx, depth+1)
		}

		fieldCache := v.getOrCacheTypeInfo(fieldValue)

		// 验证实现了接口的字段
		if fieldCache.isValidatable && fieldCache.validationRules != nil {
			v.collectValidationErrors(fieldValue, fieldCache.validationRules, ctx)
		}

		if fieldCache.isCustomValidatable {
			customValidatable := fieldValue.(CustomValidatable)
			if errs := customValidatable.CustomValidate(ctx.Scene); errs != nil {
				ctx.AddErrors(errs)
			}
		}

		// 递归处理嵌套结构体
		fieldKind := field.Kind()
		if fieldKind == reflect.Ptr && !field.IsNil() {
			fieldKind = field.Elem().Kind()
		}

		if fieldKind == reflect.Struct && !fieldType.Anonymous {
			v.collectNestedStructErrors(fieldValue, ctx, depth+1)
		}
	}
}

// collectNestedValidatableErrors 收集 NestedValidatable 错误
func (v *Validator) collectNestedValidatableErrors(obj any, scene ValidateScene, ctx *ValidationContext, depth int) {
	if depth > maxNestedDepth {
		return
	}

	val := reflect.ValueOf(obj)
	if !val.IsValid() {
		return
	}

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
	numField := val.NumField()

	for i := 0; i < numField; i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if !field.CanInterface() || !field.IsValid() {
			continue
		}

		if field.Kind() == reflect.Ptr && field.IsNil() {
			continue
		}

		fieldValue := field.Interface()

		if fieldType.Anonymous {
			v.collectNestedValidatableErrors(fieldValue, scene, ctx, depth+1)
		}

		fieldCache := v.getOrCacheTypeInfo(fieldValue)
		if fieldCache.isNestedValidatable {
			nestedValidatable := fieldValue.(NestedValidatable)
			if errs := nestedValidatable.ValidateNested(scene); errs != nil {
				ctx.AddErrors(errs)
			}
		}
	}
}

// addFieldErrors 添加字段验证错误
func (v *Validator) addFieldErrors(_ any, err error, ctx *ValidationContext) {
	var validationErrors validator.ValidationErrors
	ok := errors.As(err, &validationErrors)
	if !ok {
		ctx.AddErrorByDetail("", err.Error(), nil, nil)
		return
	}

	for _, e := range validationErrors {
		ctx.AddErrorByValidator(e)
	}
}

// findFieldByJSONTag 通过 JSON tag 查找字段
// 当字段名不匹配时，尝试通过 json tag 查找对应的字段
func (v *Validator) findFieldByJSONTag(val reflect.Value, typ reflect.Type, jsonTag string) reflect.Value {
	numField := typ.NumField()
	for i := 0; i < numField; i++ {
		fieldType := typ.Field(i)
		tag := strings.SplitN(fieldType.Tag.Get("json"), ",", 2)[0]
		if tag == jsonTag {
			return val.Field(i)
		}
	}
	return reflect.Value{}
}

// ClearTypeCache 清除类型缓存
// 用于测试或需要重新加载类型信息时
// 注意：此方法会清空所有缓存的类型信息，可能影响性能
// 线程安全
func (v *Validator) ClearTypeCache() {
	v.typeCache = &sync.Map{}
}

// GetUnderlyingValidator 获取底层的 go-playground/validator 实例
// 用于需要直接访问底层验证器的高级场景
// 警告：此方法暴露了第三方库的实现细节，仅用于高级场景
func (v *Validator) GetUnderlyingValidator() *validator.Validate {
	return v.validate
}
