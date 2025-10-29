package v6

import (
	"katydid-common-account/pkg/validator/v6/core"
	"katydid-common-account/pkg/validator/v6/engine"
	"katydid-common-account/pkg/validator/v6/errors"
	"katydid-common-account/pkg/validator/v6/infrastructure"
	"katydid-common-account/pkg/validator/v6/orchestration"
	"katydid-common-account/pkg/validator/v6/strategy"
	"sync"
)

// ============================================================================
// 全局门面
// ============================================================================

var (
	defaultValidator core.Validator
	once             sync.Once
)

// Facade 获取默认验证器实例（单例）
// 设计模式：单例模式 + 门面模式
func Facade() core.Validator {
	once.Do(func() {
		defaultValidator = NewDefaultValidator()
	})
	return defaultValidator
}

// Validate 使用默认验证器执行验证
// 便捷方法：简化调用
func Validate(target any, scene core.Scene) core.ValidationError {
	return Facade().Validate(target, scene)
}

// ============================================================================
// 构建器
// ============================================================================

// Builder 验证器构建器
// 设计模式：建造者模式
type Builder struct {
	// 基础设施组件
	cache        core.CacheManager
	inspector    core.TypeInspector
	sceneMatcher core.SceneMatcher
	ruleEngine   core.RuleEngine

	// 编排组件
	orchestrator     core.StrategyOrchestrator
	interceptorChain core.InterceptorChain
	hookExecutor     core.HookExecutor
	listenerNotifier core.ListenerNotifier

	// 策略
	strategies map[core.StrategyType]struct {
		strategy core.ValidationStrategy
		priority int
	}

	// 配置
	errorFormatter core.ErrorFormatter
	maxErrors      int
	maxDepth       int
	executionMode  core.ExecutionMode
}

// NewBuilder 创建构建器
func NewBuilder() *Builder {
	return &Builder{
		strategies:    make(map[core.StrategyType]struct {
			strategy core.ValidationStrategy
			priority int
		}),
		maxErrors:     100,
		maxDepth:      50,
		executionMode: core.ExecutionModeSequential,
	}
}

// WithCache 设置缓存管理器
func (b *Builder) WithCache(cache core.CacheManager) *Builder {
	b.cache = cache
	return b
}

// WithLRUCache 使用 LRU 缓存
func (b *Builder) WithLRUCache(size int) *Builder {
	b.cache = infrastructure.NewLRUCache(size)
	return b
}

// WithNoCache 禁用缓存
func (b *Builder) WithNoCache() *Builder {
	b.cache = infrastructure.NewNoCache()
	return b
}

// WithRuleEngine 设置规则引擎
func (b *Builder) WithRuleEngine(engine core.RuleEngine) *Builder {
	b.ruleEngine = engine
	return b
}

// WithSceneMatcher 设置场景匹配器
func (b *Builder) WithSceneMatcher(matcher core.SceneMatcher) *Builder {
	b.sceneMatcher = matcher
	return b
}

// WithRuleStrategy 添加规则验证策略
func (b *Builder) WithRuleStrategy(priority int) *Builder {
	b.strategies[core.StrategyTypeRule] = struct {
		strategy core.ValidationStrategy
		priority int
	}{priority: priority}
	return b
}

// WithBusinessStrategy 添加业务验证策略
func (b *Builder) WithBusinessStrategy(priority int) *Builder {
	b.strategies[core.StrategyTypeBusiness] = struct {
		strategy core.ValidationStrategy
		priority int
	}{priority: priority}
	return b
}

// WithInterceptor 添加拦截器
func (b *Builder) WithInterceptor(interceptor core.Interceptor) *Builder {
	if b.interceptorChain == nil {
		b.interceptorChain = orchestration.NewInterceptorChain()
	}
	b.interceptorChain.Add(interceptor)
	return b
}

// WithListener 添加监听器
func (b *Builder) WithListener(listener core.ValidationListener) *Builder {
	if b.listenerNotifier == nil {
		b.listenerNotifier = orchestration.NewListenerNotifier()
	}
	b.listenerNotifier.Register(listener)
	return b
}

