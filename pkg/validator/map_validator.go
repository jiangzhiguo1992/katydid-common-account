package validator

import (
	"fmt"
	"math"
	"strings"
	"sync"
)

// MapValidator Map 验证器，用于验证 map[string]any 类型的扩展字段
// 支持必填键验证、允许键白名单验证、自定义键验证器等功能
// 线程安全，可在多个 goroutine 中并发使用
type MapValidator struct {
	// NameSpace 结构体命名空间
	NameSpace string
	// RequiredKeys 必填的键列表
	RequiredKeys []string
	// AllowedKeys 允许的键列表（如果为空则不限制）
	AllowedKeys []string
	// KeyValidators 特定键的自定义验证函数
	// key: 字段名，value: 验证函数
	KeyValidators map[string]func(value any) error
	// allowedKeysMap 内部缓存的允许键 map（优化查找性能，避免每次遍历切片）
	allowedKeysMap map[string]bool
	// mu 保护 allowedKeysMap 的并发访问，确保线程安全
	mu sync.RWMutex
}

// ValidateMap 验证 map[string]any 类型的扩展字段
// 执行必填键验证、允许键验证、自定义键验证等
// 收集所有错误后统一返回
// 参数：
//
//	kvs: 待验证的 map
//	v: MapValidator 验证器配置
//
// 返回：验证错误，nil 表示验证成功
func ValidateMap(kvs map[string]any, v *MapValidator) []*FieldError {
	// 安全检查：如果验证器为 nil，则不进行验证
	if v == nil {
		return nil
	}

	// 安全检查：防止 kvs 为 nil 导致 panic
	if kvs == nil {
		// 如果有必填键要求，则 nil map 是错误的
		if len(v.RequiredKeys) > 0 {
			return []*FieldError{NewFieldError(nil, "", "map", "required", "")}
		}
		return nil
	}

	// 创建错误收集器 TODO:GG 分场景吗?
	ctx := NewValidationContext("")

	// 1. 验证必填键
	if len(v.RequiredKeys) > 0 {
		v.collectRequiredKeyErrors(kvs, ctx)
	}

	// 2. 验证允许的键
	if len(v.AllowedKeys) > 0 {
		v.collectAllowedKeyErrors(kvs, ctx)
	}

	// 3. 执行自定义键验证器
	if len(v.KeyValidators) > 0 {
		v.collectCustomKeyErrors(kvs, ctx)
	}

	// 如果有错误，返回验证错误
	if ctx.HasErrors() {
		return ctx.Errors
	} else if len(ctx.Message) != 0 {
		return []*FieldError{NewFieldError(nil, "", ctx.Message, "", "")}
	}

	return nil
}

// collectRequiredKeyErrors 收集必填键错误
func (mv *MapValidator) collectRequiredKeyErrors(kvs map[string]any, ctx *ValidationContext) {
	for _, key := range mv.RequiredKeys {
		if _, exists := kvs[key]; !exists {
			ctx.AddErrorByDetail(nil, "", "", "required", "", "", mv.NameSpace+"."+key)
		}
	}
}

// collectAllowedKeyErrors 收集允许键错误
func (mv *MapValidator) collectAllowedKeyErrors(kvs map[string]any, ctx *ValidationContext) {
	mv.mu.RLock()
	allowedMap := mv.allowedKeysMap
	mv.mu.RUnlock()

	if allowedMap == nil {
		mv.mu.Lock()
		if mv.allowedKeysMap == nil {
			mv.allowedKeysMap = make(map[string]bool, len(mv.AllowedKeys))
			for _, key := range mv.AllowedKeys {
				mv.allowedKeysMap[key] = true
			}
		}
		allowedMap = mv.allowedKeysMap
		mv.mu.Unlock()
	}

	for key := range kvs {
		if !allowedMap[key] {
			ctx.AddErrorByDetail(nil, "", "", "allowed", "", "", mv.NameSpace+"."+key)
		}
	}
}

// collectCustomKeyErrors 收集自定义键验证错误
func (mv *MapValidator) collectCustomKeyErrors(kvs map[string]any, ctx *ValidationContext) {
	for key, validatorFunc := range mv.KeyValidators {
		if value, exists := kvs[key]; exists {
			if validatorFunc == nil {
				continue
			}

			if err := validatorFunc(value); err != nil {
				ctx.AddErrorByDetail(nil, "", "", "custom", "", "", mv.NameSpace+"."+key)
			}
		}
	}
}

