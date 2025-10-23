package v1

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
)

// MapValidators 场景化的Map验证器集合
// 设计目标：支持不同验证场景使用不同的验证规则
// 线程安全性：只读操作，多个goroutine可以并发使用
type MapValidators struct {
	// Validators 场景到验证器的映射
	Validators map[ValidateScene]*MapValidator
}

// MapValidator Map 字段验证器，专门用于验证 map[string]any 类型的动态扩展字段
// 设计目标：
//   - 单一职责：只负责 map 类型的验证逻辑
//   - 开放封闭：通过 KeyValidators 支持扩展，无需修改核心代码
//   - 高内聚低耦合：独立的验证逻辑，不依赖其他验证器
//
// 线程安全性：allowedKeysMap 的懒加载使用 sync.Once 保证线程安全
// 性能优化：使用缓存减少重复计算，预分配内存减少动态扩容
type MapValidator struct {
	// ParentNameSpace 结构体命名空间，用于生成准确的错误路径
	// 例如：User.Extras, Product.Metadata
	ParentNameSpace string

	// RequiredKeys 必填的键列表，这些键必须存在于 map 中
	RequiredKeys []string

	// AllowedKeys 允许的键白名单（如果为空则不限制）
	// 用于防止非法字段注入，提升数据安全性
	AllowedKeys []string

	// KeyValidators 特定键的自定义验证函数映射
	// key: 字段名，value: 验证函数（返回 error 表示验证失败）
	// 支持复杂的业务验证逻辑
	KeyValidators map[string]func(value any) error

	// allowedKeysMap 内部缓存的允许键 map（性能优化）
	// 使用 map 查找的时间复杂度为 O(1)，优于切片遍历的 O(n)
	allowedKeysMap map[string]bool

	// initOnce 确保 allowedKeysMap 只初始化一次（线程安全）
	initOnce sync.Once
}

// 常量定义：用于错误消息和性能优化
const (
	// maxMapKeyLength 最大键名长度，防止恶意超长键名
	maxMapKeyLength = 256

	// maxMapSize 最大 map 大小，防止 DoS 攻击
	maxMapSize = 10000

	// maxMapValueSize 单个值的最大大小（字节），防止内存溢出
	maxMapValueSize = 1024 * 1024 // 1MB
)

// ValidateMaps 验证 map 字段（场景化）
// 根据验证场景匹配相应的验证器并执行验证
// 参数：
//
//	scene: 验证场景
//	kvs: 待验证的 map 数据
//	validators: 场景化的验证器集合
//
// 返回：
//
//	验证错误列表，nil 表示验证通过
func ValidateMaps(scene ValidateScene, kvs map[string]any, validators *MapValidators) []*FieldError {
	// 防御性编程：参数校验
	if validators == nil || len(validators.Validators) == 0 {
		return nil
	}

	// 使用位运算匹配场景：遍历所有配置的场景，找到匹配的验证器
	// 支持场景组合：例如 SceneCreate | SceneUpdate 可以匹配多个场景
	var matchedValidators []*MapValidator
	for configScene, validator := range validators.Validators {
		if validator == nil {
			continue // 跳过 nil 验证器
		}
		if scene&configScene != 0 {
			// 找到匹配的场景验证器
			matchedValidators = append(matchedValidators, validator)
		}
	}

	if len(matchedValidators) == 0 {
		return nil
	}

	// 遍历验证，收集所有错误
	var allErrors []*FieldError
	for _, mv := range matchedValidators {
		errors := ValidateMap(kvs, mv)
		if len(errors) > 0 {
			allErrors = append(allErrors, errors...)
		}
	}
	return allErrors
}

