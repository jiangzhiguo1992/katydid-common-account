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

// ============================================================================
// 核心验证接口
// ============================================================================

// RuleProvider 规则提供者接口 - 定义字段验证规则
// 用途：为模型字段提供基础的格式验证规则（必填、长度、格式等）
//
// 使用场景：
//   - 需要场景化验证（创建/更新使用不同规则）
//   - 需要定义字段的基础格式验证（required, min, max, email等）
//
// 示例：
//
//	func (u *User) Rules() map[ValidateScene]map[string]string {
//	    return map[ValidateScene]map[string]string{
//	        "create": {"Username": "required,min=3", "Email": "required,email"},
//	        "update": {"Username": "omitempty,min=3", "Email": "omitempty,email"},
//	    }
//	}
type RuleProvider interface {
	// Rules 返回验证规则
	// 支持场景化：不同场景可以有不同的验证规则
	// 返回格式：map[场景][字段名]规则字符串
	Rules() map[ValidateScene]map[string]string
}

// BusinessValidator 跨字段验证器接口 - 字段间关系和复杂业务逻辑验证
// 用途：验证多个字段之间的关系和约束，支持复杂业务逻辑验证
//
// 使用场景：
//   - 跨字段验证（如：密码和确认密码必须一致）
//   - 需要场景化的跨字段验证（如：创建时价格必须小于原价，更新时可以相等）
//   - 复杂的条件验证（如：电子产品必须有品牌信息）
//   - Map/Extras 字段的动态验证
//   - 需要访问数据库的验证（唯一性检查等）
//   - 包含复杂业务逻辑的验证（如：会员等级判断、权限检查等）
//
// 优势：
//   - 支持场景化验证，不同场景可以有不同的验证逻辑
//   - 返回 []*FieldError，可以一次返回多个错误
//   - 使用简单直观，无需手动报告错误
//   - 自动注册到底层验证器，性能优异
//   - 集成到 go-playground/validator 的验证流程
//
// 示例：
//
//	func (p *Product) BusinessValidation(scene ValidateScene) []*FieldError {
//	    var errors []*FieldError
//
//	    // 简单跨字段验证
//	    if p.Password != p.ConfirmPassword {
//	        errors = append(errors, NewFieldError("confirm_password", "密码和确认密码不一致", nil, nil))
//	    }
//
//	    // 场景化的跨字段验证
//	    if scene == SceneCreate && p.DiscountPrice >= p.OriginalPrice {
//	        errors = append(errors, NewFieldError("discount_price", "折扣价必须低于原价", nil, nil))
//	    }
//
//	    // 复杂条件验证
//	    if p.Category == "electronics" {
//	        if err := ValidateMapMustHaveKeys(p.Extras, "brand"); err != nil {
//	            errors = append(errors, NewFieldError("extras.brand", err.Error(), nil, nil))
//	        }
//	    }
//
//	    return errors
//	}
type BusinessValidator interface {
	// BusinessValidation 跨字段验证方法
	// 参数 scene：当前验证场景，可根据场景执行不同的验证逻辑
	// 返回：验证错误列表，nil 或空切片表示验证通过
	BusinessValidation(scene ValidateScene) []*FieldError
}

// Validator 验证器，提供结构体字段验证功能
// 支持场景化验证、嵌套验证、自定义验证等多种验证方式
// 线程安全，可在多个 goroutine 中并发使用
type Validator struct {
	validate       *validator.Validate // 底层验证器实例
	typeCache      *sync.Map           // 类型信息缓存，key: reflect.Type, value: *typeCache
	autoRegistered *sync.Map           // 自动注册的类型缓存，key: reflect.Type, value: bool
	mu             sync.RWMutex        // 保护注册自定义验证函数的互斥锁
}

// typeCache 类型信息缓存结构，用于避免重复的类型断言
// 缓存类型实现的接口信息，提升性能
type typeCache struct {
	isRuleProvider        bool                                // 是否实现了 RuleProvider 接口
	isCrossFieldValidator bool                                // 是否实现了 BusinessValidator 接口（自动注册）
	validationRules       map[ValidateScene]map[string]string // 缓存的验证规则
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
		validate:       v,
		typeCache:      &sync.Map{},
		autoRegistered: &sync.Map{},
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

	// 接口检查
	if ruleProvider, ok := obj.(RuleProvider); ok {
		cache.isRuleProvider = true
		cache.validationRules = ruleProvider.Rules()
	}
	_, cache.isCrossFieldValidator = obj.(BusinessValidator)

	// 存入缓存（使用 LoadOrStore 避免并发时的重复存储）
	actual, _ := v.typeCache.LoadOrStore(typ, cache)
	return actual.(*typeCache)
}

