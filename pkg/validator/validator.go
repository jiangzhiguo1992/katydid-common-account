package validator

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// ValidateScene 验证场景标识符，使用位运算支持场景组合验证
// 设计目标：
//   - 使用 int64 类型，支持位运算（按位或、按位与）
//   - 允许场景组合：SceneCreate | SceneUpdate 表示同时适用于创建和更新场景
//   - 支持场景匹配：使用 scene & targetScene != 0 判断是否包含目标场景
//
// 使用示例：
//
//	const (
//	    SceneCreate ValidateScene = 1 << 0  // 0b0001 创建场景
//	    SceneUpdate ValidateScene = 1 << 1  // 0b0010 更新场景
//	    SceneDelete ValidateScene = 1 << 2  // 0b0100 删除场景
//	    SceneQuery  ValidateScene = 1 << 3  // 0b1000 查询场景
//	)
//
//	// 场景组合：创建和更新都需要的规则
//	SceneCreateUpdate := SceneCreate | SceneUpdate
//
//	// 场景匹配：判断当前场景是否包含创建场景
//	if scene & SceneCreate != 0 {
//	    // 执行创建场景的验证
//	}
type ValidateScene int64

// 预定义的通用验证场景常量
const (
	SceneNone ValidateScene = 0  // 无场景
	SceneAll  ValidateScene = -1 // 所有场景(111...111)
)

// 验证器配置常量
const (
	// maxNestedDepth 最大嵌套验证深度，防止无限递归导致栈溢出
	maxNestedDepth = 100
)

// ============================================================================
// 核心验证接口
// ============================================================================

// RuleValidator 规则验证器接口 - 定义字段验证规则
// 设计目标：单一职责 - 只负责提供基础的格式验证规则
// 用途：为模型字段提供基础的格式验证规则（必填、长度、格式等）
//
// 使用场景：
//   - 需要场景化验证（创建/更新使用不同规则）
//   - 需要定义字段的基础格式验证（required, min, max, email等）
//
// 示例：
//
//	func (u *User) RuleValidation() map[ValidateScene]map[string]string {
//	    return map[ValidateScene]map[string]string{
//	        "create": {"Username": "required,min=3", "Email": "required,email"},
//	        "update": {"Username": "omitempty,min=3", "Email": "omitempty,email"},
//	    }
//	}
type RuleValidator interface {
	// RuleValidation 返回场景化的验证规则映射
	// 返回格式：map[场景标识][字段名]规则字符串
	// 规则字符串格式遵循 go-playground/validator 的标签语法
	RuleValidation() map[ValidateScene]map[string]string
}

// CustomValidator 自定义验证器接口 - 跨字段验证和复杂业务逻辑验证
// 设计目标：
//   - 单一职责：只负责复杂的业务逻辑验证
//   - 开放封闭：通过接口扩展，无需修改验证器核心代码
//
// 用途：验证多个字段之间的关系和约束，支持复杂业务逻辑验证
//
// 使用场景：
//   - 跨字段验证（如：密码和确认密码必须一致）
//   - 场景化的跨字段验证（如：创建时价格必须小于原价）
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
//	func (p *Product) CustomValidation(scene ValidateScene) []*FieldError {
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
//	    return errors
//	}
type CustomValidator interface {
	// CustomValidation 执行业务验证逻辑
	// 参数 scene：当前验证场景，可根据场景执行不同的验证逻辑
	// 返回：验证错误列表，nil 或空切片表示验证通过
	CustomValidation(scene ValidateScene) []*FieldError
}

// Validator 验证器，提供结构体字段验证功能
// 设计原则：
//   - 单例模式：默认验证器全局唯一，减少资源消耗
//   - 工厂模式：New() 方法创建独立的验证器实例
//   - 策略模式：通过接口支持不同的验证策略
//
// 特性：
//   - 支持场景化验证、嵌套验证、自定义验证等多种验证方式
//   - 类型信息缓存，避免重复的反射操作，提升性能
//   - 懒加载注册，只在首次使用时注册验证函数
type Validator struct {
	// validate 底层验证器实例（go-playground/validator）
	validate *validator.Validate
	// typeCache 类型信息缓存，key: reflect.Type, value: *typeCache
	// 使用 sync.Map 而非 map+mutex，提升并发读性能
	typeCache *sync.Map
	// registeredCache 已注册的类型缓存，key: reflect.Type, value: bool
	// 记录已注册的类型，避免重复注册
	registeredCache *sync.Map
}