// ValidateMap 验证 map[string]any 类型的扩展字段
// 验证流程：
//  1. 检查必填键是否存在
//  2. 检查是否包含非法键（白名单模式）
//  3. 执行自定义键验证器
//
// 错误收集策略：收集所有错误后统一返回，而非遇到第一个错误就停止
// 参数：
//
//	kvs: 待验证的 map 数据
//	v: MapValidator 验证器配置
//
// 返回：
//
//	验证错误列表，nil 表示验证通过
func ValidateMap(kvs map[string]any, v *MapValidator) []*FieldError {
	// 防御性编程：如果验证器为 nil，则跳过验证
	if v == nil {
		return nil
	}

	// 安全检查：防止 kvs 为 nil 导致 panic
	if kvs == nil {
		// 如果有必填键要求，则 nil map 是错误的
		if len(v.RequiredKeys) > 0 {
			return []*FieldError{
				NewFieldError("map", "required", "").
					WithMessage("map field cannot be nil when required keys are specified"),
			}
		}
		return nil
	}

	// 安全检查：防止 DoS 攻击 - 限制 map 大小
	if len(kvs) > maxMapSize {
		return []*FieldError{
			NewFieldError("map", "size", strconv.Itoa(maxMapSize)).
				WithMessage(fmt.Sprintf("map size exceeds maximum limit %d", maxMapSize)),
		}
	}

	// 创建验证上下文（场景为0，因为 map 验证场景已在外部区分）
	ctx := NewValidationContext(0)
	defer ReleaseValidationContext(ctx) // 使用后归还到池

	// 1. 验证必填键（业务逻辑错误）
	if len(v.RequiredKeys) > 0 {
		v.collectRequiredKeyErrors(kvs, ctx)
	}

	// 2. 验证允许的键（安全性检查，防止字段注入）
	if len(v.AllowedKeys) > 0 {
		v.collectAllowedKeyErrors(kvs, ctx)
	}

	// 3. 执行自定义键验证器（复杂业务逻辑）
	if len(v.KeyValidators) > 0 {
		v.collectCustomKeyErrors(kvs, ctx)
	}

	if !ctx.HasErrors() {
		if len(ctx.Message) != 0 {
			return []*FieldError{NewFieldError("", "", "").WithMessage(ctx.Message)}
		}
	}

	// 必须复制错误列表，因为 ctx 会被归还到对象池
	// 内存优化：精确分配容量，避免浪费
	errs := make([]*FieldError, len(ctx.Errors))
	copy(errs, ctx.Errors)
	return errs
}

// collectRequiredKeyErrors 收集必填键错误
// 遍历所有必填键，检查它们是否存在于 map 中
// 性能优化：map 查找的时间复杂度为 O(1)
func (mv *MapValidator) collectRequiredKeyErrors(kvs map[string]any, ctx *ValidationContext) {
	// 防御性编程：参数校验
	if ctx == nil || len(mv.RequiredKeys) == 0 {
		return
	}

	for _, key := range mv.RequiredKeys {
		// 安全检查：防止超长键名攻击
		if len(key) > maxMapKeyLength {
			ctx.AddErrorByDetail(
				"map", "key_len", strconv.Itoa(maxMapKeyLength), len(key),
				fmt.Sprintf("required key name exceeds maximum length %d", maxMapKeyLength),
			)
			continue
		}

		// 安全检查：防止空键名或包含危险字符
		if err := validateKeyName(key); err != nil {
			ctx.AddErrorByDetail(
				"map", "invalid_key", "", key,
				fmt.Sprintf("invalid required key name '%s': %v", key, err),
			)
			continue
		}

		// 检查键是否存在
		if _, exists := kvs[key]; !exists {
			ctx.AddErrorByDetail(
				mv.getNamespace(key), "required", "", nil,
				fmt.Sprintf("required key '%s' is missing", key),
			)
		}
	}
}

