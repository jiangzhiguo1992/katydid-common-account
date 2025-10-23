package v2

// ============================================================================
// 核心接口定义 - 遵循接口隔离原则（ISP）
// ============================================================================

// RuleProvider 规则提供者接口
// 职责：提供字段级别的验证规则
type RuleProvider interface {
	// GetRules 获取验证规则
	// 返回：场景 -> 字段名 -> 规则字符串
	GetRules() map[ValidateScene]map[string]string
}

// BusinessValidator 业务验证器接口
// 职责：执行复杂的业务逻辑验证
type BusinessValidator interface {
	// ValidateBusiness 执行业务验证
	// 参数：scene - 验证场景
	// 返回：验证错误列表
	ValidateBusiness(scene ValidateScene) []ValidationError
}

// ErrorCollector 错误收集器接口
// 职责：收集和管理验证错误
type ErrorCollector interface {
	// Add 添加单个错误
	Add(err ValidationError)

	// AddAll 批量添加错误
	AddAll(errs []ValidationError)

	// HasErrors 检查是否有错误
	HasErrors() bool

	// GetAll 获取所有错误
	GetAll() []ValidationError

	// Count 获取错误数量
	Count() int

	// Clear 清空错误
	Clear()
}

// ValidationStrategy 验证策略接口
// 职责：执行特定类型的验证
// 设计模式：策略模式
type ValidationStrategy interface {
	// Execute 执行验证策略
	// 参数：
	//   - obj: 待验证对象
	//   - scene: 验证场景
	//   - collector: 错误收集器
	Execute(obj any, scene ValidateScene, collector ErrorCollector)
}

// TypeInfoCache 类型信息缓存接口
// 职责：缓存类型的元数据信息
// 设计原则：依赖倒置 - 依赖抽象而非具体实现
type TypeInfoCache interface {
	// Get 获取类型信息
	Get(obj any) *TypeMetadata

	// Clear 清除缓存
	Clear()
}

// ============================================================================
// 数据结构
// ============================================================================

// TypeMetadata 类型元数据
type TypeMetadata struct {
	// IsRuleProvider 是否实现了 RuleProvider 接口
	IsRuleProvider bool

	// IsBusinessValidator 是否实现了 BusinessValidator 接口
	IsBusinessValidator bool

	// Rules 缓存的验证规则
	Rules map[ValidateScene]map[string]string
}