// typeCache 类型信息缓存结构，用于避免重复的类型断言和反射操作
// 设计目标：性能优化 - 缓存类型信息，避免重复计算
type typeCache struct {
	// isRuleValidator 是否实现了 RuleValidator 接口
	isRuleValidator bool
	// isCustomValidator 是否实现了 CustomValidator 接口
	isCustomValidator bool
	// validationRules 缓存的验证规则（来自 RuleValidator）
	validationRules map[ValidateScene]map[string]string
}

var (
	// defaultValidator 默认验证器实例，全局单例
	// 使用单例模式减少资源消耗，提升性能
	defaultValidator *Validator
	// once 确保默认验证器只初始化一次（线程安全）
	once sync.Once
)

// Default 获取默认验证器实例（单例模式）
// 线程安全，可在多个 goroutine 中并发调用
// 返回：全局唯一的默认验证器实例
func Default() *Validator {
	once.Do(func() {
		defaultValidator = New()
	})
	return defaultValidator
}

// Validate 使用默认验证器验证对象
// 便捷函数，简化验证调用
// 参数：
//
//	obj: 待验证的对象
//	scene: 验证场景标识
//
// 返回：
//
//	验证错误列表，nil 表示验证通过
func Validate(obj any, scene ValidateScene) []*FieldError {
	return Default().Validate(obj, scene)
}

// ClearTypeCache 清除默认验证器的类型缓存
// 用于测试或需要重新加载类型信息时
// 注意：此方法会影响性能，仅用于特殊场景
func ClearTypeCache() {
	Default().ClearTypeCache()
}

// New 创建新的验证器实例
// 工厂方法模式，返回一个已配置好的验证器
// 适用场景：需要独立的验证器实例（如单元测试、隔离配置）
// 返回：已初始化的验证器实例
func New() *Validator {
	v := validator.New()

	// 注册自定义标签名函数，使用 json tag 作为字段名
	// 这样验证错误消息中会显示 json 字段名而不是结构体字段名
	// 提升 API 响应的友好性
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})

	return &Validator{
		validate:        v,
		typeCache:       &sync.Map{},
		registeredCache: &sync.Map{},
	}
}

// Validate 验证模型，支持指定场景和嵌套验证
// 验证流程（按顺序执行）：
//  1. 自动注册 CustomValidator（懒加载，需要注册到底层验证器）
//  2. 执行结构体标签验证（RuleValidator 直接读取规则，无需注册）
//  3. 递归验证嵌套的结构体字段（包括嵌入字段）
//  4. 执行跨字段验证逻辑（CustomValidator 直接调用）
//
// 错误收集策略：收集所有错误后统一返回，而非遇到第一个错误就停止
// 参数：
//
//	obj: 待验证的对象（必须是结构体或结构体指针）
//	scene: 验证场景标识
//
// 返回：
//
//	验证错误列表，nil 表示验证通过
func (v *Validator) Validate(obj any, scene ValidateScene) []*FieldError {
	// 防御性编程：防止 nil 对象
	if obj == nil {
		return []*FieldError{
			NewFieldError("struct", "struct", "required", "", "").
				WithMessage("validation target cannot be nil"),
		}
	}

	// 性能优化：获取类型缓存，避免重复的接口检查
	cache := v.getOrCacheTypeInfo(obj)

	// 步骤0: 自动注册实现了 CustomValidator 接口的类型（懒加载，只注册一次）
	v.autoRegisterIfNeeded(scene, obj, cache)

	// 创建验证上下文，用于收集所有验证错误
	ctx := NewValidationContext(scene)

	// 步骤1: 执行结构体标签验证（RuleValidator 直接读取规则，无需注册）
	if cache.isRuleValidator {
		if cache.validationRules != nil {
			// 这个不使用使用标准的 validate tag 标签，而是使用 RuleValidator 提供的规则
			v.collectValidationErrors(obj, cache.validationRules, ctx)
		}
	} else {
		// 如果没有实现 RuleValidator，使用底层验证器的 Struct 验证
		// 这支持使用标准的 validate tag 标签
		if err := v.validate.Struct(obj); err != nil {
			v.addFieldErrors(obj, err, ctx)
		}
	}

	// 步骤2: 递归验证嵌套的结构体字段（深度优先遍历）
	v.collectNestedStructErrors(obj, ctx, 0)

	// 步骤3: 执行跨字段验证逻辑（CustomValidator 直接调用）
	if cache.isCustomValidator {
		crossFieldValidator := obj.(CustomValidator)
		if errs := crossFieldValidator.CustomValidation(scene); errs != nil {
			ctx.AddErrors(errs)
		}
	}

	// 返回验证结果
	if ctx.HasErrors() {
		return ctx.Errors
	} else if len(ctx.Message) != 0 {
		// 存在总体错误消息但没有具体字段错误
		return []*FieldError{
			NewFieldError("", "", "", "", "").
				WithMessage(ctx.Message),
		}
	}

	return nil
}

