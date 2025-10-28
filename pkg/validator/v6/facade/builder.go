package facade

import (
	"katydid-common-account/pkg/validator/v6/core"
	"katydid-common-account/pkg/validator/v6/matcher"
	"katydid-common-account/pkg/validator/v6/orchestrator"
	"katydid-common-account/pkg/validator/v6/registry"
	"katydid-common-account/pkg/validator/v6/strategy"
)

// Builder 验证器构建器
// 职责：构建配置复杂的验证器
// 设计模式：建造者模式
type Builder struct {
	strategies      []core.ValidationStrategy
	plugins         []core.Plugin
	listeners       []core.ValidationListener
	eventDispatcher core.EventDispatcher
	maxErrors       int
	maxDepth        int
}

// NewBuilder 创建构建器
func NewBuilder() *Builder {
	return &Builder{
		strategies: make([]core.ValidationStrategy, 0),
		plugins:    make([]core.Plugin, 0),
		listeners:  make([]core.ValidationListener, 0),
		maxErrors:  100,
		maxDepth:   100,
	}
}

// WithStrategies 添加验证策略
func (b *Builder) WithStrategies(strategies ...core.ValidationStrategy) *Builder {
	b.strategies = append(b.strategies, strategies...)
	return b
}

// WithPlugins 添加插件
func (b *Builder) WithPlugins(plugins ...core.Plugin) *Builder {
	b.plugins = append(b.plugins, plugins...)
	return b
}

// WithListeners 添加监听器
func (b *Builder) WithListeners(listeners ...core.ValidationListener) *Builder {
	b.listeners = append(b.listeners, listeners...)
	return b
}

// WithEventDispatcher 设置事件分发器
func (b *Builder) WithEventDispatcher(dispatcher core.EventDispatcher) *Builder {
	b.eventDispatcher = dispatcher
	return b
}

// WithMaxErrors 设置最大错误数
func (b *Builder) WithMaxErrors(max int) *Builder {
	b.maxErrors = max
	return b
}

// WithMaxDepth 设置最大深度
func (b *Builder) WithMaxDepth(max int) *Builder {
	b.maxDepth = max
	return b
}

// Build 构建验证器
func (b *Builder) Build() core.Validator {
	// 创建事件分发器（如果没有设置）
	if b.eventDispatcher == nil {
		b.eventDispatcher = orchestrator.NewEventDispatcher()
	}

	// 订阅监听器
	for _, listener := range b.listeners {
		b.eventDispatcher.Subscribe(listener)
	}

	// 创建编排器
	orch := orchestrator.NewOrchestrator(
		orchestrator.WithStrategies(b.strategies...),
		orchestrator.WithPlugins(b.plugins...),
		orchestrator.WithEventDispatcher(b.eventDispatcher),
		orchestrator.WithMaxErrors(b.maxErrors),
		orchestrator.WithMaxDepth(b.maxDepth),
	)

	// 创建门面
	return NewValidatorFacade(orch)
}

// BuildDefault 构建默认验证器
func (b *Builder) BuildDefault() core.Validator {
	// 创建依赖组件
	sceneMatcher := matcher.NewSceneMatcher()
	typeRegistry := registry.NewTypeRegistry()

	// 添加默认策略
	b.WithStrategies(
		strategy.NewRuleStrategy(sceneMatcher, typeRegistry),
		strategy.NewBusinessStrategy(),
	)

	return b.Build()
}