// ValidateMapKey 验证 map[string]any 中特定键的值
// 如果键不存在，则不进行验证
// 参数：
//
//	kvs: 待验证的 map
//	key: 键名
//	validatorFunc: 验证函数
//
// 返回：验证错误，nil 表示验证成功
func ValidateMapKey(kvs map[string]any, key string, validatorFunc func(value any) error) error {
	// 安全检查：防止 kvs 为 nil
	if kvs == nil {
		return nil
	}

	// 安全检查：防止验证函数为 nil
	if validatorFunc == nil {
		return fmt.Errorf("map key validation failed: validator function cannot be nil")
	}

	value, exists := kvs[key]
	if !exists {
		return nil // 键不存在时不验证
	}

	if err := validatorFunc(value); err != nil {
		return fmt.Errorf("map key '%s' validation failed: %w", key, err)
	}

	return nil
}

// ValidateMapMustHaveKey 验证 map[string]any 必须包含指定的键
// 参数：
//
//	kvs: 待验证的 map
//	key: 必须存在的键名
//
// 返回：验证错误，nil 表示验证成功
func ValidateMapMustHaveKey(kvs map[string]any, key string) error {
	// 安全检查：防止 kvs 为 nil
	if kvs == nil {
		return fmt.Errorf("map validation failed: map cannot be nil")
	}

	if _, exists := kvs[key]; !exists {
		return fmt.Errorf("map validation failed: missing required key '%s'", key)
	}
	return nil
}

// ValidateMapMustHaveKeys 验证 map[string]any 必须包含指定的多个键
// 参数：
//
//	kvs: 待验证的 map
//	keys: 必须存在的键名列表
//
// 返回：验证错误，nil 表示验证成功
func ValidateMapMustHaveKeys(kvs map[string]any, keys ...string) error {
	// 安全检查：防止 kvs 为 nil
	if kvs == nil {
		if len(keys) > 0 {
			return fmt.Errorf("map validation failed: map cannot be nil when required keys are specified")
		}
		return nil
	}

	if len(keys) == 0 {
		return nil
	}

	// 优化：收集所有缺失的键，一次性报告
	var missingKeys []string
	for _, key := range keys {
		if _, exists := kvs[key]; !exists {
			missingKeys = append(missingKeys, key)
		}
	}

	if len(missingKeys) > 0 {
		return fmt.Errorf("map validation failed: missing required keys: %s", strings.Join(missingKeys, ", "))
	}

	return nil
}

// ValidateMapStringKey 验证 map[string]any 中字符串类型的键
// 验证指定键的值是否为字符串类型，并检查长度限制
// 参数：
//
//	kvs: 待验证的 map
//	key: 键名
//	minLen: 最小长度（0 表示不限制）
//	maxLen: 最大长度（0 表示不限制）
//
// 返回：验证错误，nil 表示验证成功
func ValidateMapStringKey(kvs map[string]any, key string, minLen, maxLen int) error {
	// 安全检查：防止 kvs 为 nil
	if kvs == nil {
		return nil
	}

	// 安全检查：防止无效的长度参数
	if minLen < 0 {
		return fmt.Errorf("map string key validation failed: minLen cannot be negative")
	}
	if maxLen < 0 {
		return fmt.Errorf("map string key validation failed: maxLen cannot be negative")
	}
	if minLen > 0 && maxLen > 0 && minLen > maxLen {
		return fmt.Errorf("map string key validation failed: minLen cannot be greater than maxLen")
	}

	value, exists := kvs[key]
	if !exists {
		return nil // 键不存在时不验证
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("map key '%s' validation failed: value must be string type, got %T", key, value)
	}

	strLen := len(str)
	if minLen > 0 && strLen < minLen {
		return fmt.Errorf("map key '%s' validation failed: string length must be at least %d, got %d", key, minLen, strLen)
	}

	if maxLen > 0 && strLen > maxLen {
		return fmt.Errorf("map key '%s' validation failed: string length must be at most %d, got %d", key, maxLen, strLen)
	}

	return nil
}

