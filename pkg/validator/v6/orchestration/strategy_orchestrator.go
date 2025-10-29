package orchestration

import (
	"katydid-common-account/pkg/validator/v6/core"
	"sort"
	"sync"
)

// strategyEntry 策略条目
type strategyEntry struct {
	strategy core.IValidationStrategy
	priority int // 数字越小优先级越高
}

// strategyOrchestrator 策略编排器实现
// 职责：管理和编排验证策略的执行顺序
// 设计原则：责任链模式 + 策略模式
type strategyOrchestrator struct {
	strategies    []strategyEntry
	executionMode core.ExecutionMode
	//mu            sync.RWMutex // 一般就初始化，不必枷锁
}

// NewStrategyOrchestrator 创建策略编排器
func NewStrategyOrchestrator() core.IStrategyOrchestrator {
	return &strategyOrchestrator{
		strategies:    make([]strategyEntry, 0),
		executionMode: core.ExecutionModeSequential, // 默认串行执行
	}
}

// Register 注册策略
func (o *strategyOrchestrator) Register(strategy core.IValidationStrategy, priority int) {
	//o.mu.Lock()
	//defer o.mu.Unlock()

	// 添加策略
	o.strategies = append(o.strategies, strategyEntry{
		strategy: strategy,
		priority: priority,
	})

	// 按优先级排序
	sort.Slice(o.strategies, func(i, j int) bool {
		return o.strategies[i].priority < o.strategies[j].priority
	})
}

// Unregister 注销策略
func (o *strategyOrchestrator) Unregister(strategyType core.StrategyType) {
	//o.mu.Lock()
	//defer o.mu.Unlock()

	// 过滤掉指定类型的策略
	filtered := make([]strategyEntry, 0, len(o.strategies))
	for _, entry := range o.strategies {
		if entry.strategy.Type() != strategyType {
			filtered = append(filtered, entry)
		}
	}
	o.strategies = filtered
}

// GetStrategies 获取所有策略
func (o *strategyOrchestrator) GetStrategies() []core.IValidationStrategy {
	//o.mu.RLock()
	//defer o.mu.RUnlock()

	strategies := make([]core.IValidationStrategy, len(o.strategies))
	for i, entry := range o.strategies {
		strategies[i] = entry.strategy
	}
	return strategies
}

// Execute 执行所有策略
func (o *strategyOrchestrator) Execute(target any, ctx core.IContext, collector core.IErrorCollector) error {
	//o.mu.RLock()
	//defer o.mu.RUnlock()

	if o.executionMode == core.ExecutionModeParallel {
		return o.executeParallel(target, ctx, collector)
	}
	return o.executeSequential(target, ctx, collector)
}

// executeSequential 串行执行策略
func (o *strategyOrchestrator) executeSequential(target any, ctx core.IContext, collector core.IErrorCollector) error {
	for _, entry := range o.strategies {
		// 检查是否已达到最大错误数
		if collector.Count() >= collector.MaxErrors() {
			break
		}

		// 执行策略
		if err := entry.strategy.Validate(target, ctx, collector); err != nil {
			// 策略执行出错，中断当前执行
			return err
		}
	}
	return nil
}

// executeParallel 并行执行策略
func (o *strategyOrchestrator) executeParallel(target any, ctx core.IContext, collector core.IErrorCollector) error {
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, entry := range o.strategies {
		wg.Add(1)

		// 并行执行每个策略
		go func(s core.IValidationStrategy) {
			defer wg.Done()

			// 检查是否已达到最大错误数
			mu.Lock()
			if collector.Count() >= collector.MaxErrors() {
				mu.Unlock()
				return
			}
			mu.Unlock()

			// 执行策略 TODO:GG err要中断执行
			s.Validate(target, ctx, collector)
		}(entry.strategy)
	}

	wg.Wait()
	return nil
}

// SetExecutionMode 设置执行模式
func (o *strategyOrchestrator) SetExecutionMode(mode core.ExecutionMode) {
	//o.mu.Lock()
	//defer o.mu.Unlock()
	o.executionMode = mode
}
