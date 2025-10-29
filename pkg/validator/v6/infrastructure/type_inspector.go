package infrastructure

import (
	"katydid-common-account/pkg/validator/v6/core"
	"reflect"
)

// typeInspector 类型检查器实现
// 设计原则：缓存代理模式
type typeInspector struct {
	cache core.ICacheManager
}

// NewTypeInspector 创建类型检查器
func NewTypeInspector(cache core.ICacheManager) core.ITypeInspector {
	if cache == nil {
		cache = NewSimpleCache()
	}
	return &typeInspector{
		cache: cache,
	}
}

// Inspect 检查类型信息
func (i *typeInspector) Inspect(target any) core.ITypeInfo {
	if target == nil {
		return nil
	}

	// 获取类型
	typ := reflect.TypeOf(target)
	if typ == nil {
		return nil
	}

	// 处理指针类型
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// 只处理结构体
	if typ.Kind() != reflect.Struct {
		return nil
	}

	// 尝试从缓存获取
	if info, ok := i.cache.Get(typ); ok {
		return info.(core.ITypeInfo)
	}

	// 创建类型信息
	info := i.buildTypeInfo(target, typ)

	// 存入缓存
	i.cache.Set(typ, info)

	return info
}

// buildTypeInfo 构建类型信息
func (i *typeInspector) buildTypeInfo(target any, typ reflect.Type) core.ITypeInfo {
	info := &typeInfo{
		typeName:  typ.Name(),
		accessors: make(map[string]core.FieldAccessor),
	}

	// 检查接口实现（懒加载）
	info.isRuleProvider = i.implementsRuleProvider(target)
	info.isBusinessValidator = i.implementsBusinessValidator(target)
	info.isLifecycleHooks = i.implementsLifecycleHooks(target)

	// 如果实现了 IRuleValidator，获取规则
	if info.isRuleProvider {
		if _, ok := target.(core.IRuleValidator); ok {
			// 缓存所有场景的规则
			info.rulesCache = make(map[core.Scene]map[string]string)
			// 这里只是占位，实际规则在调用 ValidateRules 时获取
		}
	}

	// 预编译字段访问器
	i.buildFieldAccessors(typ, info)

	return info
}

// buildFieldAccessors 构建字段访问器
func (i *typeInspector) buildFieldAccessors(typ reflect.Type, info *typeInfo) {
	numField := typ.NumField()
	for idx := 0; idx < numField; idx++ {
		field := typ.Field(idx)

		// 跳过未导出字段
		if !field.IsExported() {
			continue
		}

		// 获取 JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			// 使用 JSON tag 作为字段名
			if commaIdx := len(jsonTag); commaIdx > 0 {
				for j, c := range jsonTag {
					if c == ',' {
						commaIdx = j
						break
					}
				}
				jsonTag = jsonTag[:commaIdx]
			}
		}

		// 创建访问器（闭包捕获索引）
		fieldIndex := idx
		accessor := func(value any) (any, bool) {
			v := reflect.ValueOf(value)
			if v.Kind() == reflect.Ptr {
				if v.IsNil() {
					return nil, false
				}
				v = v.Elem()
			}

			if v.Kind() != reflect.Struct {
				return nil, false
			}

			if fieldIndex >= v.NumField() {
				return nil, false
			}

			fieldValue := v.Field(fieldIndex)
			if !fieldValue.IsValid() || !fieldValue.CanInterface() {
				return nil, false
			}

			return fieldValue.Interface(), true
		}

		// 同时用字段名和 JSON tag 作为 key
		info.accessors[field.Name] = accessor
		if jsonTag != "" && jsonTag != field.Name {
			info.accessors[jsonTag] = accessor
		}
	}
}

// implementsRuleProvider 检查是否实现了 IRuleValidator
func (i *typeInspector) implementsRuleProvider(target any) bool {
	_, ok := target.(core.IRuleValidator)
	return ok
}

// implementsBusinessValidator 检查是否实现了 IBusinessValidator
func (i *typeInspector) implementsBusinessValidator(target any) bool {
	_, ok := target.(core.IBusinessValidator)
	return ok
}

// implementsLifecycleHooks 检查是否实现了 LifecycleHooks
func (i *typeInspector) implementsLifecycleHooks(target any) bool {
	_, ok := target.(core.LifecycleHooks)
	return ok
}

// ClearCache 清除缓存
func (i *typeInspector) ClearCache() {
	i.cache.Clear()
}

// Stats 获取统计信息
func (i *typeInspector) Stats() core.CacheStats {
	return i.cache.Stats()
}

// ============================================================================
// ITypeInfo 实现
// ============================================================================

// typeInfo 类型信息实现
type typeInfo struct {
	typeName            string
	isRuleProvider      bool
	isBusinessValidator bool
	isLifecycleHooks    bool
	rulesCache          map[core.Scene]map[string]string
	accessors           map[string]core.FieldAccessor
}

// IsRuleValidator 实现 ITypeInfo 接口
func (t *typeInfo) IsRuleValidator() bool {
	return t.isRuleProvider
}

// IsBusinessValidator 实现 ITypeInfo 接口
func (t *typeInfo) IsBusinessValidator() bool {
	return t.isBusinessValidator
}

// IsLifecycleHooks 实现 ITypeInfo 接口
func (t *typeInfo) IsLifecycleHooks() bool {
	return t.isLifecycleHooks
}

// ValidateRules 实现 ITypeInfo 接口
func (t *typeInfo) ValidateRules(scene core.Scene) map[string]string {
	// 如果有缓存，直接返回
	if t.rulesCache != nil {
		if rules, ok := t.rulesCache[scene]; ok {
			return rules
		}
	}
	return nil
}

// FieldAccessor 实现 ITypeInfo 接口
func (t *typeInfo) FieldAccessor(fieldName string) core.FieldAccessor {
	return t.accessors[fieldName]
}

// TypeName 实现 ITypeInfo 接口
func (t *typeInfo) TypeName() string {
	return t.typeName
}
