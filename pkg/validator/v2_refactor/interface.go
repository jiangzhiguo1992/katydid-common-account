package v2

// ============================================================================
// 核心接口定义 - 保持与 v1 一致
// ============================================================================

// RuleProvider 规则提供者接口 - 提供场景化验证规则
// 对应 v1 的 RuleValidator 接口
type RuleProvider interface {
	// RuleValidation 返回场景化的验证规则映射
	// 返回格式：map[场景标识][字段名]规则字符串
	RuleValidation() map[Scene]map[string]string
}

// CustomValidator 自定义验证器接口 - 执行复杂业务逻辑验证
// 对应 v1 的 CustomValidator 接口
type CustomValidator interface {
	// CustomValidation 执行自定义验证逻辑
	// scene: 当前验证场景
	// report: 错误报告函数
	CustomValidation(scene Scene, report FuncReportError)
}

// FuncReportError 错误报告函数类型
// 与 v1 保持一致的签名
type FuncReportError func(namespace, tag, param string)

// ErrorCollector 错误收集器接口 - 内部使用
type ErrorCollector interface {
	// Report 报告错误
	Report(namespace, tag, param string)
	// AddError 添加已构造的错误
	AddError(err *FieldError)
	// HasErrors 是否有错误
	HasErrors() bool
	// GetErrors 获取所有错误
	GetErrors() ValidationErrors
	// Clear 清空错误
	Clear()
}