// collectAllowedKeyErrors 收集非法键错误（白名单验证）
// 使用懒加载+线程安全的方式构建允许键缓存，提升性能
func (mv *MapValidator) collectAllowedKeyErrors(kvs map[string]any, ctx *ValidationContext) {
	// 防御性编程：参数校验
	if ctx == nil || len(mv.AllowedKeys) == 0 {
		return
	}

	// 性能优化：懒加载 allowedKeysMap
	mv.initOnce.Do(func() {
		// 内存优化：预分配精确的容量
		mv.allowedKeysMap = make(map[string]bool, len(mv.AllowedKeys))
		for _, key := range mv.AllowedKeys {
			mv.allowedKeysMap[key] = true
		}
	})

	// 检查所有键是否在白名单中
	for key := range kvs {
		// 安全检查：防止超长键名攻击
		if len(key) > maxMapKeyLength {
			ctx.AddErrorByDetail(
				"map", "key_len", strconv.Itoa(maxMapKeyLength), len(key),
				fmt.Sprintf("key name exceeds maximum length %d", maxMapKeyLength),
			)
			continue
		}

		// 安全检查：验证键名有效性
		if err := validateKeyName(key); err != nil {
			ctx.AddErrorByDetail(
				"map", "invalid_key", "", key,
				fmt.Sprintf("invalid key name '%s': %v", key, err),
			)
			continue
		}

		// 检查键是否在白名单中
		if !mv.allowedKeysMap[key] {
			ctx.AddErrorByDetail(
				mv.getNamespace(key), "not_allowed", "", key,
				fmt.Sprintf("key '%s' is not in the allowed list", key),
			)
		}
	}
}

// collectCustomKeyErrors 收集自定义键验证错误
// 执行用户定义的复杂验证逻辑
// 错误恢复：即使某个验证函数 panic，也不影响其他验证
func (mv *MapValidator) collectCustomKeyErrors(kvs map[string]any, ctx *ValidationContext) {
	// 防御性编程：参数校验
	if ctx == nil || len(mv.KeyValidators) == 0 {
		return
	}

	for key, validatorFunc := range mv.KeyValidators {
		// 安全检查：防止 nil 验证函数
		if validatorFunc == nil {
			continue
		}

		value, exists := kvs[key]
		if !exists {
			continue // 键不存在时不执行验证
		}

		// 错误恢复：防止验证函数 panic 导致整个验证流程中断
		func() {
			defer func() {
				if r := recover(); r != nil {
					ctx.AddErrorByDetail(
						"map", "validator_panic", "", value,
						fmt.Sprintf("validator function for namsepace '%s' panicked: %v", mv.getNamespace(key), r),
					)
				}
			}()

			if err := validatorFunc(value); err != nil {
				ctx.AddErrorByDetail(
					mv.getNamespace(key), "custom", "", value,
					err.Error(),
				)
			}
		}()
	}
}

// validateKeyName 验证键名的有效性
// 防止注入攻击和非法字符
// 参数：
//
//	key: 待验证的键名
//
// 返回：
//
//	error: 如果键名无效返回错误，否则返回 nil
func validateKeyName(key string) error {
	if key == "" {
		return fmt.Errorf("key name cannot be empty")
	}

	// 检查是否包含控制字符（ASCII 0-31）
	for _, r := range key {
		if r < 32 {
			return fmt.Errorf("key name contains control character (code %d)", r)
		}
		// 检查是否包含危险字符（防止注入攻击）
		if r == 0x7F { // DEL 字符
			return fmt.Errorf("key name contains delete character")
		}
	}

	return nil
}

// getNamespace 获取完整的命名空间路径
// 用于生成准确的错误定位信息
// 参数：
//
//	key: 键名
//
// 返回：
//
//	完整的命名空间路径
func (mv *MapValidator) getNamespace(key string) string {
	if mv.ParentNameSpace == "" {
		return key
	}

	// 内存优化：从对象池获取 strings.Builder
	builder := acquireStringBuilder()
	defer releaseStringBuilder(builder)

	builder.Grow(len(mv.ParentNameSpace) + len(key) + 1)
	builder.WriteString(mv.ParentNameSpace)
	builder.WriteString(".")
	builder.WriteString(key)
	return builder.String()
}

