package v2

// ============================================================================
// 核心接口定义 - 基于接口隔离原则（ISP）
// ============================================================================

// Validator 验证器核心接口 - 单一职责：只负责执行验证
// 依赖倒置原则（DIP）：依赖抽象而非具体实现
type Validator interface {
	// Validate 验证对象
	// 参数：
	//   - obj: 待验证的对象
	//   - scene: 验证场景
	// 返回：验证结果
	Validate(obj any, scene Scene) Result

	// ValidateFields 只验证指定字段
	// 参数：
	//   - obj: 待验证的对象
	//   - scene: 验证场景
	//   - fields: 需要验证的字段名列表
	// 返回：验证结果
	ValidateFields(obj any, scene Scene, fields ...string) Result

	// ValidateExcept 验证除指定字段外的所有字段
	// 参数：
	//   - obj: 待验证的对象
	//   - scene: 验证场景
	//   - excludeFields: 需要排除的字段名列表
	// 返回：验证结果
	ValidateExcept(obj any, scene Scene, excludeFields ...string) Result

	// RegisterAlias 注册验证标签别名
	// 参数：
	//   - alias: 别名标签名
	//   - tags: 实际的验证规则字符串
	RegisterAlias(alias, tags string)
}

// RuleValidator 规则提供者接口 - 提供字段验证规则
// 单一职责原则（SRP）：只负责提供验证规则，不执行验证
type RuleValidator interface {
	// ValidateRules 提供场景化的验证规则
	// 返回格式：map[场景][字段名]规则字符串
	ValidateRules() map[Scene]FieldRules
}

// CustomValidator 自定义验证器接口 - 复杂业务逻辑验证
// 单一职责原则（SRP）：只负责复杂的业务逻辑验证
type CustomValidator interface {
	// ValidateCustom 执行自定义验证逻辑
	// 参数：
	//   - scene: 当前验证场景
	//   - reporter: 错误报告器
	ValidateCustom(scene Scene, reporter ErrorReporter)
}

// ErrorReporter 错误报告器接口 - 用于收集验证错误
// 接口隔离原则（ISP）：只提供必要的报告方法
type ErrorReporter interface {
	// Report 报告一个验证错误
	Report(namespace, tag, param string)

	// ReportMsg 报告一个带自定义消息的验证错误
	ReportMsg(namespace, tag, param, message string)

	// ReportWithValue 报告一个带值的验证错误
	ReportWithValue(namespace, tag, param string, value any)

	// ReportDetail 报告一个详细的验证错误（包含值和消息）
	ReportDetail(namespace, tag, param string, value any, message string)
}

// ValidationStrategy 验证策略接口 - 策略模式
// 开放封闭原则（OCP）：对扩展开放，对修改封闭
type ValidationStrategy interface {
	// Execute 执行验证策略
	// 参数：
	//   - obj: 待验证的对象
	//   - scene: 验证场景
	//   - collector: 错误收集器
	// 返回：是否应继续执行后续策略
	Execute(obj any, scene Scene, collector ErrorCollector) bool
}

// ErrorCollector 错误收集器接口 - 收集和管理验证错误
// 接口隔离原则（ISP）：分离错误收集和查询功能
type ErrorCollector interface {
	ErrorReporter

	// Add 添加一个错误
	Add(err *FieldError)

	// AddAll 批量添加错误
	AddAll(errs []*FieldError)

	// HasErrors 是否存在错误
	HasErrors() bool

	// GetErrors 获取所有错误
	GetErrors() []*FieldError

	// Clear 清空所有错误
	Clear()
}

// Result 验证结果接口 - 查询验证结果
// 接口隔离原则（ISP）：只提供结果查询功能
type Result interface {
	// IsValid 验证是否通过
	IsValid() bool

	// Errors 获取所有错误
	Errors() []*FieldError

	// FirstError 获取第一个错误
	FirstError() *FieldError

	// ErrorsByField 获取指定字段的错误
	ErrorsByField(field string) []*FieldError

	// ErrorsByTag 获取指定标签的错误
	ErrorsByTag(tag string) []*FieldError

	// Error 实现 error 接口
	Error() string
}

// TypeCache 类型缓存接口 - 缓存类型信息
// 单一职责原则（SRP）：只负责类型信息的缓存
type TypeCache interface {
	// Get 获取类型信息
	Get(obj any) *TypeInfo

	// Clear 清空缓存
	Clear()
}

// RegistryManager 注册管理器接口 - 管理已注册的类型
// 单一职责原则（SRP）：只负责注册状态管理
type RegistryManager interface {
	// IsRegistered 检查类型是否已注册
	IsRegistered(obj any) bool

	// MarkRegistered 标记类型已注册
	MarkRegistered(obj any)

	// Clear 清空注册记录
	Clear()
}

// MapValidatorConfig Map 验证器配置接口
// 单一职责原则（SRP）：只负责 Map 验证配置
type MapValidatorConfig interface {
	// GetNamespace 获取命名空间
	GetNamespace() string

	// GetRequiredKeys 获取必填键列表
	GetRequiredKeys() []string

	// GetAllowedKeys 获取允许的键列表
	GetAllowedKeys() []string

	// GetKeyValidator 获取指定键的验证器
	GetKeyValidator(key string) func(value any) error

	// Validate 验证 map 数据
	Validate(data map[string]any) []*FieldError
}

// ValidatorBuilder 验证器构建器接口 - 建造者模式
// 开放封闭原则（OCP）：通过构建器扩展配置，而不修改验证器
type ValidatorBuilder interface {
	// WithStrategy 添加验证策略
	WithStrategy(strategy ValidationStrategy) ValidatorBuilder

	// WithTypeCache 设置类型缓存
	WithTypeCache(cache TypeCache) ValidatorBuilder

	// WithRegistry 设置注册管理器
	WithRegistry(registry RegistryManager) ValidatorBuilder

	// Build 构建验证器
	Build() Validator
}
