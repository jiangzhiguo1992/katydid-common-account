package registry

import (
	"katydid-common-account/pkg/validator/v5/core"
	"reflect"
)

// TypeInfo 类型信息，缓存类型的验证能力信息
type TypeInfo struct {
	// reflectType 类型
	reflectType reflect.Type
	// isRuleRegistry 是否实现了 IRuleRegistry
	isRuleRegistry bool
	// isRuleValidator 是否实现了 IRuleValidation
	isRuleValidator bool
	// isBusinessValidator 是否实现了 IBusinessValidation
	isBusinessValidator bool
	// isLifecycleHooks 是否实现了 ILifecycleHooks
	isLifecycleHooks bool
	// rules 缓存的规则（如果实现了 IRuleValidation）
	rules map[core.Scene]map[string]string
	// accessors 字段访问器缓存（优化：避免重复的 FieldByName 查找）
	accessors map[string]core.FieldAccessor
}

func (t TypeInfo) IsRuleRegistry() bool {
	return t.isRuleRegistry
}

func (t TypeInfo) IsRuleValidation() bool {
	return t.isRuleValidator
}

func (t TypeInfo) IsBusinessValidation() bool {
	return t.isBusinessValidator
}

func (t TypeInfo) IsLifecycleHooks() bool {
	return t.isLifecycleHooks
}

func (t TypeInfo) Rules() map[core.Scene]map[string]string {
	return t.rules
}

func (t TypeInfo) FieldAccessor(fieldName string) core.FieldAccessor {
	if accessor, ok := t.accessors[fieldName]; ok {
		return accessor
	}
	return nil
}
