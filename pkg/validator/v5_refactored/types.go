package v5_refactored

// ============================================================================
// 场景定义
// ============================================================================

// Scene 验证场景，使用位运算支持场景组合
type Scene int64

// 预定义场景
const (
	SceneNone   Scene = 0  // 无场景
	SceneAll    Scene = -1 // 所有场景
	SceneCreate Scene = 1 << iota
	SceneUpdate
	SceneDelete
	SceneQuery
	SceneImport
	SceneExport
)

// Has 检查是否包含指定场景
func (s Scene) Has(scene Scene) bool {
	return s&scene != 0
}

// Add 添加场景
func (s Scene) Add(scene Scene) Scene {
	return s | scene
}

// Remove 移除场景
func (s Scene) Remove(scene Scene) Scene {
	return s &^ scene
}

// IsEmpty 是否为空场景
func (s Scene) IsEmpty() bool {
	return s == SceneNone
}

// IsAll 是否为全部场景
func (s Scene) IsAll() bool {
	return s == SceneAll
}

// ============================================================================
// 字段错误
// ============================================================================

// FieldError 字段错误
// 职责：描述单个字段的验证错误
type FieldError struct {
	// Namespace 字段的完整命名空间路径（如 User.Profile.Email）
	Namespace string

	// Field 字段名（不含路径）
	Field string

	// Tag 验证标签，描述验证规则类型（如 required, email, min, max 等）
	Tag string

	// Param 验证参数，提供验证规则的具体配置值
	Param string

	// Value 字段的实际值
	Value any

	// Message 用户友好的错误消息
	Message string
}

// NewFieldError 创建字段错误
func NewFieldError(namespace, tag string) *FieldError {
	return &FieldError{
		Namespace: namespace,
		Tag:       tag,
	}
}

// NewFieldErrorWithMessage 创建仅带消息的字段错误
func NewFieldErrorWithMessage(message string) *FieldError {
	return &FieldError{
		Message: message,
	}
}

// WithField 设置字段名
func (fe *FieldError) WithField(field string) *FieldError {
	fe.Field = field
	return fe
}

// WithParam 设置参数
func (fe *FieldError) WithParam(param string) *FieldError {
	fe.Param = param
	return fe
}

// WithValue 设置值
func (fe *FieldError) WithValue(value any) *FieldError {
	fe.Value = value
	return fe
}

// WithMessage 设置消息
func (fe *FieldError) WithMessage(message string) *FieldError {
	fe.Message = message
	return fe
}

// Error 实现 error 接口
func (fe *FieldError) Error() string {
	if fe.Message != "" {
		return fe.Message
	}

	if fe.Namespace != "" && fe.Tag != "" {
		msg := "field '" + fe.Namespace + "' validation failed on tag '" + fe.Tag + "'"
		if fe.Param != "" {
			msg += " with param '" + fe.Param + "'"
		}
		return msg
	}

	return "validation error"
}

// ============================================================================
// 验证错误
// ============================================================================

// ValidationError 验证错误
// 职责：包装所有字段错误
type ValidationError struct {
	// Errors 所有字段错误
	Errors []*FieldError

	// formatter 错误格式化器
	formatter ErrorFormatter
}

// NewValidationError 创建验证错误
func NewValidationError(formatter ErrorFormatter) *ValidationError {
	if formatter == nil {
		formatter = NewDefaultErrorFormatter()
	}
	return &ValidationError{
		Errors:    make([]*FieldError, 0),
		formatter: formatter,
	}
}

// WithError 添加单个错误
func (ve *ValidationError) WithError(err *FieldError) *ValidationError {
	ve.Errors = append(ve.Errors, err)
	return ve
}

// WithErrors 添加多个错误
func (ve *ValidationError) WithErrors(errs []*FieldError) *ValidationError {
	ve.Errors = append(ve.Errors, errs...)
	return ve
}

// WithMessage 设置自定义消息
func (ve *ValidationError) WithMessage(message string) *ValidationError {
	ve.Errors = append(ve.Errors, NewFieldErrorWithMessage(message))
	return ve
}

// Error 实现 error 接口
func (ve *ValidationError) Error() string {
	return ve.formatter.FormatAll(ve.Errors)
}

// HasErrors 是否有错误
func (ve *ValidationError) HasErrors() bool {
	return len(ve.Errors) > 0
}

// Count 错误数量
func (ve *ValidationError) Count() int {
	return len(ve.Errors)
}

// First 获取第一个错误
func (ve *ValidationError) First() *FieldError {
	if len(ve.Errors) > 0 {
		return ve.Errors[0]
	}
	return nil
}

// GetByField 按字段获取错误
func (ve *ValidationError) GetByField(field string) []*FieldError {
	result := make([]*FieldError, 0)
	for _, err := range ve.Errors {
		if err.Field == field || err.Namespace == field {
			result = append(result, err)
		}
	}
	return result
}

// ============================================================================
// 类型信息
// ============================================================================

// TypeInfo 类型信息，缓存类型的验证能力信息
type TypeInfo struct {
	// Type 类型
	Type any

	// IsRuleProvider 是否实现了 RuleProvider
	IsRuleProvider bool

	// IsBusinessValidator 是否实现了 BusinessValidator
	IsBusinessValidator bool

	// IsLifecycleHooks 是否实现了 LifecycleHooks
	IsLifecycleHooks bool

	// Rules 缓存的规则（按场景组织）
	Rules map[Scene]map[string]string

	// RuleProvider 规则提供者实例（避免类型断言）
	RuleProvider RuleProvider

	// BusinessValidator 业务验证器实例（避免类型断言）
	BusinessValidator BusinessValidator

	// LifecycleHooks 生命周期钩子实例（避免类型断言）
	LifecycleHooks LifecycleHooks
}

// NewTypeInfo 创建类型信息
func NewTypeInfo() *TypeInfo {
	return &TypeInfo{
		Rules: make(map[Scene]map[string]string),
	}
}

// HasRules 是否有规则
func (ti *TypeInfo) HasRules() bool {
	return len(ti.Rules) > 0
}

// GetRulesForScene 获取指定场景的规则
func (ti *TypeInfo) GetRulesForScene(scene Scene) map[string]string {
	if ti.Rules == nil {
		return nil
	}
	return ti.Rules[scene]
}