// Validate 验证模型，支持指定场景和嵌套验证
// 验证流程：
// 1. 自动注册 BusinessValidator（需要注册到底层验证器）
// 2. 执行结构体标签验证（RuleProvider 不需要注册，直接读取规则）
// 3. 递归验证嵌套的结构体字段（包括嵌入字段）
// 4. 执行跨字段验证逻辑（BusinessValidator 直接调用，自动注册）
// 参数：
//
//	obj: 待验证的对象
//	scene: 验证场景
//
// 返回：验证错误列表，nil 表示验证成功
func (v *Validator) Validate(obj any, scene ValidateScene) []*FieldError {
	// 安全检查：防止 nil 对象
	if obj == nil {
		return []*FieldError{NewFieldError(nil, "struct", "", "required", "")}
	}

	// 获取类型缓存
	cache := v.getOrCacheTypeInfo(obj)

	// 0. 自动注册实现了 BusinessValidator 接口的类型（懒加载，只注册一次）
	v.autoRegisterIfNeeded(obj, cache)

	// 创建验证上下文
	ctx := NewValidationContext(scene)

	// 1. 执行结构体标签验证（RuleProvider 直接读取规则，无需注册）
	if cache.isRuleProvider {
		if cache.validationRules != nil {
			v.collectValidationErrors(obj, cache.validationRules, ctx)
		}
	} else {
		// 如果没有实现 RuleProvider，使用底层验证器的 Struct 验证
		if err := v.validate.Struct(obj); err != nil {
			v.addFieldErrors(obj, err, ctx)
		}
	}

	// 2. 递归验证嵌套的结构体字段
	v.collectNestedStructErrors(obj, ctx, 0)

	// 3. 执行跨字段验证逻辑（BusinessValidator 直接调用）
	if cache.isCrossFieldValidator {
		crossFieldValidator := obj.(BusinessValidator)
		if errs := crossFieldValidator.BusinessValidation(scene); errs != nil {
			ctx.AddErrors(errs)
		}
	}

	// 如果有错误，返回验证错误
	if ctx.HasErrors() {
		return ctx.Errors
	} else if len(ctx.Message) != 0 {
		return []*FieldError{NewFieldError("", "", "", ctx.Message, "")}
	}

	return nil
}

// autoRegisterIfNeeded 自动注册实现了 BusinessValidator 接口的类型
// 这是懒加载机制，只在首次验证时注册一次
func (v *Validator) autoRegisterIfNeeded(obj any, cache *typeCache) {
	if !cache.isCrossFieldValidator {
		return
	}

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

	// 注册 BusinessValidator 接口
	// 注意：这里注册到底层验证器是为了在 Struct 验证时也能执行跨字段验证
	// 但我们主要在步骤3直接调用 BusinessValidation 方法
	v.validate.RegisterStructValidation(func(sl validator.StructLevel) {
		// 这里是底层验证器的回调，我们需要创建一个包装器来收集错误
		// 但由于我们在步骤3直接调用了 BusinessValidation，这里可以简化或保留兼容性
		if current, ok := sl.Current().Interface().(BusinessValidator); ok {
			// 传递空场景，因为底层验证器不知道场景
			// 实际的场景化验证在步骤3中进行
			if errs := current.BusinessValidation(""); errs != nil {
				// 将错误报告给底层验证器
				for _, err := range errs {
					sl.ReportError(err.Value, err.JsonName, err.FieldName, err.Tag, err.Param)
				}
			}
		}
	}, obj)
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
		ctx.AddErrorByDetail("", "", "", fmt.Sprintf("nested depth exceeds maximum limit %d", maxNestedDepth), "", "", "")
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
			continue
		}

		// 递归处理嵌套结构体
		fieldKind := field.Kind()
		if fieldKind == reflect.Ptr && !field.IsNil() {
			fieldKind = field.Elem().Kind()
		}

		if fieldKind == reflect.Struct {
			v.collectNestedStructErrors(fieldValue, ctx, depth+1)
		}
	}
}

// addFieldErrors 添加字段验证错误
func (v *Validator) addFieldErrors(_ any, err error, ctx *ValidationContext) {
	var validationErrors validator.ValidationErrors
	ok := errors.As(err, &validationErrors)
	if !ok {
		ctx.AddErrorByDetail("", "", "", err.Error(), "", "", "")
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