// ValidateMapIntKey 验证 map[string]any 中整数类型的键
// 验证指定键的值是否为整数类型，并检查值范围
// 支持多种整数类型的自动转换
// 参数：
//
//	kvs: 待验证的 map
//	key: 键名
//	min: 最小值
//	max: 最大值
//
// 返回：验证错误，nil 表示验证成功
func ValidateMapIntKey(kvs map[string]any, key string, min, max int) error {
	// 安全检查：防止 kvs 为 nil
	if kvs == nil {
		return nil
	}

	// 安全检查：防止无效的范围参数
	if min > max {
		return fmt.Errorf("map int key validation failed: min cannot be greater than max")
	}

	value, exists := kvs[key]
	if !exists {
		return nil // 键不存在时不验证
	}

	// 尝试转换为整数（支持多种数值类型）
	var intValue int
	switch v := value.(type) {
	case int:
		intValue = v
	case int64:
		// 安全检查：防止整数溢出
		if v > math.MaxInt || v < math.MinInt {
			return fmt.Errorf("map key '%s' validation failed: int64 value overflows int type", key)
		}
		intValue = int(v)
	case int32:
		intValue = int(v)
	case int16:
		intValue = int(v)
	case int8:
		intValue = int(v)
	case uint:
		// 安全检查：防止整数溢出
		if v > uint(math.MaxInt) {
			return fmt.Errorf("map key '%s' validation failed: uint value overflows int type", key)
		}
		intValue = int(v)
	case uint64:
		// 安全检查：防止整数溢出
		if v > uint64(math.MaxInt) {
			return fmt.Errorf("map key '%s' validation failed: uint64 value overflows int type", key)
		}
		intValue = int(v)
	case uint32:
		// 安全检查：防止整数溢出
		// 在 32 位系统上 math.MaxInt 可能小于 math.MaxUint32
		if uint64(v) > uint64(math.MaxInt) {
			return fmt.Errorf("map key '%s' validation failed: uint32 value overflows int type", key)
		}
		intValue = int(v)
	case uint16:
		intValue = int(v)
	case uint8:
		intValue = int(v)
	case float64:
		// 检查是否为整数
		if v != float64(int(v)) {
			return fmt.Errorf("map key '%s' validation failed: float64 value is not an integer", key)
		}
		// 安全检查：防止整数溢出
		if v > float64(math.MaxInt) || v < float64(math.MinInt) {
			return fmt.Errorf("map key '%s' validation failed: float64 value overflows int type", key)
		}
		intValue = int(v)
	case float32:
		// 检查是否为整数
		if v != float32(int(v)) {
			return fmt.Errorf("map key '%s' validation failed: float32 value is not an integer", key)
		}
		intValue = int(v)
	default:
		return fmt.Errorf("map key '%s' validation failed: value must be integer type, got %T", key, value)
	}

	if intValue < min {
		return fmt.Errorf("map key '%s' validation failed: value must be at least %d, got %d", key, min, intValue)
	}

	if intValue > max {
		return fmt.Errorf("map key '%s' validation failed: value must be at most %d, got %d", key, max, intValue)
	}

	return nil
}

// ValidateMapFloatKey 验证 map[string]any 中浮点数类型的键
// 验证指定键的值是否为数字类型，并检查值范围
// 支持多种数值类型的自动转换
// 参数：
//
//	kvs: 待验证的 map
//	key: 键名
//	min: 最小值
//	max: 最大值
//
// 返回：验证错误，nil 表示验证成功
func ValidateMapFloatKey(kvs map[string]any, key string, min, max float64) error {
	// 安全检查：防止 kvs 为 nil
	if kvs == nil {
		return nil
	}

	// 安全检查：防止无效的范围参数
	if min > max {
		return fmt.Errorf("map float key validation failed: min cannot be greater than max")
	}

	// 安全检查：防止 NaN 和 Inf
	if math.IsNaN(min) || math.IsNaN(max) {
		return fmt.Errorf("map float key validation failed: min and max cannot be NaN")
	}
	if math.IsInf(min, 0) || math.IsInf(max, 0) {
		return fmt.Errorf("map float key validation failed: min and max cannot be Inf")
	}

	value, exists := kvs[key]
	if !exists {
		return nil
	}

	// 尝试转换为浮点数（支持多种数值类型）
	var floatValue float64
	switch v := value.(type) {
	case float64:
		floatValue = v
	case float32:
		floatValue = float64(v)
	case int:
		floatValue = float64(v)
	case int64:
		floatValue = float64(v)
	case int32:
		floatValue = float64(v)
	case int16:
		floatValue = float64(v)
	case int8:
		floatValue = float64(v)
	case uint:
		floatValue = float64(v)
	case uint64:
		floatValue = float64(v)
	case uint32:
		floatValue = float64(v)
	case uint16:
		floatValue = float64(v)
	case uint8:
		floatValue = float64(v)
	default:
		return fmt.Errorf("map key '%s' validation failed: value must be numeric type, got %T", key, value)
	}

	// 安全检查：防止 NaN 和 Inf
	if math.IsNaN(floatValue) {
		return fmt.Errorf("map key '%s' validation failed: value cannot be NaN", key)
	}
	if math.IsInf(floatValue, 0) {
		return fmt.Errorf("map key '%s' validation failed: value cannot be Inf", key)
	}

	if floatValue < min {
		return fmt.Errorf("map key '%s' validation failed: value must be at least %f, got %f", key, min, floatValue)
	}

	if floatValue > max {
		return fmt.Errorf("map key '%s' validation failed: value must be at most %f, got %f", key, max, floatValue)
	}

	return nil
}

