package v3

// FieldError 单个字段的验证错误信息
// 设计原则：单一职责 - 只负责描述字段验证错误的详细信息
type FieldError struct {
	// Namespace 字段的完整命名空间路径（如 User.Profile.Email）
	// 用于嵌套结构体的错误定位，支持复杂对象的精确错误追踪
	Namespace string `json:"namespace"`

	// 字段名
	Field string `json:"field"`

	// Tag 验证标签，描述验证规则类型（如 required, email, min, max 等）
	Tag string `json:"tag"`

	// Param 验证参数，提供验证规则的具体配置值
	// 例如：min=3 中的 "3"，len=11 中的 "11"
	Param string `json:"param"`

	// Value 字段的实际值（用于 sl.ReportError 的 value 参数）
	// 用于调试和详细错误信息，可能包含敏感信息，谨慎使用
	Value any `json:"value,omitempty"`

	// Message 用户友好的错误消息（可选，用于直接显示给终端用户）
	// 支持国际化，建议使用本地化后的错误消息
	Message string `json:"message,omitempty"`
}