// WithErrorFormatter 设置错误格式化器
func (b *Builder) WithErrorFormatter(formatter core.ErrorFormatter) *Builder {
	b.errorFormatter = formatter
	return b
}

// WithMaxErrors 设置最大错误数
func (b *Builder) WithMaxErrors(maxErrors int) *Builder {
	b.maxErrors = maxErrors
	return b
}

// WithMaxDepth 设置最大深度
func (b *Builder) WithMaxDepth(maxDepth int) *Builder {
	b.maxDepth = maxDepth
	return b
}

// WithExecutionMode 设置策略执行模式
func (b *Builder) WithExecutionMode(mode core.ExecutionMode) *Builder {
	b.executionMode = mode
	return b
}

// Build 构建验证器
func (b *Builder) Build() core.Validator {
	// 初始化基础设施组件
	b.initInfrastructure()

	// 初始化编排组件
	b.initOrchestration()

	// 注册策略
	b.registerStrategies()

	// 创建引擎
	return engine.NewValidatorEngine(
		b.orchestrator,
		engine.WithInterceptorChain(b.interceptorChain),
		engine.WithHookExecutor(b.hookExecutor),
		engine.WithListenerNotifier(b.listenerNotifier),
		engine.WithErrorFormatter(b.errorFormatter),
		engine.WithMaxErrors(b.maxErrors),
		engine.WithMaxDepth(b.maxDepth),
	)
}

// initInfrastructure 初始化基础设施组件
func (b *Builder) initInfrastructure() {
	// 缓存管理器
	if b.cache == nil {
		b.cache = infrastructure.NewSimpleCache()
	}

	// 类型检查器
	if b.inspector == nil {
		b.inspector = infrastructure.NewTypeInspector(b.cache)
	}

	// 场景匹配器
	if b.sceneMatcher == nil {
		b.sceneMatcher = infrastructure.NewBitSceneMatcher()
	}

	// 规则引擎
	if b.ruleEngine == nil {
		b.ruleEngine = infrastructure.NewPlaygroundRuleEngine()
	}
}

// initOrchestration 初始化编排组件
func (b *Builder) initOrchestration() {
	// 策略编排器
	if b.orchestrator == nil {
		b.orchestrator = orchestration.NewStrategyOrchestrator()
		b.orchestrator.SetExecutionMode(b.executionMode)
	}

	// 钩子执行器
	if b.hookExecutor == nil {
		b.hookExecutor = orchestration.NewHookExecutor(b.inspector)
	}
}

// registerStrategies 注册策略
func (b *Builder) registerStrategies() {
	for strategyType, entry := range b.strategies {
		var s core.ValidationStrategy

		switch strategyType {
		case core.StrategyTypeRule:
			s = strategy.NewRuleStrategy(b.ruleEngine, b.inspector, b.sceneMatcher)
		case core.StrategyTypeBusiness:
			s = strategy.NewBusinessStrategy(b.inspector)
		}

		if s != nil {
			b.orchestrator.Register(s, entry.priority)
		}
	}
}

// ============================================================================
// 便捷工厂方法
// ============================================================================

// NewDefaultValidator 创建默认验证器
func NewDefaultValidator() core.Validator {
	return NewBuilder().
		WithRuleStrategy(10).
		WithBusinessStrategy(20).
		WithErrorFormatter(errors.NewDefaultFormatter()).
		Build()
}

// NewFastValidator 创建快速验证器（禁用缓存）
func NewFastValidator() core.Validator {
	return NewBuilder().
		WithNoCache().
		WithRuleStrategy(10).
		WithBusinessStrategy(20).
		Build()
}

// NewEnterpriseValidator 创建企业级验证器（完整功能）
func NewEnterpriseValidator() core.Validator {
	return NewBuilder().
		WithLRUCache(1000).
		WithRuleStrategy(10).
		WithBusinessStrategy(20).
		WithErrorFormatter(errors.NewDetailedFormatter()).
		WithMaxErrors(200).
		WithMaxDepth(100).
		Build()
}