// getOrCacheTypeInfo 获取或缓存类型信息
// 性能优化：通过缓存避免重复的类型断言和反射操作
// 线程安全：使用 sync.Map 的 LoadOrStore 方法避免并发问题
// 参数：
//
//	obj: 待验证的对象
//
// 返回：
//
//	该对象类型的缓存信息
func (v *Validator) getOrCacheTypeInfo(obj any) *typeCache {
	// 防御性编程：防止 nil 对象导致 panic
	if obj == nil {
		return &typeCache{}
	}

	typ := reflect.TypeOf(obj)

	// 防御性编程：防止反射类型为 nil（极少见，但理论上可能）
	if typ == nil {
		return &typeCache{}
	}

	// 性能优化：尝试从缓存获取（热路径）
	if cached, ok := v.typeCache.Load(typ); ok {
		return cached.(*typeCache)
	}

	// 缓存未命中，创建新的缓存项（冷路径）
	cache := &typeCache{}

	// 接口检查：判断对象实现了哪些验证接口
	if ruleValidator, ok := obj.(RuleValidator); ok {
		cache.isRuleValidator = true
		cache.validationRules = ruleValidator.RuleValidation()
	}
	_, cache.isCustomValidator = obj.(CustomValidator)

	// 存入缓存（使用 LoadOrStore 避免并发时的重复存储）
	actual, _ := v.typeCache.LoadOrStore(typ, cache)
	return actual.(*typeCache)
}

// autoRegisterIfNeeded 自动注册实现了 CustomValidator 接口的类型
// 懒加载机制：只在首次验证时注册一次，避免重复注册
// 线程安全：使用 sync.Map 记录已注册的类型
// 参数：
//
//	obj: 待注册的对象
//	cache: 对象的类型缓存信息
func (v *Validator) autoRegisterIfNeeded(scene ValidateScene, obj any, cache *typeCache) {
	// 如果没有实现 CustomValidator 接口，直接返回
	if !cache.isCustomValidator {
		return
	}

	typ := reflect.TypeOf(obj)
	if typ == nil {
		return
	}

	// 检查是否已经自动注册过（热路径：已注册的情况）
	if _, registered := v.registeredCache.Load(typ); registered {
		return
	}

	// 冷路径：首次注册
	v.registeredCache.Store(typ, true)

	// 注册 CustomValidator 接口到底层验证器
	// 注意：这里注册是为了在 Struct 验证时也能执行跨字段验证
	// 但我们主要在步骤3直接调用 CustomValidation 方法
	v.validate.RegisterStructValidation(func(sl validator.StructLevel) {
		// 底层验证器的回调，集成到验证流程中
		if current, ok := sl.Current().Interface().(CustomValidator); ok {
			// 实际的场景化验证在步骤3中进行
			// TODO:GG 这里注册了，哪里验证？
			if errs := current.CustomValidation(scene); errs != nil {
				// 将错误报告给底层验证器
				// TODO:GG 报的错，在哪里返回？
				for _, err := range errs {
					sl.ReportError(err.Value, err.FieldName, err.JsonName, err.Tag, err.Param)
				}
			}
		}
	}, obj)
}

