package registry

import (
	"katydid-common-account/pkg/validator/v5/core"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// TypeRegistry 默认类型注册表实现
type TypeRegistry struct {
	validator *validator.Validate
	cache     sync.Map // key: reflect.reflectType, value: *TypeInfo
}

// NewTypeRegistry 创建默认类型注册表
func NewTypeRegistry(validator *validator.Validate) core.ITypeRegistry {
	return &TypeRegistry{
		validator: validator,
		cache:     sync.Map{},
	}
}

// Register 注册类型信息
func (r *TypeRegistry) Register(target any) core.ITypeInfo {
	if target == nil {
		return nil
	}

	typ := reflect.TypeOf(target)
	if typ == nil {
		return &TypeInfo{}
	} else if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// 尝试从缓存获取（热路径）
	if cached, ok := r.cache.Load(typ); ok {
		return cached.(core.ITypeInfo)
	}

	// 缓存未命中，创建新的缓存项（冷路径）
	info := &TypeInfo{
		reflectType: typ,
		accessors:   make(map[string]core.FieldAccessor),
	}

	// 检查是否实现规则注册接口
	var ruleRegistry core.IRuleRegistry
	if _, info.isRuleRegistry = target.(core.IRuleRegistry); info.isRuleRegistry {
		register := NewRuleRegister(r.validator)
		ruleRegistry.RegisterRules(register)
	}

	// 检查是否实现规则验证接口
	var ruleProvider core.IRuleValidation
	if ruleProvider, info.isRuleValidator = target.(core.IRuleValidation); info.isRuleValidator {
		// 预加载常用场景的规则，不用深拷贝验证规则，外部不会修改影响缓存
		info.rules = ruleProvider.ValidateRules()

		// 优化：为所有规则字段构建访问器缓存
		if typ.Kind() == reflect.Struct {
			// 构建字段访问器
			info.accessors = buildFieldAccessors(typ, info.rules)
		}

		// 弃用，不能分场景注册，且一个target类型只能注册一次，同理下面的RegisterStructValidation
		// 注册到底层验证器，直接注册rules，而不是写在struct里，更灵活
		//r.validator.RegisterStructValidationMapRules(info.rules, target)
	}

	// 检查是否实现业务验证接口
	if _, info.isBusinessValidator = target.(core.IBusinessValidation); info.isBusinessValidator {
		// 注册到底层验证器（用于缓存优化）
		// 注意：这里提供空回调，实际验证在步骤4执行
		// 原因：
		//   1. 避免 scene 被闭包捕获（类型只注册一次，但 scene 每次可能不同）
		//   2. 确保验证逻辑在步骤4统一执行，使用正确的 scene
		//   3. 让底层验证器缓存类型元数据，提升性能
		r.validator.RegisterStructValidation(func(sl validator.StructLevel) {
			// 空回调：仅用于类型注册和缓存优化
			// 实际的 CustomValidation 在步骤4中调用
		}, target)
	}

	// 检查是否实现生命周期钩子
	_, info.isLifecycleHooks = target.(core.ILifecycleHooks)

	// 存入缓存（使用 LoadOrStore 避免并发时的重复存储）
	actual, _ := r.cache.LoadOrStore(typ, info)
	return actual.(core.ITypeInfo)
}

// Get 获取类型信息
func (r *TypeRegistry) Get(target any) (core.ITypeInfo, bool) {
	if target == nil {
		return nil, false
	}

	typ := reflect.TypeOf(target)
	if typ == nil {
		return nil, false
	}

	if cached, ok := r.cache.Load(typ); ok {
		return cached.(core.ITypeInfo), true
	}

	return nil, false
}

// Clear 清除缓存
func (r *TypeRegistry) Clear() {
	r.cache = sync.Map{}
}

// Stats 获取统计信息
func (r *TypeRegistry) Stats() int {
	count := 0
	r.cache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// buildFieldAccessors 为所有规则字段构建访问器
func buildFieldAccessors(t reflect.Type, rules map[core.Scene]map[string]string) map[string]core.FieldAccessor {
	accessors := make(map[string]core.FieldAccessor)

	// 遍历所有场景的规则
	for _, sceneRules := range rules {
		for fieldName := range sceneRules {
			// 避免重复构建
			if _, exists := accessors[fieldName]; !exists {
				accessors[fieldName] = buildFieldAccessor(t, fieldName)
			}
		}
	}

	return accessors
}

// buildFieldAccessor 构建字段访问器
// 优化：使用字段索引访问，时间复杂度从 O(n) 降到 O(1)
func buildFieldAccessor(t reflect.Type, fieldName string) core.FieldAccessor {
	// 首先尝试直接字段名
	if field, ok := t.FieldByName(fieldName); ok {
		index := field.Index
		return func(v reflect.Value) reflect.Value {
			return v.FieldByIndex(index) // O(1) 访问
		}
	}

	// 尝试通过 JSON tag 查找
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			tagName := strings.Split(jsonTag, ",")[0]
			if tagName == fieldName {
				index := field.Index
				return func(v reflect.Value) reflect.Value {
					return v.FieldByIndex(index) // O(1) 访问
				}
			}
		}
	}

	// 未找到，返回空访问器
	return func(v reflect.Value) reflect.Value {
		return reflect.Value{}
	}
}
