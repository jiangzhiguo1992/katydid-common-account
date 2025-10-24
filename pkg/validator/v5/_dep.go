package v5

// ============================================================================
// ValidationPipeline 验证管道
// ============================================================================

// ValidationPipeline 验证管道
// 职责：按顺序执行多个验证器
// 设计模式：责任链模式
type ValidationPipeline struct {
	validators []ValidationStrategy
}

// NewValidationPipeline 创建验证管道
func NewValidationPipeline() *ValidationPipeline {
	return &ValidationPipeline{
		validators: make([]ValidationStrategy, 0),
	}
}

// Add 添加验证器
func (p *ValidationPipeline) Add(validator ValidationStrategy) *ValidationPipeline {
	p.validators = append(p.validators, validator)
	return p
}

// Execute 执行管道
func (p *ValidationPipeline) Execute(target any, ctx *ValidationContext) error {
	for _, v := range p.validators {
		if err := v.Validate(target, ctx); err != nil {
			return err
		}
	}
	return nil
}

// ============================================================================
// 监听器实现示例
// ============================================================================

// LoggingListener 日志监听器
// 职责：记录验证过程的日志
type LoggingListener struct {
	logger Logger
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// NewLoggingListener 创建日志监听器
func NewLoggingListener(logger Logger) *LoggingListener {
	return &LoggingListener{logger: logger}
}

// OnValidationStart 验证开始
func (l *LoggingListener) OnValidationStart(ctx *ValidationContext) {
	if l.logger != nil {
		l.logger.Debug("validation started", "scene", ctx.Scene, "target", ctx.Target)
	}
}

// OnValidationEnd 验证结束
func (l *LoggingListener) OnValidationEnd(ctx *ValidationContext) {
	if l.logger != nil {
		if ctx.HasErrors() {
			l.logger.Warn("validation failed", "errors", ctx.ErrorCount())
		} else {
			l.logger.Debug("validation passed")
		}
	}
}

// OnError 发生错误
func (l *LoggingListener) OnError(ctx *ValidationContext, err *FieldError) {
	if l.logger != nil {
		l.logger.Debug("validation error", "field", err.Namespace, "tag", err.Tag)
	}
}

// ============================================================================
// MetricsListener 指标监听器
// ============================================================================

// MetricsListener 指标监听器
// 职责：收集验证指标
type MetricsListener struct {
	validationCount int64
	errorCount      int64
	mu              sync.RWMutex
}

// NewMetricsListener 创建指标监听器
func NewMetricsListener() *MetricsListener {
	return &MetricsListener{}
}

// OnValidationStart 验证开始
func (m *MetricsListener) OnValidationStart(ctx *ValidationContext) {
	m.mu.Lock()
	m.validationCount++
	m.mu.Unlock()
}

// OnValidationEnd 验证结束
func (m *MetricsListener) OnValidationEnd(ctx *ValidationContext) {
	if ctx.HasErrors() {
		m.mu.Lock()
		m.errorCount += int64(ctx.ErrorCount())
		m.mu.Unlock()
	}
}

// OnError 发生错误
func (m *MetricsListener) OnError(ctx *ValidationContext, err *FieldError) {
	// 可以在这里收集更详细的错误指标
}

// GetMetrics 获取指标
func (m *MetricsListener) GetMetrics() (validationCount, errorCount int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.validationCount, m.errorCount
}

// Reset 重置指标
func (m *MetricsListener) Reset() {
	m.mu.Lock()
	m.validationCount = 0
	m.errorCount = 0
	m.mu.Unlock()
}

// ============================================================================
// 对象池 - 内存优化
// ============================================================================

var (
	// validationContextPool 验证上下文对象池
	validationContextPool = sync.Pool{
		New: func() interface{} {
			return &ValidationContext{
				Metadata: make(map[string]any),
			}
		},
	}

	// errorCollectorPool 错误收集器对象池
	errorCollectorPool = sync.Pool{
		New: func() interface{} {
			return NewDefaultErrorCollector()
		},
	}
)

// AcquireValidationContext 从对象池获取验证上下文
func AcquireValidationContext(scene Scene, target any) *ValidationContext {
	ctx := validationContextPool.Get().(*ValidationContext)
	ctx.Scene = scene
	ctx.Target = target
	ctx.Depth = 0

	if ctx.Metadata == nil {
		ctx.Metadata = make(map[string]any)
	} else {
		// 清空 metadata
		for k := range ctx.Metadata {
			delete(ctx.Metadata, k)
		}
	}

	// 获取错误收集器
	ctx.errorCollector = AcquireErrorCollector()

	return ctx
}

// ReleaseValidationContext 归还验证上下文到对象池
func ReleaseValidationContext(ctx *ValidationContext) {
	if ctx == nil {
		return
	}

	// 归还错误收集器
	if ctx.errorCollector != nil {
		ReleaseErrorCollector(ctx.errorCollector)
		ctx.errorCollector = nil
	}

	// 清空字段
	ctx.Context = nil
	ctx.Scene = SceneNone
	ctx.Target = nil
	ctx.Depth = 0

	// 清空 metadata
	if ctx.Metadata != nil {
		for k := range ctx.Metadata {
			delete(ctx.Metadata, k)
		}
	}

	validationContextPool.Put(ctx)
}

// AcquireErrorCollector 从对象池获取错误收集器
func AcquireErrorCollector() ErrorCollector {
	collector := errorCollectorPool.Get().(*DefaultErrorCollector)
	collector.Clear()
	return collector
}

// ReleaseErrorCollector 归还错误收集器到对象池
func ReleaseErrorCollector(collector ErrorCollector) {
	if collector == nil {
		return
	}

	if dc, ok := collector.(*DefaultErrorCollector); ok {
		dc.Clear()
		errorCollectorPool.Put(dc)
	}
}