// collectValidationErrors 收集验证错误（不中断验证流程）
// 错误收集策略：收集所有错误后统一返回
// 参数：
//
//	obj: 待验证的对象
//	rules: 验证规则映射
//	ctx: 验证上下文
func (v *Validator) collectValidationErrors(obj any, rules map[ValidateScene]map[string]string, ctx *ValidationContext) {
	// 防御性编程：参数检查
	if rules == nil || ctx == nil {
		return
	}

	// 遍历rules的scene，收集所有符合的的scene
	matchedRules := make(map[string]string)
	for scene, sceneRules := range rules {
		// 场景匹配：使用位运算判断是否包含目标场景
		if scene&ctx.Scene != 0 {
			// 合并规则（后面的规则会覆盖前面的）
			for fieldName, rule := range sceneRules {
				matchedRules[fieldName] = rule
			}
		}
	}

	// 如果没有匹配的规则，直接返回
	if len(matchedRules) == 0 {
		return
	}

	// 获取对象的反射值
	val := reflect.ValueOf(obj)
	if !val.IsValid() {
		return
	}

	// 处理指针类型
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}

	// 只处理结构体类型
	if val.Kind() != reflect.Struct {
		return
	}

	// 验证所有字段，收集所有错误（不中断）
	for fieldName, rule := range matchedRules {
		if rule == "" {
			continue // 跳过空规则
		}

		// 获取字段值
		field := val.FieldByName(fieldName)
		if !field.IsValid() {
			// 字段名可能是 JSON 名称，尝试通过 JSON tag 查找
			typ := val.Type()
			field = v.findFieldByJSONTag(val, typ, fieldName)
		}

		// 字段不存在或不可访问
		if !field.IsValid() || !field.CanInterface() {
			continue
		}

		// 验证字段（使用底层验证器的 Var 方法）
		// 只验证一个变量，不涉及结构体，不需要提前注册
		if err := v.validate.Var(field.Interface(), rule); err != nil {
			v.addFieldErrors(obj, err, ctx)
		}
	}
}

// collectNestedStructErrors 递归收集嵌套结构体的验证错误
// 深度优先遍历：递归验证所有嵌套的结构体字段
// 防止无限递归：使用深度计数器限制递归深度
// 参数：
//
//	obj: 待验证的对象
//	ctx: 验证上下文
//	depth: 当前递归深度
func (v *Validator) collectNestedStructErrors(obj any, ctx *ValidationContext, depth int) {
	// 防止栈溢出：限制最大递归深度
	if depth > maxNestedDepth {
		ctx.AddErrorByDetail(
			"Struct", "Struct", "nest_depth", "", "",
			fmt.Sprintf("nested validation depth exceeds maximum limit %d", maxNestedDepth),
			obj,
		)
		return
	}

	// 获取对象的反射值
	val := reflect.ValueOf(obj)
	if !val.IsValid() {
		return
	}

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return // nil 指针不需要递归验证
		}
		val = val.Elem()
	}

	// 只处理结构体类型
	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()
	numField := val.NumField()

	// 遍历所有字段
	for i := 0; i < numField; i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过不可访问的字段（私有字段）
		if !field.CanInterface() || !field.IsValid() {
			continue
		}

		// 跳过 nil 指针字段
		if field.Kind() == reflect.Ptr && field.IsNil() {
			continue
		}

		fieldValue := field.Interface()

		// 处理嵌入字段（匿名字段）：直接递归
		if fieldType.Anonymous {
			// 对嵌入字段执行完整验证流程
			cache := v.getOrCacheTypeInfo(fieldValue)
			v.autoRegisterIfNeeded(ctx.Scene, fieldValue, cache)
			if cache.isRuleValidator && cache.validationRules != nil {
				v.collectValidationErrors(fieldValue, cache.validationRules, ctx)
			}
			v.collectNestedStructErrors(fieldValue, ctx, depth+1)
			continue
		}

		// 递归处理嵌套结构体
		fieldKind := field.Kind()
		if fieldKind == reflect.Ptr && !field.IsNil() {
			fieldKind = field.Elem().Kind()
		}

		if fieldKind == reflect.Struct {
			// 对嵌套结构体执行完整验证流程
			cache := v.getOrCacheTypeInfo(fieldValue)
			v.autoRegisterIfNeeded(ctx.Scene, fieldValue, cache)
			if cache.isRuleValidator && cache.validationRules != nil {
				v.collectValidationErrors(fieldValue, cache.validationRules, ctx)
			}
			v.collectNestedStructErrors(fieldValue, ctx, depth+1)
		}
	}
}