// ValidateMapKey 验证 map[string]any 中特定键的值
// 便捷函数：简化单个键的验证逻辑
// 参数：
//
//	kvs: 待验证的 map
//	key: 键名
//	validatorFunc: 验证函数
//
// 返回：
//
//	验证错误，nil 表示验证成功
func ValidateMapKey(kvs map[string]any, key string, validatorFunc func(value any) error) error {
	// 安全检查：防止 kvs 为 nil
	if kvs == nil {
		return nil // 键不存在时不验证
	}

	// 安全检查：防止验证函数为 nil
	if validatorFunc == nil {
		return fmt.Errorf("map key validation failed: validator function cannot be nil")
	}

	// 安全检查：防止空键名
	if key == "" {
		return fmt.Errorf("map key validation failed: key name cannot be empty")
	}

	// 安全检查：验证键名有效性
	if err := validateKeyName(key); err != nil {
		return fmt.Errorf("map key validation failed: %w", err)
	}

	value, exists := kvs[key]
	if !exists {
		return nil // 键不存在时不验证
	}

	// 错误恢复：防止验证函数 panic
	var validationErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				validationErr = fmt.Errorf("map key '%s' validation failed: validator panicked: %v", key, r)
			}
		}()
		validationErr = validatorFunc(value)
	}()

	if validationErr != nil {
		return fmt.Errorf("map key '%s' validation failed: %w", key, validationErr)
	}

	return nil
}

// ValidateMapMustHaveKey 验证 map[string]any 必须包含指定的键
// 便捷函数：简化必填键验证
// 参数：
//
//	kvs: 待验证的 map
//	key: 必须存在的键名
//
// 返回：
//
//	验证错误，nil 表示验证成功
func ValidateMapMustHaveKey(kvs map[string]any, key string) error {
	// 安全检查：防止 kvs 为 nil
	if kvs == nil {
		return fmt.Errorf("map validation failed: map cannot be nil")
	}

	// 安全检查：防止空键名
	if key == "" {
		return fmt.Errorf("map validation failed: key name cannot be empty")
	}

	// 安全检查：验证键名有效性
	if err := validateKeyName(key); err != nil {
		return fmt.Errorf("map validation failed: %w", err)
	}

	if _, exists := kvs[key]; !exists {
		return fmt.Errorf("map validation failed: missing required key '%s'", key)
	}
	return nil
}

