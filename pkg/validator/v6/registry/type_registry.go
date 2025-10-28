package registry

import (
	"reflect"
	"strings"
	"sync"

	"katydid-common-account/pkg/validator/v6/core"
)

// TypeInfoImpl 类型信息实现
type TypeInfoImpl struct {
	reflectType           reflect.Type
	hasRuleValidation     bool
	hasBusinessValidation bool
	hasLifecycleHooks     bool
	rules                 map[core.Scene]map[string]string
	fieldAccessors        map[string]core.FieldAccessor
}

// HasRuleValidation 是否实现了规则验证
func (t *TypeInfoImpl) HasRuleValidation() bool {
	return t.hasRuleValidation
}

// HasBusinessValidation 是否实现了业务验证
func (t *TypeInfoImpl) HasBusinessValidation() bool {
	return t.hasBusinessValidation
}

// HasLifecycleHooks 是否实现了生命周期钩子
func (t *TypeInfoImpl) HasLifecycleHooks() bool {
	return t.hasLifecycleHooks
}

// GetRules 获取验证规则
func (t *TypeInfoImpl) GetRules() map[core.Scene]map[string]string {
	return t.rules
}

// GetFieldAccessor 获取字段访问器
func (t *TypeInfoImpl) GetFieldAccessor(fieldName string) core.FieldAccessor {
	if accessor, ok := t.fieldAccessors[fieldName]; ok {
		return accessor
	}
	return nil
}

// TypeRegistryImpl 类型注册表实现
// 职责：注册和缓存类型信息
// 设计原则：单一职责、性能优化（缓存）
type TypeRegistryImpl struct {
	cache sync.Map // key: reflect.Type, value: *TypeInfoImpl
}

// NewTypeRegistry 创建类型注册表
func NewTypeRegistry() core.TypeRegistry {
	return &TypeRegistryImpl{}
}

// Register 注册类型
func (r *TypeRegistryImpl) Register(target any) core.TypeInfo {
	if target == nil {
		return nil
	}

	// 获取类型
	typ := reflect.TypeOf(target)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// 尝试从缓存获取
	if cached, ok := r.cache.Load(typ); ok {
		return cached.(core.TypeInfo)
	}

	// 缓存未命中，创建类型信息
	typeInfo := r.buildTypeInfo(target, typ)

	// 存入缓存
	r.cache.Store(typ, typeInfo)

	return typeInfo
}

// Get 获取类型信息
func (r *TypeRegistryImpl) Get(target any) (core.TypeInfo, bool) {
	if target == nil {
		return nil, false
	}

	typ := reflect.TypeOf(target)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	cached, ok := r.cache.Load(typ)
	if !ok {
		return nil, false
	}

	return cached.(core.TypeInfo), true
}

// Clear 清除缓存
func (r *TypeRegistryImpl) Clear() {
	r.cache = sync.Map{}
}

// buildTypeInfo 构建类型信息
func (r *TypeRegistryImpl) buildTypeInfo(target any, typ reflect.Type) *TypeInfoImpl {
	typeInfo := &TypeInfoImpl{
		reflectType:    typ,
		fieldAccessors: make(map[string]core.FieldAccessor),
	}

	// 检查是否实现了接口
	typeInfo.hasRuleValidation = r.implementsRuleProvider(target)
	typeInfo.hasBusinessValidation = r.implementsBusinessValidator(target)
	typeInfo.hasLifecycleHooks = r.implementsLifecycleHook(target)

	// 如果实现了规则验证，获取规则
	if typeInfo.hasRuleValidation {
		if provider, ok := target.(core.RuleProvider); ok {
			typeInfo.rules = provider.GetRules()
			// 构建字段访问器
			typeInfo.fieldAccessors = r.buildFieldAccessors(typ, typeInfo.rules)
		}
	}

	return typeInfo
}

// implementsRuleProvider 检查是否实现了 RuleProvider
func (r *TypeRegistryImpl) implementsRuleProvider(target any) bool {
	_, ok := target.(core.RuleProvider)
	return ok
}

// implementsBusinessValidator 检查是否实现了 BusinessValidator
func (r *TypeRegistryImpl) implementsBusinessValidator(target any) bool {
	_, ok := target.(core.BusinessValidator)
	return ok
}

// implementsLifecycleHook 检查是否实现了 LifecycleHook
func (r *TypeRegistryImpl) implementsLifecycleHook(target any) bool {
	_, ok := target.(core.LifecycleHook)
	return ok
}

// buildFieldAccessors 构建字段访问器
// 优化：使用字段索引访问，时间复杂度从 O(n) 降到 O(1)
func (r *TypeRegistryImpl) buildFieldAccessors(typ reflect.Type, rules map[core.Scene]map[string]string) map[string]core.FieldAccessor {
	accessors := make(map[string]core.FieldAccessor)

	// 遍历所有场景的规则
	for _, sceneRules := range rules {
		for fieldName := range sceneRules {
			// 避免重复构建
			if _, exists := accessors[fieldName]; exists {
				continue
			}

			// 构建访问器
			accessor := r.buildFieldAccessor(typ, fieldName)
			if accessor != nil {
				accessors[fieldName] = accessor
			}
		}
	}

	return accessors
}

// buildFieldAccessor 构建单个字段访问器
func (r *TypeRegistryImpl) buildFieldAccessor(typ reflect.Type, fieldName string) core.FieldAccessor {
	// 首先尝试直接字段名
	if field, ok := typ.FieldByName(fieldName); ok {
		index := field.Index
		return func(target any) (any, bool) {
			val := reflect.ValueOf(target)
			if val.Kind() == reflect.Ptr {
				if val.IsNil() {
					return nil, false
				}
				val = val.Elem()
			}
			if val.Kind() != reflect.Struct {
				return nil, false
			}
			fieldVal := val.FieldByIndex(index)
			if !fieldVal.IsValid() {
				return nil, false
			}
			return fieldVal.Interface(), true
		}
	}

	// 尝试通过 JSON tag 查找
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			tagName := strings.Split(jsonTag, ",")[0]
			if tagName == fieldName {
				index := field.Index
				return func(target any) (any, bool) {
					val := reflect.ValueOf(target)
					if val.Kind() == reflect.Ptr {
						if val.IsNil() {
							return nil, false
						}
						val = val.Elem()
					}
					if val.Kind() != reflect.Struct {
						return nil, false
					}
					fieldVal := val.FieldByIndex(index)
					if !fieldVal.IsValid() {
						return nil, false
					}
					return fieldVal.Interface(), true
				}
			}
		}
	}

	// 未找到
	return nil
}
