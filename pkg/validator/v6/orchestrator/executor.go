package orchestrator

import (
	"fmt"
	"sort"

	"katydid-common-account/pkg/validator/v6/core"
)

// StrategyExecutorImpl 策略执行器实现
// 职责：执行验证策略，处理异常恢复
// 设计原则：单一职责 - 只负责执行策略
type StrategyExecutorImpl struct {
	// 可以添加配置项
}

// NewStrategyExecutor 创建策略执行器
func NewStrategyExecutor() core.StrategyExecutor {
	return &StrategyExecutorImpl{}
}

// Execute 执行单个策略
func (e *StrategyExecutorImpl) Execute(strategy core.ValidationStrategy, req *core.ValidationRequest, ctx core.ValidationContext) error {
	if strategy == nil {
		return fmt.Errorf("strategy is nil")
	}

	// 使用 recover 捕获 panic
	defer func() {
		if r := recover(); r != nil {
			// 将 panic 转换为错误
			err := fmt.Errorf("strategy %s panic: %v", strategy.Name(), r)
			ctx.ErrorCollector().Add(core.NewFieldError("", "panic").WithMessage(err.Error()))
		}
	}()

	// 执行策略
	return strategy.Validate(req, ctx)
}

// ExecuteAll 执行所有策略
func (e *StrategyExecutorImpl) ExecuteAll(strategies []core.ValidationStrategy, req *core.ValidationRequest, ctx core.ValidationContext) error {
	if len(strategies) == 0 {
		return nil
	}

	// 按优先级排序
	sortedStrategies := e.sortByPriority(strategies)

	// 依次执行
	for _, strategy := range sortedStrategies {
		if err := e.Execute(strategy, req, ctx); err != nil {
			// 记录错误，但继续执行其他策略
			// 如果需要在第一个错误后停止，可以直接 return err
			ctx.ErrorCollector().Add(core.NewFieldError("", "strategy_error").WithMessage(err.Error()))
		}

		// 检查是否已达到最大错误数
		if ctx.ErrorCollector().Count() >= 100 { // TODO: 从配置获取
			break
		}
	}

	return nil
}

// sortByPriority 按优先级排序
func (e *StrategyExecutorImpl) sortByPriority(strategies []core.ValidationStrategy) []core.ValidationStrategy {
	// 创建副本，避免修改原数组
	sorted := make([]core.ValidationStrategy, len(strategies))
	copy(sorted, strategies)

	// 按优先级排序（数字越小优先级越高）
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority() < sorted[j].Priority()
	})

	return sorted
}