// addFieldErrors 添加字段验证错误到上下文
// 适配器模式：将底层验证器的错误转换为内部错误类型
// 参数：
//
//	obj: 验证的对象（未使用，保留用于未来扩展）
//	err: 底层验证器产生的错误
//	ctx: 验证上下文
func (v *Validator) addFieldErrors(_ any, err error, ctx *ValidationContext) {
	// 尝试转换为 ValidationErrors 类型
	var validationErrors validator.ValidationErrors
	ok := errors.As(err, &validationErrors)
	if !ok {
		// 不是标准的验证错误，作为普通错误处理
		ctx.AddErrorByDetail("", "", "", "", "", err.Error(), "")
		return
	}

	// 逐个添加字段错误
	for _, e := range validationErrors {
		ctx.AddErrorByValidator(e)
	}
}

// findFieldByJSONTag 通过 JSON tag 查找字段
// 当字段名不匹配时，尝试通过 json tag 查找对应的字段
// 性能优化：线性搜索，适用于字段数量较少的情况
// 参数：
//
//	val: 结构体的反射值
//	typ: 结构体的反射类型
//	jsonTag: JSON 标签名
//
// 返回：
//
//	匹配的字段反射值，如果未找到返回零值
func (v *Validator) findFieldByJSONTag(val reflect.Value, typ reflect.Type, jsonTag string) reflect.Value {
	numField := typ.NumField()
	for i := 0; i < numField; i++ {
		fieldType := typ.Field(i)
		// 提取 json tag 的第一部分（逗号前）
		tag := strings.SplitN(fieldType.Tag.Get("json"), ",", 2)[0]
		if tag == jsonTag {
			return val.Field(i)
		}
	}
	return reflect.Value{} // 未找到，返回零值
}

// ClearTypeCache 清除类型缓存
// 用于测试或需要重新加载类型信息时
// 注意：此方法会清空所有缓存的类型信息，影响性能，仅用于特殊场景
// 线程安全：创建新的 sync.Map 实例
func (v *Validator) ClearTypeCache() {
	v.typeCache = &sync.Map{}
	v.registeredCache = &sync.Map{}
}

// GetUnderlyingValidator 获取底层的 go-playground/validator 实例
// 用于需要直接访问底层验证器的高级场景
// 警告：此方法暴露了第三方库的实现细节，破坏了封装性，仅用于高级场景
// 返回：
//
//	底层验证器实例
func (v *Validator) GetUnderlyingValidator() *validator.Validate {
	return v.validate
}

// TypeCacheStats 获取类型缓存统计信息
// 用于监控和调试，了解缓存使用情况
// 返回：
//
//	缓存的类型数量
func (v *Validator) TypeCacheStats() (typeCacheCount, autoRegisteredCount int) {

	// 统计 typeCache 中的条目数
	v.typeCache.Range(func(key, value interface{}) bool {
		typeCacheCount++
		return true
	})

	// 统计 registeredCache 中的条目数
	v.registeredCache.Range(func(key, value interface{}) bool {
		autoRegisteredCount++
		return true
	})

	return typeCacheCount, autoRegisteredCount
}
