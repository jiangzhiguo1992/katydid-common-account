package validator

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type MapValidators struct {
	Validators map[ValidateScene]*MapValidator
}

// MapValidator Map 字段验证器，专门用于验证 map[string]any 类型的动态扩展字段
// 设计目标：
//   - 单一职责：只负责 map 类型的验证逻辑
//   - 开放封闭：通过 KeyValidators 支持扩展，无需修改核心代码
//   - 高内聚低耦合：独立的验证逻辑，不依赖其他验证器
//
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
}

// 常量定义：用于错误消息和性能优化
const (
	// defaultMapCapacity 默认 map 容量，用于预分配内存
	defaultMapCapacity = 8
	// maxMapKeyLength 最大键名长度，防止恶意超长键名
	maxMapKeyLength = 256
)

func ValidateMaps(scene ValidateScene, kvs map[string]any, validators *MapValidators) []*FieldError {
	if validators == nil || len(validators.Validators) == 0 {
		return nil
	}

	validator, exists := validators.Validators[scene]
	if !exists {
		return nil
	}

	return ValidateMap(kvs, validator)
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
				// 这里的namespace不填，统一错误信息模板
				NewFieldError("map", "map", "required", "", "").
					WithMessage("map field cannot be nil when required keys are specified"),
			}
		}
		return nil
	}

	// 创建验证上下文（场景为空字符串，因为 map 验证场景已在外部区分）
	ctx := NewValidationContext("")

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

	// 返回验证结果
	if ctx.HasErrors() {
		return ctx.Errors
	} else if len(ctx.Message) != 0 {
		// 这里的namespace不填，统一错误信息模板
		return []*FieldError{NewFieldError("", "", ctx.Message, "", "")}
	}

	return nil
}

// collectRequiredKeyErrors 收集必填键错误
// 遍历所有必填键，检查它们是否存在于 map 中
// 性能优化：map 查找的时间复杂度为 O(1)
func (mv *MapValidator) collectRequiredKeyErrors(kvs map[string]any, ctx *ValidationContext) {
	if ctx == nil || len(mv.RequiredKeys) == 0 {
		return
	}

	for _, key := range mv.RequiredKeys {
		// 安全检查：防止超长键名攻击
		if len(key) > maxMapKeyLength {
			// 这里的namespace不填，统一错误信息模板
			ctx.AddErrorByDetail(
				"map", "map", "key_len", strconv.Itoa(maxMapKeyLength), "",
				fmt.Sprintf("key name exceeds maximum length %d", maxMapKeyLength),
				len(key),
			)
			continue
		}

		if _, exists := kvs[key]; !exists {
			ctx.AddErrorByDetail(
				key, key, "required", "", mv.getNamespace(key),
				fmt.Sprintf("required key '%s' is missing", key),
				nil,
			)
		}
	}
}

// collectAllowedKeyErrors 收集非法键错误（白名单验证）
// 使用懒加载+双重检查锁模式构建允许键缓存，提升性能
func (mv *MapValidator) collectAllowedKeyErrors(kvs map[string]any, ctx *ValidationContext) {
	if ctx == nil || len(mv.AllowedKeys) == 0 {
		return
	}

	// 性能优化：懒加载 allowedKeysMap
	if mv.allowedKeysMap == nil {
		// 内存优化：预分配精确的容量
		mv.allowedKeysMap = make(map[string]bool, len(mv.AllowedKeys))
		for _, key := range mv.AllowedKeys {
			mv.allowedKeysMap[key] = true
		}
	}

	// 检查所有键是否在白名单中
	for key := range kvs {
		// 安全检查：防止超长键名攻击
		if len(key) > maxMapKeyLength {
			// 这里的namespace不填，统一错误信息模板
			ctx.AddErrorByDetail(
				"map", "map", "key_len", strconv.Itoa(maxMapKeyLength), "",
				fmt.Sprintf("key name exceeds maximum length %d", maxMapKeyLength),
				len(key),
			)
			continue
		}

		if !mv.allowedKeysMap[key] {
			ctx.AddErrorByDetail(
				key, key, "allowed", "", mv.getNamespace(key),
				fmt.Sprintf("key '%s' is not in the allowed list", key),
				key,
			)
		}
	}
}

// collectCustomKeyErrors 收集自定义键验证错误
// 执行用户定义的复杂验证逻辑
// 错误恢复：即使某个验证函数 panic，也不影响其他验证
func (mv *MapValidator) collectCustomKeyErrors(kvs map[string]any, ctx *ValidationContext) {
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
					// 这里的namespace不填，统一错误信息模板
					ctx.AddErrorByDetail(
						"map", "map", "validator_panic", "", "",
						fmt.Sprintf("validator function panicked: %v", r),
						value,
					)
				}
			}()

			if err := validatorFunc(value); err != nil {
				ctx.AddErrorByDetail(
					key, key, "custom", "", mv.getNamespace(key),
					err.Error(),
					value,
				)
			}
		}()
	}
}

// getNamespace 获取完整的命名空间路径
// 用于生成准确的错误定位信息
func (mv *MapValidator) getNamespace(key string) string {
	if mv.ParentNameSpace == "" {
		return key
	}
	// 内存优化：使用 strings.Builder
	var builder strings.Builder
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
	for _, key := range keys {
		if key == "" {
			continue // 忽略空键名
		}
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
	switch v := value.(type) {
	case int:
		intValue = v
	case int64:
		// 安全检查：防止整数溢出
		if v > math.MaxInt || v < math.MinInt {
			return fmt.Errorf("map key '%s' validation failed: int64 value %d overflows int type", key, v)
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
			return fmt.Errorf("map key '%s' validation failed: uint value %d overflows int type", key, v)
		}
		intValue = int(v)
	case uint64:
		// 安全检查：防止整数溢出
		if v > uint64(math.MaxInt) {
			return fmt.Errorf("map key '%s' validation failed: uint64 value %d overflows int type", key, v)
		}
		intValue = int(v)
	case uint32:
		// 安全检查：防止整数溢出（在 32 位系统上可能溢出）
		if uint64(v) > uint64(math.MaxInt) {
			return fmt.Errorf("map key '%s' validation failed: uint32 value %d overflows int type", key, v)
		}
		intValue = int(v)
	case uint16:
		intValue = int(v)
	case uint8:
		intValue = int(v)
	case float64:
		// 检查是否为整数
		if v != float64(int(v)) {
			return fmt.Errorf("map key '%s' validation failed: float64 value %f is not an integer", key, v)
		}
		// 安全检查：防止整数溢出
		if v > float64(math.MaxInt) || v < float64(math.MinInt) {
			return fmt.Errorf("map key '%s' validation failed: float64 value %f overflows int type", key, v)
		}
		intValue = int(v)
	case float32:
		// 检查是否为整数
		if v != float32(int(v)) {
			return fmt.Errorf("map key '%s' validation failed: float32 value %f is not an integer", key, v)
		}
		intValue = int(v)
	default:
		return fmt.Errorf("map key '%s' validation failed: value must be integer type, got %T", key, value)
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

func (mv *MapValidators) Validate(scene ValidateScene, kvs map[string]any) []*FieldError {
	return ValidateMaps(scene, kvs, mv)
}