// ValidateMapBoolKey 验证 map[string]any 中布尔类型的键
// 验证指定键的值是否为布尔类型
// 参数：
//
//	kvs: 待验证的 map
//	key: 键名
//
// 返回：验证错误，nil 表示验证成功
func ValidateMapBoolKey(kvs map[string]any, key string) error {
	// 安全检查：防止 kvs 为 nil
	if kvs == nil {
		return nil
	}

	value, exists := kvs[key]
	if !exists {
		return nil
	}

	if _, ok := value.(bool); !ok {
		return fmt.Errorf("map key '%s' validation failed: value must be bool type, got %T", key, value)
	}

	return nil
}

// NewMapValidator 创建一个新的 MapValidator
// 返回一个已初始化的 MapValidator 实例，可通过链式调用配置
func NewMapValidator() *MapValidator {
	return &MapValidator{
		RequiredKeys:  make([]string, 0),
		AllowedKeys:   make([]string, 0),
		KeyValidators: make(map[string]func(value any) error),
	}
}

// WithRequiredKeys 设置必填键（链式调用）
// 参数：
//
//	keys: 必填的键名列表
//
// 返回：MapValidator 实例，支持链式调用
func (mv *MapValidator) WithRequiredKeys(keys ...string) *MapValidator {
	// 安全检查：防止 nil 切片
	if keys == nil {
		mv.RequiredKeys = make([]string, 0)
		return mv
	}
	mv.RequiredKeys = keys
	return mv
}

// WithAllowedKeys 设置允许的键（链式调用）
// 参数：
//
//	keys: 允许的键名列表
//
// 返回：MapValidator 实例，支持链式调用
func (mv *MapValidator) WithAllowedKeys(keys ...string) *MapValidator {
	mv.mu.Lock()
	defer mv.mu.Unlock()

	// 安全检查：防止 nil 切片
	if keys == nil {
		mv.AllowedKeys = make([]string, 0)
	} else {
		mv.AllowedKeys = keys
	}

	// 清除缓存，下次验证时重新构建
	mv.allowedKeysMap = nil
	return mv
}

// WithKeyValidator 添加键验证器（链式调用）
// 参数：
//
//	key: 键名
//	validatorFunc: 验证函数
//
// 返回：MapValidator 实例，支持链式调用
func (mv *MapValidator) WithKeyValidator(key string, validatorFunc func(value any) error) *MapValidator {
	// 安全检查：确保 KeyValidators map 已初始化
	if mv.KeyValidators == nil {
		mv.KeyValidators = make(map[string]func(value any) error)
	}

	// 安全检查：防止空键名
	if key == "" {
		return mv
	}

	mv.KeyValidators[key] = validatorFunc
	return mv
}

// AddRequiredKey 添加单个必填键
// 参数：
//
//	key: 必填的键名
//
// 返回：MapValidator 实例，支持链式调用
func (mv *MapValidator) AddRequiredKey(key string) *MapValidator {
	// 安全检查：防止空键名
	if key == "" {
		return mv
	}

	// 安全检查：确保切片已初始化
	if mv.RequiredKeys == nil {
		mv.RequiredKeys = make([]string, 0)
	}

	mv.RequiredKeys = append(mv.RequiredKeys, key)
	return mv
}

// AddAllowedKey 添加单个允许的键
// 参数：
//
//	key: 允许的键名
//
// 返回：MapValidator 实例，支持链式调用
func (mv *MapValidator) AddAllowedKey(key string) *MapValidator {
	// 安全检查：防止空键名
	if key == "" {
		return mv
	}

	mv.mu.Lock()
	defer mv.mu.Unlock()

	// 安全检查：确保切片已初始化
	if mv.AllowedKeys == nil {
		mv.AllowedKeys = make([]string, 0)
	}

	mv.AllowedKeys = append(mv.AllowedKeys, key)
	// 清除缓存，下次验证时重新构建
	mv.allowedKeysMap = nil
	return mv
}

// Validate 验证 map（方法形式，支持链式调用后直接验证）
// 参数：
//
//	kvs: 待验证的 map
//
// 返回：验证错误，nil 表示验证成功
func (mv *MapValidator) Validate(kvs map[string]any) []*FieldError {
	return ValidateMap(kvs, mv)
}
