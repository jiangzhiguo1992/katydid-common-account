# Validator v5 重构版 - 架构设计文档

## 📋 目录

- [设计原则](#设计原则)
- [架构概览](#架构概览)
- [核心模块](#核心模块)
- [设计模式](#设计模式)
- [相比 v5 的改进](#相比-v5-的改进)

---

## 🎯 设计原则

### SOLID 原则应用

#### 1. 单一职责原则 (SRP)

**v5 问题**：
- `ValidatorEngine` 承担了太多职责：策略编排、监听器管理、钩子执行、错误收集
- `TypeRegistry` 同时负责类型缓存和字段访问器构建

**v5_refactored 解决方案**：

```
ValidatorEngine (协调器)
├── PipelineExecutor (管道执行器) - 负责策略编排和执行
├── EventBus (事件总线) - 负责事件发布和监听器管理
├── HookManager (钩子管理器) - 负责生命周期钩子
├── ErrorCollector (错误收集器) - 负责错误收集和聚合
└── TypeRegistry (类型注册表) - 只负责类型信息缓存
```

**职责分离**：
- `ValidatorEngine`：只负责组件协调和依赖注入
- `PipelineExecutor`：只负责验证策略的执行流程
- `EventBus`：只负责事件的发布订阅
- `HookManager`：只负责生命周期管理
- `ErrorCollector`：只负责错误收集和管理

#### 2. 开放封闭原则 (OCP)

**扩展点设计**：

```go
// 策略扩展
type ValidationStrategy interface {
    Type() StrategyType
    Priority() int8
    Validate(target any, ctx *ValidationContext) error
}

// 事件监听扩展
type EventListener interface {
    OnEvent(event Event)
    EventTypes() []EventType
}

// 错误格式化扩展
type ErrorFormatter interface {
    Format(err *FieldError) string
    FormatAll(errs []*FieldError) string
}

// 场景匹配扩展
type SceneMatcher interface {
    Match(current, target Scene) bool
    MatchRules(current Scene, rules map[Scene]map[string]string) map[string]string
}

// 类型缓存扩展
type TypeCache interface {
    Get(typ reflect.Type) (*TypeInfo, bool)
    Set(typ reflect.Type, info *TypeInfo)
    Clear()
}
```

#### 3. 里氏替换原则 (LSP)

所有实现必须可以无缝替换：

```go
// 所有策略实现可互相替换
var _ ValidationStrategy = (*RuleStrategy)(nil)
var _ ValidationStrategy = (*BusinessStrategy)(nil)
var _ ValidationStrategy = (*NestedStrategy)(nil)

// 所有错误收集器实现可互相替换
var _ ErrorCollector = (*DefaultErrorCollector)(nil)
var _ ErrorCollector = (*ConcurrentErrorCollector)(nil)

// 所有事件总线实现可互相替换
var _ EventBus = (*SyncEventBus)(nil)
var _ EventBus = (*AsyncEventBus)(nil)
```

#### 4. 接口隔离原则 (ISP)

**细粒度接口设计**：

```go
// v5 问题：Registry 接口过于庞大
type Registry interface {
    Register(target any) *TypeInfo
    Get(target any) (*TypeInfo, bool)
    Clear()
    Stats() (count int)
}

// v5_refactored：拆分为细粒度接口
type TypeInfoReader interface {
    Get(typ reflect.Type) (*TypeInfo, bool)
}

type TypeInfoWriter interface {
    Set(typ reflect.Type, info *TypeInfo)
}

type TypeInfoCache interface {
    TypeInfoReader
    TypeInfoWriter
    Clear()
}

type TypeAnalyzer interface {
    Analyze(target any) *TypeInfo
}

type TypeRegistry interface {
    TypeInfoCache
    TypeAnalyzer
}
```

#### 5. 依赖倒置原则 (DIP)

**完全面向接口编程**：

```go
type ValidatorEngine struct {
    // 所有依赖都是接口
    pipeline      PipelineExecutor    // 而非具体实现
    eventBus      EventBus            // 而非具体实现
    hookManager   HookManager         // 而非具体实现
    registry      TypeRegistry        // 而非具体实现
    errorCollector ErrorCollector     // 而非具体实现
}

// 构造函数注入
func NewValidatorEngine(
    pipeline PipelineExecutor,
    eventBus EventBus,
    hookManager HookManager,
    registry TypeRegistry,
    errorCollector ErrorCollector,
) *ValidatorEngine {
    return &ValidatorEngine{
        pipeline:       pipeline,
        eventBus:       eventBus,
        hookManager:    hookManager,
        registry:       registry,
        errorCollector: errorCollector,
    }
}
```

---

## 🏗️ 架构概览

### 分层架构

```
┌─────────────────────────────────────────────────────────┐
│                   应用层 (Application)                   │
│  - 业务模型 (User, Product, Order...)                   │
│  - 实现验证接口 (RuleProvider, BusinessValidator...)     │
└─────────────────────────────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────┐
│                   门面层 (Facade)                        │
│  - ValidatorEngine (验证引擎)                            │
│  - ValidatorFactory (验证器工厂)                         │
│  - Global API (全局便捷函数)                             │
└─────────────────────────────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────┐
│                   编排层 (Orchestration)                 │
│  - PipelineExecutor (管道执行器)                         │
│  - EventBus (事件总线)                                   │
│  - HookManager (钩子管理器)                              │
└─────────────────────────────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────┐
│                   策略层 (Strategy)                      │
│  - RuleStrategy (规则策略)                               │
│  - BusinessStrategy (业务策略)                           │
│  - NestedStrategy (嵌套策略)                             │
└─────────────────────────────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────┐
│                   基础设施层 (Infrastructure)             │
│  - TypeRegistry (类型注册表)                             │
│  - ErrorCollector (错误收集器)                           │
│  - SceneMatcher (场景匹配器)                             │
│  - ErrorFormatter (错误格式化器)                         │
│  - ValidationContext (验证上下文)                        │
└─────────────────────────────────────────────────────────┘
```

### 组件交互图

```
                    ┌──────────────────┐
                    │ ValidatorEngine  │
                    │   (协调者)        │
                    └────────┬─────────┘
                             │
          ┌──────────────────┼──────────────────┐
          │                  │                  │
          ▼                  ▼                  ▼
  ┌───────────────┐  ┌──────────────┐  ┌──────────────┐
  │ PipelineExec  │  │  EventBus    │  │ HookManager  │
  │   (执行)      │  │   (事件)      │  │   (钩子)      │
  └───────┬───────┘  └──────┬───────┘  └──────┬───────┘
          │                  │                  │
          │         ┌────────┴────────┐        │
          │         │                 │        │
          ▼         ▼                 ▼        ▼
  ┌───────────────────────────────────────────────┐
  │          ValidationContext (上下文)            │
  └───────────────────────────────────────────────┘
          │                                   │
          ▼                                   ▼
  ┌──────────────┐                   ┌──────────────┐
  │ TypeRegistry │                   │ErrorCollector│
  │  (类型缓存)   │                   │ (错误收集)    │
  └──────────────┘                   └──────────────┘
```

---

## 🧩 核心模块

### 1. ValidatorEngine (验证引擎)

**职责**：协调各个组件，提供统一的验证入口

**依赖**：
- `PipelineExecutor`：执行验证管道
- `EventBus`：发布验证事件
- `HookManager`：管理生命周期钩子
- `TypeRegistry`：类型信息缓存
- `ErrorCollector`：错误收集

**方法**：
```go
type ValidatorEngine interface {
    // Validate 执行完整验证
    Validate(target any, scene Scene) *ValidationError
    
    // ValidateFields 验证指定字段
    ValidateFields(target any, scene Scene, fields ...string) *ValidationError
    
    // ValidateFieldsExcept 验证除指定字段外的所有字段
    ValidateFieldsExcept(target any, scene Scene, fields ...string) *ValidationError
}
```

### 2. PipelineExecutor (管道执行器)

**职责**：编排和执行验证策略

**特性**：
- 策略优先级排序
- 异常恢复机制
- 短路执行支持
- 并发执行支持（可选）

**方法**：
```go
type PipelineExecutor interface {
    // Execute 执行验证管道
    Execute(target any, ctx *ValidationContext) error
    
    // AddStrategy 添加策略
    AddStrategy(strategy ValidationStrategy)
    
    // RemoveStrategy 移除策略
    RemoveStrategy(strategyType StrategyType)
}
```

### 3. EventBus (事件总线)

**职责**：事件发布订阅，解耦组件

**特性**：
- 支持同步/异步事件
- 支持事件过滤
- 支持优先级
- 线程安全

**方法**：
```go
type EventBus interface {
    // Subscribe 订阅事件
    Subscribe(listener EventListener)
    
    // Unsubscribe 取消订阅
    Unsubscribe(listener EventListener)
    
    // Publish 发布事件
    Publish(event Event)
}
```

**事件类型**：
```go
type EventType int

const (
    EventValidationStart EventType = iota + 1
    EventValidationEnd
    EventStrategyStart
    EventStrategyEnd
    EventFieldValidated
    EventErrorOccurred
)
```

### 4. HookManager (钩子管理器)

**职责**：管理生命周期钩子

**方法**：
```go
type HookManager interface {
    // ExecuteBefore 执行前置钩子
    ExecuteBefore(target any, ctx *ValidationContext) error
    
    // ExecuteAfter 执行后置钩子
    ExecuteAfter(target any, ctx *ValidationContext) error
}
```

### 5. ErrorCollector (错误收集器)

**职责**：收集和管理验证错误

**特性**：
- 最大错误数限制
- 线程安全（可选）
- 错误去重（可选）
- 错误分组（可选）

**方法**：
```go
type ErrorCollector interface {
    // Add 添加错误
    Add(err *FieldError) bool
    
    // GetAll 获取所有错误
    GetAll() []*FieldError
    
    // GetByField 按字段获取错误
    GetByField(field string) []*FieldError
    
    // HasErrors 是否有错误
    HasErrors() bool
    
    // Count 错误数量
    Count() int
    
    // Clear 清空错误
    Clear()
}
```

### 6. TypeRegistry (类型注册表)

**职责**：类型信息缓存和分析

**拆分为两个职责**：
- `TypeAnalyzer`：分析类型信息
- `TypeCache`：缓存类型信息

**方法**：
```go
type TypeRegistry interface {
    // Analyze 分析类型
    Analyze(target any) *TypeInfo
    
    // Get 获取缓存的类型信息
    Get(typ reflect.Type) (*TypeInfo, bool)
    
    // Clear 清空缓存
    Clear()
}
```

---

## 🎨 设计模式

### 1. 策略模式 (Strategy Pattern)

**应用场景**：验证策略

```go
type ValidationStrategy interface {
    Type() StrategyType
    Priority() int8
    Validate(target any, ctx *ValidationContext) error
}

// 具体策略
type RuleStrategy struct { /* ... */ }
type BusinessStrategy struct { /* ... */ }
type NestedStrategy struct { /* ... */ }
```

### 2. 责任链模式 (Chain of Responsibility)

**应用场景**：策略按优先级执行

```go
type PipelineExecutor interface {
    Execute(target any, ctx *ValidationContext) error
}

// 策略链执行
for _, strategy := range executor.strategies {
    if err := strategy.Validate(target, ctx); err != nil {
        // 处理错误
    }
}
```

### 3. 观察者模式 (Observer Pattern)

**应用场景**：事件监听

```go
type EventBus interface {
    Subscribe(listener EventListener)
    Publish(event Event)
}

type EventListener interface {
    OnEvent(event Event)
}
```

### 4. 工厂模式 (Factory Pattern)

**应用场景**：验证器创建

```go
type ValidatorFactory interface {
    Create(opts ...EngineOption) Validator
    CreateDefault() Validator
}
```

### 5. 建造者模式 (Builder Pattern)

**应用场景**：复杂配置

```go
type ValidatorBuilder interface {
    WithStrategies(strategies ...ValidationStrategy) ValidatorBuilder
    WithEventBus(bus EventBus) ValidatorBuilder
    WithRegistry(registry TypeRegistry) ValidatorBuilder
    Build() Validator
}
```

### 6. 对象池模式 (Object Pool Pattern)

**应用场景**：上下文复用

```go
var contextPool = sync.Pool{
    New: func() interface{} {
        return &ValidationContext{}
    },
}

func AcquireContext() *ValidationContext {
    return contextPool.Get().(*ValidationContext)
}

func ReleaseContext(ctx *ValidationContext) {
    ctx.Reset()
    contextPool.Put(ctx)
}
```

### 7. 适配器模式 (Adapter Pattern)

**应用场景**：第三方库集成

```go
type ValidatorAdapter interface {
    Adapt(v *validator.Validate) ValidationStrategy
}

type PlaygroundValidatorAdapter struct {
    validator *validator.Validate
}
```

### 8. 模板方法模式 (Template Method Pattern)

**应用场景**：验证流程模板

```go
type BaseStrategy struct{}

func (s *BaseStrategy) Validate(target any, ctx *ValidationContext) error {
    if err := s.prepare(target, ctx); err != nil {
        return err
    }
    
    if err := s.doValidate(target, ctx); err != nil {
        return err
    }
    
    return s.cleanup(target, ctx)
}
```

---

## 🚀 相比 v5 的改进

### 改进对比表

| 维度 | v5 | v5_refactored | 改进 |
|------|----|--------------|----- |
| **单一职责** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | Engine 职责进一步拆分 |
| **开放封闭** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 更多扩展点 |
| **接口隔离** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 接口更细粒度 |
| **依赖倒置** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 完全面向接口 |
| **可测试性** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 组件可独立测试 |
| **可扩展性** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 更灵活的扩展机制 |
| **可维护性** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 职责更清晰 |
| **性能** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 并发支持、更好的缓存 |

### 核心改进点

#### 1. 职责分离

**v5**：
```go
type ValidatorEngine struct {
    validator      *validator.Validate
    sceneMatcher   SceneMatcher
    registry       Registry
    strategies     []ValidationStrategy
    listeners      []ValidationListener  // 混在一起
    errorFormatter ErrorFormatter
    maxDepth       int
    maxErrors      int
}
```

**v5_refactored**：
```go
type ValidatorEngine struct {
    pipeline       PipelineExecutor    // 策略执行
    eventBus       EventBus            // 事件管理
    hookManager    HookManager         // 钩子管理
    registry       TypeRegistry        // 类型缓存
    errorCollector ErrorCollector      // 错误收集
}
```

#### 2. 接口细化

**v5**：
```go
type Registry interface {
    Register(target any) *TypeInfo
    Get(target any) (*TypeInfo, bool)
    Clear()
    Stats() (count int)
}
```

**v5_refactored**：
```go
type TypeInfoReader interface {
    Get(typ reflect.Type) (*TypeInfo, bool)
}

type TypeInfoWriter interface {
    Set(typ reflect.Type, info *TypeInfo)
}

type TypeAnalyzer interface {
    Analyze(target any) *TypeInfo
}

type TypeRegistry interface {
    TypeInfoReader
    TypeInfoWriter
    TypeAnalyzer
}
```

#### 3. 事件驱动

**v5**：直接调用监听器
```go
func (e *ValidatorEngine) notifyValidationStart(ctx *ValidationContext) {
    for _, listener := range e.listeners {
        listener.OnValidationStart(ctx)
    }
}
```

**v5_refactored**：通过事件总线解耦
```go
func (e *ValidatorEngine) Validate(target any, scene Scene) *ValidationError {
    e.eventBus.Publish(NewEvent(EventValidationStart, ctx))
    // ...
}
```

#### 4. 错误收集

**v5**：错误收集在 Context 中
```go
type ValidationContext struct {
    errors []*FieldError  // 混在上下文中
}
```

**v5_refactored**：独立的错误收集器
```go
type ErrorCollector interface {
    Add(err *FieldError) bool
    GetAll() []*FieldError
    GetByField(field string) []*FieldError
    HasErrors() bool
}
```

---

## 📊 性能优化

### 1. 并发执行

对于独立的验证策略，支持并发执行：

```go
type ConcurrentPipelineExecutor struct {
    strategies []ValidationStrategy
    workers    int
}

func (e *ConcurrentPipelineExecutor) Execute(target any, ctx *ValidationContext) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(e.strategies))
    
    for _, strategy := range e.strategies {
        wg.Add(1)
        go func(s ValidationStrategy) {
            defer wg.Done()
            if err := s.Validate(target, ctx); err != nil {
                errChan <- err
            }
        }(strategy)
    }
    
    wg.Wait()
    close(errChan)
    
    // 收集错误
    for err := range errChan {
        ctx.AddError(err)
    }
    
    return nil
}
```

### 2. 缓存优化

多级缓存策略：

```go
type MultiLevelTypeCache struct {
    l1 *sync.Map           // 一级缓存：热点数据
    l2 map[reflect.Type]*TypeInfo  // 二级缓存：完整数据
    mu sync.RWMutex
}
```

### 3. 内存池

更细粒度的对象池：

```go
var (
    contextPool      sync.Pool
    errorCollectorPool sync.Pool
    eventPool        sync.Pool
)
```

---

## 🧪 可测试性

### 1. 组件独立测试

每个组件都可以独立测试：

```go
func TestPipelineExecutor(t *testing.T) {
    executor := NewDefaultPipelineExecutor()
    executor.AddStrategy(NewMockStrategy())
    
    ctx := NewValidationContext(SceneCreate, 10)
    err := executor.Execute(&User{}, ctx)
    
    assert.NoError(t, err)
}
```

### 2. Mock 支持

所有依赖都是接口，易于 Mock：

```go
type MockEventBus struct {
    events []Event
}

func (m *MockEventBus) Publish(event Event) {
    m.events = append(m.events, event)
}

func TestValidatorEngine(t *testing.T) {
    mockBus := &MockEventBus{}
    engine := NewValidatorEngine(
        NewDefaultPipelineExecutor(),
        mockBus,  // 注入 Mock
        // ...
    )
    
    engine.Validate(&User{}, SceneCreate)
    
    assert.Equal(t, 2, len(mockBus.events))  // 验证事件发布
}
```

---

## 📝 总结

v5_refactored 相比 v5 的核心改进：

1. ✅ **更好的职责分离**：每个组件只做一件事
2. ✅ **更细的接口粒度**：符合接口隔离原则
3. ✅ **完全的依赖倒置**：所有依赖都是接口
4. ✅ **事件驱动架构**：组件间解耦更彻底
5. ✅ **更强的扩展性**：更多的扩展点和钩子
6. ✅ **更好的可测试性**：组件可独立测试
7. ✅ **性能优化**：支持并发、多级缓存
8. ✅ **更清晰的代码**：职责明确，易于理解

这是一个真正意义上的**企业级、生产就绪**的验证器框架。

