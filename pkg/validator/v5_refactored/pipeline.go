package v5_refactored

import (
	"fmt"
	"sort"
	"sync"
)

// ============================================================================
// 默认管道执行器
// ============================================================================

// DefaultPipelineExecutor 默认管道执行器
// 职责：顺序执行验证策略
// 设计原则：单一职责 - 只负责策略编排和执行
type DefaultPipelineExecutor struct {
	// strategies 验证策略列表
	strategies []ValidationStrategy

	// mu 保护策略列表
	mu sync.RWMutex
}

// NewDefaultPipelineExecutor 创建默认管道执行器
func NewDefaultPipelineExecutor() *DefaultPipelineExecutor {
	return &DefaultPipelineExecutor{
		strategies: make([]ValidationStrategy, 0),
	}
}

// Execute 执行验证管道
func (e *DefaultPipelineExecutor) Execute(target any, ctx *ValidationContext, collector ErrorCollector) error {
	if target == nil {
		return fmt.Errorf("target is nil")
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// 按优先级执行所有策略
	for _, strategy := range e.strategies {
		// 检查是否已达到最大错误数
		if collector.IsFull() {
			break
		}

		// 执行策略（带异常恢复）
		if err := e.executeWithRecovery(strategy, target, ctx, collector); err != nil {
			// 策略执行失败，添加错误
			collector.Add(NewFieldErrorWithMessage(fmt.Sprintf("strategy %s failed: %v", strategy.Name(), err)))
		}
	}

	return nil
}

// AddStrategy 添加策略
func (e *DefaultPipelineExecutor) AddStrategy(strategy ValidationStrategy) {
	if strategy == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.strategies = append(e.strategies, strategy)

	// 按优先级排序
	sort.Slice(e.strategies, func(i, j int) bool {
		return e.strategies[i].Priority() < e.strategies[j].Priority()
	})
}

// RemoveStrategy 移除策略
func (e *DefaultPipelineExecutor) RemoveStrategy(strategyType StrategyType) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 过滤掉指定类型的策略
	filtered := make([]ValidationStrategy, 0, len(e.strategies))
	for _, s := range e.strategies {
		if s.Type() != strategyType {
			filtered = append(filtered, s)
		}
	}

	e.strategies = filtered
}

// GetStrategies 获取所有策略
func (e *DefaultPipelineExecutor) GetStrategies() []ValidationStrategy {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// 返回副本，避免外部修改
	result := make([]ValidationStrategy, len(e.strategies))
	copy(result, e.strategies)
	return result
}

// executeWithRecovery 执行策略（带异常恢复）
func (e *DefaultPipelineExecutor) executeWithRecovery(
	strategy ValidationStrategy,
	target any,
	ctx *ValidationContext,
	collector ErrorCollector,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// 捕获 panic 并转换为 error
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("panic in strategy %s: %v", strategy.Name(), r)
			}
		}
	}()

	return strategy.Validate(target, ctx, collector)
}

// ============================================================================
// 并发管道执行器
// ============================================================================

// ConcurrentPipelineExecutor 并发管道执行器
// 职责：并发执行独立的验证策略
// 设计原则：适用于策略之间无依赖的场景
type ConcurrentPipelineExecutor struct {
	// strategies 验证策略列表
	strategies []ValidationStrategy

	// workers 并发工作数
	workers int

	// mu 保护策略列表
	mu sync.RWMutex
}

// NewConcurrentPipelineExecutor 创建并发管道执行器
func NewConcurrentPipelineExecutor(workers int) *ConcurrentPipelineExecutor {
	if workers <= 0 {
		workers = 4 // 默认 4 个并发
	}

	return &ConcurrentPipelineExecutor{
		strategies: make([]ValidationStrategy, 0),
		workers:    workers,
	}
}

// Execute 执行验证管道
func (e *ConcurrentPipelineExecutor) Execute(target any, ctx *ValidationContext, collector ErrorCollector) error {
	if target == nil {
		return fmt.Errorf("target is nil")
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// 使用并发安全的错误收集器
	safeCollector := NewConcurrentErrorCollector(collector.(*DefaultErrorCollector).maxErrors)

	// 创建工作通道
	taskChan := make(chan ValidationStrategy, len(e.strategies))
	var wg sync.WaitGroup

	// 启动工作协程
	for i := 0; i < e.workers && i < len(e.strategies); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for strategy := range taskChan {
				// 执行策略（带异常恢复）
				if err := e.executeWithRecovery(strategy, target, ctx, safeCollector); err != nil {
					safeCollector.Add(NewFieldErrorWithMessage(fmt.Sprintf("strategy %s failed: %v", strategy.Name(), err)))
				}

				// 检查是否已满
				if safeCollector.IsFull() {
					break
				}
			}
		}()
	}

	// 分发任务
	for _, strategy := range e.strategies {
		taskChan <- strategy
	}
	close(taskChan)

	// 等待所有任务完成
	wg.Wait()

	// 将结果复制到原始收集器
	for _, err := range safeCollector.GetAll() {
		if !collector.Add(err) {
			break
		}
	}

	return nil
}

// AddStrategy 添加策略
func (e *ConcurrentPipelineExecutor) AddStrategy(strategy ValidationStrategy) {
	if strategy == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.strategies = append(e.strategies, strategy)

	// 按优先级排序
	sort.Slice(e.strategies, func(i, j int) bool {
		return e.strategies[i].Priority() < e.strategies[j].Priority()
	})
}

// RemoveStrategy 移除策略
func (e *ConcurrentPipelineExecutor) RemoveStrategy(strategyType StrategyType) {
	e.mu.Lock()
	defer e.mu.Unlock()

	filtered := make([]ValidationStrategy, 0, len(e.strategies))
	for _, s := range e.strategies {
		if s.Type() != strategyType {
			filtered = append(filtered, s)
		}
	}

	e.strategies = filtered
}

// GetStrategies 获取所有策略
func (e *ConcurrentPipelineExecutor) GetStrategies() []ValidationStrategy {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]ValidationStrategy, len(e.strategies))
	copy(result, e.strategies)
	return result
}

// executeWithRecovery 执行策略（带异常恢复）
func (e *ConcurrentPipelineExecutor) executeWithRecovery(
	strategy ValidationStrategy,
	target any,
	ctx *ValidationContext,
	collector ErrorCollector,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("panic in strategy %s: %v", strategy.Name(), r)
			}
		}
	}()

	return strategy.Validate(target, ctx, collector)
}
