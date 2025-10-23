package v2

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// Map 验证器实现 - 单一职责：专门处理 Map 类型数据验证
// ============================================================================

// defaultMapValidator 默认的 Map 验证器实现
type defaultMapValidator struct {
	validate       *validator.Validate
	cache          CacheManager
	errorFormatter ErrorFormatter
}

// NewMapValidator 创建 Map 验证器
// 工厂方法模式：封装对象创建逻辑
func NewMapValidator(opts ...ValidatorOption) *defaultMapValidator {
	mv := &defaultMapValidator{
		validate: validator.New(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(mv)
	}

	return mv
}

// ValidateMap 验证 Map 数据
// 实现 MapValidator 接口
func (v *defaultMapValidator) ValidateMap(data map[string]interface{}, rules map[string]string) error {
	if data == nil {
		return nil
	}

	if len(rules) == 0 {
		return nil
	}

	collector := GetPooledErrorCollector()
	defer PutPooledErrorCollector(collector)

	for field, rule := range rules {
		value, exists := data[field]

		// 检查必填字段
		if strings.Contains(rule, "required") && !exists {
			collector.AddError(field, "required")
			continue
		}

		// 如果字段不存在且不是必填，跳过
		if !exists {
			continue
		}

		// 验证字段值
		if err := v.validate.Var(value, rule); err != nil {
			if errs, ok := err.(validator.ValidationErrors); ok {
				for _, e := range errs {
					collector.AddFieldError(field, e.Tag(), e.Param(), "")
				}
			} else {
				collector.AddError(field, err.Error())
			}
		}
	}

	if collector.HasErrors() {
		return collector.GetErrors()
	}

	return nil
}

// ValidateMapWithScene 场景化的 Map 验证
// 支持根据不同场景应用不同的验证规则
func (v *defaultMapValidator) ValidateMapWithScene(data map[string]interface{}, scene Scene, validators *MapValidators) error {
	if data == nil || validators == nil {
		return nil
	}

	// 查找匹配的验证规则
	var matchedRules []MapValidationRule
	for configScene, rule := range validators.Validators {
		if scene.Has(configScene) {
			matchedRules = append(matchedRules, rule)
		}
	}

	if len(matchedRules) == 0 {
		return nil
	}

	collector := GetPooledErrorCollector()
	defer PutPooledErrorCollector(collector)

	// 应用所有匹配的验证规则
	for _, rule := range matchedRules {
		v.validateMapRule(data, rule, collector)
	}

	if collector.HasErrors() {
		return collector.GetErrors()
	}

	return nil
}

// validateMapRule 验证单个 Map 规则
// 内部方法：封装验证逻辑，提高代码复用性
func (v *defaultMapValidator) validateMapRule(data map[string]interface{}, rule MapValidationRule, collector ErrorCollector) {
	// 安全检查：防止 DoS 攻击
	const maxMapSize = 10000
	if len(data) > maxMapSize {
		collector.AddError("map", fmt.Sprintf("size exceeds maximum limit %d", maxMapSize))
		return
	}

	// 1. 验证必填键
	v.checkRequiredKeys(data, rule, collector)

	// 2. 验证允许的键（白名单）
	v.checkAllowedKeys(data, rule, collector)

	// 3. 应用字段验证规则
	v.applyFieldRules(data, rule, collector)

	// 4. 执行自定义键验证器
	v.applyCustomKeyValidators(data, rule, collector)
}

// checkRequiredKeys 检查必填键
func (v *defaultMapValidator) checkRequiredKeys(data map[string]interface{}, rule MapValidationRule, collector ErrorCollector) {
	for _, key := range rule.RequiredKeys {
		if _, exists := data[key]; !exists {
			fieldPath := v.buildFieldPath(rule.ParentNameSpace, key)
			collector.AddFieldError(fieldPath, "required", "", "")
		}
	}
}

// checkAllowedKeys 检查允许的键（白名单验证）
func (v *defaultMapValidator) checkAllowedKeys(data map[string]interface{}, rule MapValidationRule, collector ErrorCollector) {
	if len(rule.AllowedKeys) == 0 {
		return
	}

	// 构建快速查找 map
	allowedMap := make(map[string]bool, len(rule.AllowedKeys))
	for _, key := range rule.AllowedKeys {
		allowedMap[key] = true
	}

	// 检查非法键
	for key := range data {
		if !allowedMap[key] {
			fieldPath := v.buildFieldPath(rule.ParentNameSpace, key)
			collector.AddFieldError(fieldPath, "not_allowed", "", fmt.Sprintf("key '%s' is not allowed", key))
		}
	}
}

// applyFieldRules 应用字段验证规则
func (v *defaultMapValidator) applyFieldRules(data map[string]interface{}, rule MapValidationRule, collector ErrorCollector) {
	for field, ruleStr := range rule.Rules {
		value, exists := data[field]

		// 必填检查
		if strings.Contains(ruleStr, "required") && !exists {
			fieldPath := v.buildFieldPath(rule.ParentNameSpace, field)
			collector.AddFieldError(fieldPath, "required", "", "")
			continue
		}

		// 字段不存在且非必填，跳过
		if !exists {
			continue
		}

		// 执行验证
		if err := v.validate.Var(value, ruleStr); err != nil {
			fieldPath := v.buildFieldPath(rule.ParentNameSpace, field)
			if errs, ok := err.(validator.ValidationErrors); ok {
				for _, e := range errs {
					collector.AddFieldError(fieldPath, e.Tag(), e.Param(), "")
				}
			} else {
				collector.AddError(fieldPath, err.Error())
			}
		}
	}
}

// applyCustomKeyValidators 应用自定义键验证器
func (v *defaultMapValidator) applyCustomKeyValidators(data map[string]interface{}, rule MapValidationRule, collector ErrorCollector) {
	for key, validatorFunc := range rule.KeyValidators {
		value, exists := data[key]
		if !exists {
			continue
		}

		if err := validatorFunc(value); err != nil {
			fieldPath := v.buildFieldPath(rule.ParentNameSpace, key)
			collector.AddError(fieldPath, err.Error())
		}
	}
}

// buildFieldPath 构建字段路径
func (v *defaultMapValidator) buildFieldPath(namespace, field string) string {
	if namespace == "" {
		return field
	}
	return namespace + "." + field
}

// ============================================================================
// Map 验证辅助函数 - 便捷使用
// ============================================================================

// ValidateMap 使用默认 Map 验证器验证
func ValidateMap(data map[string]interface{}, rules map[string]string) error {
	v := NewMapValidator()
	return v.ValidateMap(data, rules)
}

// ValidateMapWithScene 使用默认 Map 验证器进行场景化验证
func ValidateMapWithScene(data map[string]interface{}, scene Scene, validators *MapValidators) error {
	v := NewMapValidator()
	return v.ValidateMapWithScene(data, scene, validators)
}

// ============================================================================
// Map 验证规则构建器 - 流式 API
// ============================================================================

// MapValidationRuleBuilder Map 验证规则构建器
// 建造者模式：简化复杂对象的构建
type MapValidationRuleBuilder struct {
	rule MapValidationRule
}

// NewMapValidationRuleBuilder 创建 Map 验证规则构建器
func NewMapValidationRuleBuilder() *MapValidationRuleBuilder {
	return &MapValidationRuleBuilder{
		rule: MapValidationRule{
			Rules:         make(map[string]string),
			KeyValidators: make(map[string]func(value interface{}) error),
		},
	}
}

// WithParentNameSpace 设置父命名空间
func (b *MapValidationRuleBuilder) WithParentNameSpace(namespace string) *MapValidationRuleBuilder {
	b.rule.ParentNameSpace = namespace
	return b
}

// WithRequiredKeys 设置必填键
func (b *MapValidationRuleBuilder) WithRequiredKeys(keys ...string) *MapValidationRuleBuilder {
	b.rule.RequiredKeys = append(b.rule.RequiredKeys, keys...)
	return b
}

// WithAllowedKeys 设置允许的键
func (b *MapValidationRuleBuilder) WithAllowedKeys(keys ...string) *MapValidationRuleBuilder {
	b.rule.AllowedKeys = append(b.rule.AllowedKeys, keys...)
	return b
}

// AddRule 添加字段验证规则
func (b *MapValidationRuleBuilder) AddRule(field, rule string) *MapValidationRuleBuilder {
	b.rule.Rules[field] = rule
	return b
}

// AddKeyValidator 添加自定义键验证器
func (b *MapValidationRuleBuilder) AddKeyValidator(key string, validator func(value interface{}) error) *MapValidationRuleBuilder {
	b.rule.KeyValidators[key] = validator
	return b
}

// Build 构建 Map 验证规则
func (b *MapValidationRuleBuilder) Build() MapValidationRule {
	return b.rule
}

// ============================================================================
// Map 值验证辅助函数
// ============================================================================

// ValidateMapValue 验证 Map 中单个值
// 便捷函数：快速验证单个字段值
func ValidateMapValue(value interface{}, rule string) error {
	v := validator.New()
	return v.Var(value, rule)
}

// ValidateMapValueWithType 验证 Map 值并检查类型
// 类型安全：在验证前检查值的类型
func ValidateMapValueWithType(value interface{}, expectedType reflect.Kind, rule string) error {
	// 检查类型
	if value == nil {
		if strings.Contains(rule, "required") {
			return fmt.Errorf("value is required but got nil")
		}
		return nil
	}

	actualType := reflect.TypeOf(value).Kind()
	if actualType != expectedType {
		return fmt.Errorf("expected type %s but got %s", expectedType, actualType)
	}

	// 执行验证
	return ValidateMapValue(value, rule)
}

// ============================================================================
// 安全检查辅助函数
// ============================================================================

const (
	maxMapKeyLength = 256
	maxMapSize      = 10000
)

// IsSafeMapKey 检查 Map 键是否安全
// 安全检查：防止恶意键名
func IsSafeMapKey(key string) bool {
	if key == "" || len(key) > maxMapKeyLength {
		return false
	}
	// 防止包含危险字符
	dangerousChars := []string{"\x00", "\n", "\r", "\t"}
	for _, char := range dangerousChars {
		if strings.Contains(key, char) {
			return false
		}
	}
	return true
}

// IsSafeMapSize 检查 Map 大小是否安全
// 安全检查：防止 DoS 攻击
func IsSafeMapSize(size int) bool {
	return size >= 0 && size <= maxMapSize
}

// SanitizeMapKeys 清理 Map 键（移除不安全的键）
// 安全处理：过滤危险键，返回安全的 Map
func SanitizeMapKeys(data map[string]interface{}) map[string]interface{} {
	if !IsSafeMapSize(len(data)) {
		return make(map[string]interface{})
	}

	result := make(map[string]interface{}, len(data))
	for key, value := range data {
		if IsSafeMapKey(key) {
			result[key] = value
		}
	}
	return result
}

// ConvertMapKeysToValidationRules 将 Map 转换为验证规则
// 工具函数：动态生成验证规则
func ConvertMapKeysToValidationRules(data map[string]interface{}, defaultRule string) map[string]string {
	rules := make(map[string]string, len(data))
	for key := range data {
		rules[key] = defaultRule
	}
	return rules
}

// MergeMapValidationRules 合并多个 Map 验证规则
// 工具函数：规则组合和复用
func MergeMapValidationRules(rules ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, rule := range rules {
		for key, value := range rule {
			result[key] = value
		}
	}
	return result
}

// GetMapValueAsString 安全地获取 Map 值为字符串
// 类型转换：安全的类型断言
func GetMapValueAsString(data map[string]interface{}, key string) (string, bool) {
	value, exists := data[key]
	if !exists {
		return "", false
	}
	strValue, ok := value.(string)
	return strValue, ok
}

// GetMapValueAsInt 安全地获取 Map 值为整数
func GetMapValueAsInt(data map[string]interface{}, key string) (int, bool) {
	value, exists := data[key]
	if !exists {
		return 0, false
	}

	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		if intVal, err := strconv.Atoi(v); err == nil {
			return intVal, true
		}
	}
	return 0, false
}

// GetMapValueAsBool 安全地获取 Map 值为布尔值
func GetMapValueAsBool(data map[string]interface{}, key string) (bool, bool) {
	value, exists := data[key]
	if !exists {
		return false, false
	}

	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		if boolVal, err := strconv.ParseBool(v); err == nil {
			return boolVal, true
		}
	}
	return false, false
}