// ValidateMapMustHaveKeys 验证 map[string]any 必须包含指定的多个键
// 性能优化：一次性检查所有键，收集所有缺失的键名
// 参数：
//
//	kvs: 待验证的 map
//	keys: 必须存在的键名列表
//
// 返回：
//
//	验证错误，nil 表示验证成功
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

	// 性能优化：收集所有缺失的键，一次性报告（而非遇到第一个就返回）
	var missingKeys []string
	var invalidKeys []string

	for _, key := range keys {
		if key == "" {
			continue // 忽略空键名
		}

		// 验证键名有效性
		if err := validateKeyName(key); err != nil {
			invalidKeys = append(invalidKeys, key)
			continue
		}

		if _, exists := kvs[key]; !exists {
			missingKeys = append(missingKeys, key)
		}
	}

	// 构建错误消息
	if len(invalidKeys) > 0 || len(missingKeys) > 0 {
		// 内存优化：从对象池获取 strings.Builder
		errMsg := acquireStringBuilder()
		defer releaseStringBuilder(errMsg)

		errMsg.WriteString("map validation failed: ")

		if len(invalidKeys) > 0 {
			errMsg.WriteString(fmt.Sprintf("invalid key names: %s", strings.Join(invalidKeys, ", ")))
		}

		if len(missingKeys) > 0 {
			if len(invalidKeys) > 0 {
				errMsg.WriteString("; ")
			}
			errMsg.WriteString(fmt.Sprintf("missing required keys: %s", strings.Join(missingKeys, ", ")))
		}

		return fmt.Errorf("%s", errMsg.String())
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
// 返回：
//
//	验证错误，nil 表示验证成功
func ValidateMapStringKey(kvs map[string]any, key string, minLen, maxLen int) error {
	// 安全检查：防止 kvs 为 nil
	if kvs == nil {
		return nil
	}

	// 安全检查：防止空键名
	if key == "" {
		return fmt.Errorf("map string key validation failed: key name cannot be empty")
	}

	// 安全检查：防止无效的长度参数
	if minLen < 0 {
		return fmt.Errorf("map string key validation failed: minLen cannot be negative")
	}
	if maxLen < 0 {
		return fmt.Errorf("map string key validation failed: maxLen cannot be negative")
	}
	if minLen > 0 && maxLen > 0 && minLen > maxLen {
		return fmt.Errorf("map string key validation failed: minLen (%d) cannot be greater than maxLen (%d)", minLen, maxLen)
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
// 支持多种整数类型的自动转换，并进行溢出检查
// 参数：
//
//	kvs: 待验证的 map
//	key: 键名
//	min: 最小值
//	max: 最大值
//
// 返回：
//
//	验证错误，nil 表示验证成功
func ValidateMapIntKey(kvs map[string]any, key string, min, max int) error {
	// 安全检查：防止 kvs 为 nil
	if kvs == nil {
		return nil
	}

	// 安全检查：防止空键名
	if key == "" {
		return fmt.Errorf("map int key validation failed: key name cannot be empty")
	}

	// 安全检查：防止无效的范围参数
	if min > max {
		return fmt.Errorf("map int key validation failed: min (%d) cannot be greater than max (%d)", min, max)
	}

	value, exists := kvs[key]
	if !exists {
		return nil // 键不存在时不验证
	}

	// 性能优化：尝试转换为整数（支持多种数值类型）
	var intValue int
	var convertErr error

	switch v := value.(type) {
	case int:
		intValue = v
	case int64:
		// 安全检查：防止整数溢出
		if v > int64(math.MaxInt) || v < int64(math.MinInt) {
			convertErr = fmt.Errorf("int64 value %d overflows int type", v)
		} else {
			intValue = int(v)
		}
	case int32:
		intValue = int(v)
	case int16:
		intValue = int(v)
	case int8:
		intValue = int(v)
	case uint:
		// 安全检查：防止整数溢出
		if v > uint(math.MaxInt) {
			convertErr = fmt.Errorf("uint value %d overflows int type", v)
		} else {
			intValue = int(v)
		}
	case uint64:
		// 安全检查：防止整数溢出
		if v > uint64(math.MaxInt) {
			convertErr = fmt.Errorf("uint64 value %d overflows int type", v)
		} else {
			intValue = int(v)
		}
	case uint32:
		// 安全检查：防止整数溢出（在 32 位系统上可能溢出）
		if uint64(v) > uint64(math.MaxInt) {
			convertErr = fmt.Errorf("uint32 value %d overflows int type", v)
		} else {
			intValue = int(v)
		}
	case uint16:
		intValue = int(v)
	case uint8:
		intValue = int(v)
	case float64:
		// 检查是否为整数
		if v != float64(int(v)) {
			convertErr = fmt.Errorf("float64 value %f is not an integer", v)
		} else if v > float64(math.MaxInt) || v < float64(math.MinInt) {
			// 安全检查：防止整数溢出
			convertErr = fmt.Errorf("float64 value %f overflows int type", v)
		} else {
			intValue = int(v)
		}
	case float32:
		// 检查是否为整数
		if v != float32(int(v)) {
			convertErr = fmt.Errorf("float32 value %f is not an integer", v)
		} else {
			intValue = int(v)
		}
	default:
		return fmt.Errorf("map key '%s' validation failed: value must be integer type, got %T", key, value)
	}

	if convertErr != nil {
		return fmt.Errorf("map key '%s' validation failed: %w", key, convertErr)
	}

	// 范围检查
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
// 支持多种数值类型的自动转换，并进行 NaN/Inf 检查
// 参数：
//
//	kvs: 待验证的 map
//	key: 键名
//	min: 最小值
//	max: 最大值
//
// 返回：
//
//	验证错误，nil 表示验证成功
func ValidateMapFloatKey(kvs map[string]any, key string, min, max float64) error {
	// 安全检查：防止 kvs 为 nil
	if kvs == nil {
		return nil
	}

	// 安全检查：防止空键名
	if key == "" {
		return fmt.Errorf("map float key validation failed: key name cannot be empty")
	}

	// 安全检查：防止无效的范围参数
	if min > max {
		return fmt.Errorf("map float key validation failed: min (%f) cannot be greater than max (%f)", min, max)
	}

	// 安全检查：防止 NaN 和 Inf 参数
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

	// 安全检查：防止 NaN 和 Inf 值
	if math.IsNaN(floatValue) {
		return fmt.Errorf("map key '%s' validation failed: value cannot be NaN", key)
	}
	if math.IsInf(floatValue, 0) {
		return fmt.Errorf("map key '%s' validation failed: value cannot be Inf", key)
	}

	// 范围检查
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
// 返回：
//
//	验证错误，nil 表示验证成功
func ValidateMapBoolKey(kvs map[string]any, key string) error {
	// 安全检查：防止 kvs 为 nil
	if kvs == nil {
		return nil
	}

	// 安全检查：防止空键名
	if key == "" {
		return fmt.Errorf("map bool key validation failed: key name cannot be empty")
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
// 工厂方法模式，确保对象正确初始化
// 返回：
//
//	已初始化的 MapValidator 实例，支持链式调用配置
func NewMapValidator() *MapValidator {
	return &MapValidator{
		RequiredKeys:  make([]string, 0),
		AllowedKeys:   make([]string, 0),
		KeyValidators: make(map[string]func(value any) error),
	}
}

// WithNameSpace 设置命名空间（链式调用）
// 流式接口模式，提升代码可读性
// 参数：
//
//	namespace: 命名空间路径
//
// 返回：
//
//	MapValidator 实例，支持链式调用
func (mv *MapValidator) WithNameSpace(namespace string) *MapValidator {
	mv.ParentNameSpace = namespace
	return mv
}

// WithRequiredKeys 设置必填键（链式调用）
// 参数：
//
//	keys: 必填的键名列表
//
// 返回：
//
//	MapValidator 实例，支持链式调用
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
// 返回：
//
//	MapValidator 实例，支持链式调用
func (mv *MapValidator) WithAllowedKeys(keys ...string) *MapValidator {
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
// 返回：
//
//	MapValidator 实例，支持链式调用
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
// 返回：
//
//	MapValidator 实例，支持链式调用
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
// 返回：
//
//	MapValidator 实例，支持链式调用
func (mv *MapValidator) AddAllowedKey(key string) *MapValidator {
	// 安全检查：防止空键名
	if key == "" {
		return mv
	}

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
// 返回：
//
//	验证错误列表，nil 表示验证成功
func (mv *MapValidator) Validate(kvs map[string]any) []*FieldError {
	return ValidateMap(kvs, mv)
}

// Reset 重置验证器状态，清除缓存
// 用于重用验证器实例，减少内存分配
func (mv *MapValidator) Reset() {
	mv.allowedKeysMap = nil
}

// Validate 场景化Map验证（MapValidators的方法）
// 参数：
//
//	scene: 验证场景
//	kvs: 待验证的 map
//
// 返回：
//
//	验证错误列表，nil 表示验证成功
func (mvs *MapValidators) Validate(scene ValidateScene, kvs map[string]any) []*FieldError {
	return ValidateMaps(scene, kvs, mvs)
}

// NewMapValidators 创建场景化的Map验证器集合
// 工厂方法模式，确保对象正确初始化
// 返回：
//
//	已初始化的 MapValidators 实例
func NewMapValidators() *MapValidators {
	return &MapValidators{
		Validators: make(map[ValidateScene]*MapValidator),
	}
}

// AddValidator 添加场景验证器
// 参数：
//
//	scene: 验证场景
//	validator: 验证器实例
//
// 返回：
//
//	MapValidators 实例，支持链式调用
func (mvs *MapValidators) AddValidator(scene ValidateScene, validator *MapValidator) *MapValidators {
	if mvs == nil {
		return nil
	}

	if mvs.Validators == nil {
		mvs.Validators = make(map[ValidateScene]*MapValidator)
	}

	mvs.Validators[scene] = validator
	return mvs
}
