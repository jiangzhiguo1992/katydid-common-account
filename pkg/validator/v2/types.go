package v2

// ValidateScene 验证场景类型
// 使用 int64 支持位运算，实现场景组合
type ValidateScene int64

// 预定义验证场景
const (
	SceneNone   ValidateScene = 0      // 无场景
	SceneCreate ValidateScene = 1 << 0 // 创建场景 (1)
	SceneUpdate ValidateScene = 1 << 1 // 更新场景 (2)
	SceneDelete ValidateScene = 1 << 2 // 删除场景 (4)
	SceneQuery  ValidateScene = 1 << 3 // 查询场景 (8)
	SceneAll    ValidateScene = -1     // 所有场景
)

// ValidationError 验证错误接口
// 设计原则：接口隔离 - 只暴露必要的方法
type ValidationError interface {
	// Field 返回字段名
	Field() string

	// Tag 返回验证标签
	Tag() string

	// Message 返回错误消息
	Message() string

	// Error 实现 error 接口
	Error() string
}

// FieldError 字段验证错误实现
type FieldError struct {
	FieldName    string `json:"field"`
	TagName      string `json:"tag"`
	ErrorMessage string `json:"message"`
	Value        any    `json:"value,omitempty"`
}

// Field 返回字段名
func (e *FieldError) Field() string {
	return e.FieldName
}

// Tag 返回验证标签
func (e *FieldError) Tag() string {
	return e.TagName
}

// Message 返回错误消息
func (e *FieldError) Message() string {
	if e.ErrorMessage != "" {
		return e.ErrorMessage
	}
	return e.FieldName + " validation failed on " + e.TagName
}

// Error 实现 error 接口
func (e *FieldError) Error() string {
	return e.Message()
}

// NewFieldError 创建字段错误
func NewFieldError(field, tag, message string) *FieldError {
	return &FieldError{
		FieldName:    field,
		TagName:      tag,
		ErrorMessage: message,
	}
}

// WithValue 设置字段值（链式调用）
func (e *FieldError) WithValue(value any) *FieldError {
	e.Value = value
	return e
}
