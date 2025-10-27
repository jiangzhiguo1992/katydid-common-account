package v5

import (
	"reflect"
	"sync"

	"github.com/go-playground/validator/v10"
)

// TypeInfo 类型信息，缓存类型的验证能力信息
type TypeInfo struct {
	// IsRuleValidator 是否实现了 RuleValidation
	IsRuleValidator bool
	// IsBusinessValidator 是否实现了 BusinessValidation
	IsBusinessValidator bool
	// IsLifecycleHooks 是否实现了 LifecycleHooks
	IsLifecycleHooks bool
	// Rules 缓存的规则（如果实现了 RuleValidation）
	Rules map[Scene]map[string]string
}

// Registry 类型注册表接口
type Registry interface {
	// Register 注册类型信息
	Register(target any) *TypeInfo
	// Get 获取类型信息
	Get(target any) (*TypeInfo, bool)
	// Clear 清除缓存
	Clear()
	// Stats 获取统计信息
	Stats() (count int)
}

// TypeRegistry 默认类型注册表实现
type TypeRegistry struct {
	validator *validator.Validate
	cache     sync.Map // key: reflect.Type, value: *TypeInfo
	mu        sync.RWMutex
}

// NewTypeRegistry 创建默认类型注册表
func NewTypeRegistry(validator *validator.Validate) *TypeRegistry {
	return &TypeRegistry{
		validator: validator,
		cache:     sync.Map{},
	}
}

// Register 注册类型信息
func (r *TypeRegistry) Register(target any) *TypeInfo {
	if target == nil {
		return &TypeInfo{}
	}

	typ := reflect.TypeOf(target)
	if typ == nil {
		return &TypeInfo{}
	}

	// 尝试从缓存获取（热路径）
	if cached, ok := r.cache.Load(typ); ok {
		return cached.(*TypeInfo)
	}

	// 缓存未命中，创建新的缓存项（冷路径）
	info := &TypeInfo{}

	// 检查接口实现
	var ruleProvider RuleValidation
	if ruleProvider, info.IsRuleValidator = target.(RuleValidation); info.IsRuleValidator {
		// 预加载常用场景的规则，不用深拷贝验证规则，外部不会修改影响缓存
		info.Rules = ruleProvider.ValidateRules()
	}
	if _, info.IsBusinessValidator = target.(BusinessValidation); info.IsBusinessValidator {
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
	_, info.IsLifecycleHooks = target.(LifecycleHooks)

	// 存入缓存（使用 LoadOrStore 避免并发时的重复存储）
	actual, _ := r.cache.LoadOrStore(typ, info)
	return actual.(*TypeInfo)
}

// Get 获取类型信息
func (r *TypeRegistry) Get(target any) (*TypeInfo, bool) {
	if target == nil {
		return nil, false
	}

	typ := reflect.TypeOf(target)
	if typ == nil {
		return nil, false
	}

	if cached, ok := r.cache.Load(typ); ok {
		return cached.(*TypeInfo), true
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
