package v5

import (
	"reflect"
	"sync"
)

// ============================================================================
// TypeRegistry 实现 - 类型注册表
// ============================================================================

// DefaultTypeRegistry 默认类型注册表实现
// 职责：管理类型信息缓存
// 设计原则：单一职责、线程安全
type DefaultTypeRegistry struct {
	cache sync.Map // key: reflect.Type, value: *TypeInfo
	mu    sync.RWMutex
}

// NewDefaultTypeRegistry 创建默认类型注册表
func NewDefaultTypeRegistry() *DefaultTypeRegistry {
	return &DefaultTypeRegistry{}
}

// Register 注册类型信息
func (r *DefaultTypeRegistry) Register(target any) *TypeInfo {
	if target == nil {
		return &TypeInfo{}
	}

	typ := reflect.TypeOf(target)
	if typ == nil {
		return &TypeInfo{}
	}

	// 尝试从缓存获取
	if cached, ok := r.cache.Load(typ); ok {
		return cached.(*TypeInfo)
	}

	// 创建类型信息
	info := &TypeInfo{}

	// 检查接口实现
	_, info.IsRuleProvider = target.(RuleProvider)
	_, info.IsBusinessValidator = target.(BusinessValidator)
	_, info.IsLifecycleHooks = target.(LifecycleHooks)

	// 如果实现了 RuleProvider，缓存所有场景的规则
	if provider, ok := target.(RuleProvider); ok {
		info.Rules = make(map[Scene]map[string]string)
		// 预加载常用场景的规则
		for _, scene := range []Scene{SceneCreate, SceneUpdate, SceneDelete, SceneQuery} {
			rules := provider.GetRules(scene)
			if len(rules) > 0 {
				info.Rules[scene] = rules
			}
		}
	}

	// 存入缓存
	actual, _ := r.cache.LoadOrStore(typ, info)
	return actual.(*TypeInfo)
}

// Get 获取类型信息
func (r *DefaultTypeRegistry) Get(target any) (*TypeInfo, bool) {
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
func (r *DefaultTypeRegistry) Clear() {
	r.cache = sync.Map{}
}

// Stats 获取统计信息
func (r *DefaultTypeRegistry) Stats() int {
	count := 0
	r.cache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// ============================================================================
// SceneMatcher 实现 - 场景匹配器
// ============================================================================

// DefaultSceneMatcher 默认场景匹配器
// 职责：处理场景匹配逻辑
// 设计原则：单一职责
type DefaultSceneMatcher struct{}

// NewDefaultSceneMatcher 创建默认场景匹配器
func NewDefaultSceneMatcher() *DefaultSceneMatcher {
	return &DefaultSceneMatcher{}
}

// Match 判断场景是否匹配
func (m *DefaultSceneMatcher) Match(current, target Scene) bool {
	if target == SceneAll || current == SceneAll {
		return true
	}
	return current&target != 0
}

// MatchRules 匹配并合并规则
func (m *DefaultSceneMatcher) MatchRules(current Scene, rules map[Scene]map[string]string) map[string]string {
	if rules == nil || len(rules) == 0 {
		return nil
	}

	result := make(map[string]string)

	// 遍历所有规则场景
	for scene, sceneRules := range rules {
		if m.Match(current, scene) {
			// 合并规则（后面的覆盖前面的）
			for field, rule := range sceneRules {
				result[field] = rule
			}
		}
	}

	return result
}

// ============================================================================
// ErrorCollector 实现 - 错误收集器
// ============================================================================

// DefaultErrorCollector 默认错误收集器
// 职责：收集和管理验证错误
// 设计原则：单一职责、线程安全
type DefaultErrorCollector struct {
	errors   []*FieldError
	mu       sync.RWMutex
	maxCount int
}

// NewDefaultErrorCollector 创建默认错误收集器
func NewDefaultErrorCollector() *DefaultErrorCollector {
	return &DefaultErrorCollector{
		errors:   make([]*FieldError, 0, 8),
		maxCount: 1000,
	}
}

// NewErrorCollectorWithLimit 创建带限制的错误收集器
func NewErrorCollectorWithLimit(maxCount int) *DefaultErrorCollector {
	return &DefaultErrorCollector{
		errors:   make([]*FieldError, 0, 8),
		maxCount: maxCount,
	}
}

// AddError 添加错误
func (c *DefaultErrorCollector) AddError(err *FieldError) {
	if err == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否达到上限
	if len(c.errors) >= c.maxCount {
		return
	}

	c.errors = append(c.errors, err)
}

// AddErrors 批量添加错误
func (c *DefaultErrorCollector) AddErrors(errs []*FieldError) {
	if len(errs) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 计算可添加的数量
	remaining := c.maxCount - len(c.errors)
	if remaining <= 0 {
		return
	}

	if len(errs) > remaining {
		errs = errs[:remaining]
	}

	c.errors = append(c.errors, errs...)
}

// GetErrors 获取所有错误
func (c *DefaultErrorCollector) GetErrors() []*FieldError {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 返回副本，避免外部修改
	result := make([]*FieldError, len(c.errors))
	copy(result, c.errors)
	return result
}

// HasErrors 是否有错误
func (c *DefaultErrorCollector) HasErrors() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.errors) > 0
}

// Clear 清除错误
func (c *DefaultErrorCollector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors = c.errors[:0]
}

// ErrorCount 错误数量
func (c *DefaultErrorCollector) ErrorCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.errors)
}

// ============================================================================
// ErrorFormatter 实现 - 错误格式化器
// ============================================================================

// DefaultErrorFormatter 默认错误格式化器
// 职责：格式化错误信息
// 设计原则：单一职责
type DefaultErrorFormatter struct{}

// NewDefaultErrorFormatter 创建默认错误格式化器
func NewDefaultErrorFormatter() *DefaultErrorFormatter {
	return &DefaultErrorFormatter{}
}

// Format 格式化单个错误
func (f *DefaultErrorFormatter) Format(err *FieldError) string {
	if err == nil {
		return ""
	}

	if err.Message != "" {
		return err.Message
	}

	if err.Param != "" {
		return "field '" + err.Field + "' failed validation on tag '" + err.Tag + "' with param '" + err.Param + "'"
	}

	return "field '" + err.Field + "' failed validation on tag '" + err.Tag + "'"
}

// FormatAll 格式化所有错误
func (f *DefaultErrorFormatter) FormatAll(errs []*FieldError) string {
	if len(errs) == 0 {
		return "validation passed"
	}

	if len(errs) == 1 {
		return f.Format(errs[0])
	}

	var result string
	maxDisplay := 10
	displayCount := len(errs)
	if displayCount > maxDisplay {
		displayCount = maxDisplay
	}

	for i := 0; i < displayCount; i++ {
		if i > 0 {
			result += "; "
		}
		result += f.Format(errs[i])
	}

	if len(errs) > maxDisplay {
		result += "; ... and more errors"
	}

	return result
}
