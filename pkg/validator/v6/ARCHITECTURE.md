# Validator v6 架构设计文档

## 设计目标

基于 SOLID 原则和软件工程最佳实践，重构 v5 验证器框架，实现：

1. **高内聚低耦合**：每个模块职责单一，模块间依赖最小化
2. **可扩展性**：通过插件机制和扩展点支持功能扩展
3. **可维护性**：清晰的代码结构和职责划分
4. **可测试性**：所有组件可独立测试
5. **可读性**：直观的 API 和良好的文档
6. **可复用性**：组件可在不同场景复用

## 架构分层

```
┌─────────────────────────────────────────────────────────────┐
│                      应用层 (Application)                     │
│                    全局实例、便捷API                           │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                    门面层 (Facade)                            │
│              ValidatorFacade - 统一入口                        │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                  编排层 (Orchestration)                       │
│         ValidationOrchestrator - 流程编排                     │
│         EventDispatcher - 事件分发                            │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                   策略层 (Strategy)                           │
│    RuleStrategy | BusinessStrategy | NestedStrategy         │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                   核心层 (Core)                               │
│    SceneMatcher | ErrorCollector | TypeRegistry             │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                  基础设施层 (Infrastructure)                  │
│         对象池 | 缓存 | 工具类                                │
└─────────────────────────────────────────────────────────────┘
```

## SOLID 原则应用

### 1. 单一职责原则 (SRP)

**问题**：v5 的 `ValidatorEngine` 承担了过多职责

**解决方案**：

- `ValidatorFacade`：对外统一入口（门面模式）
- `ValidationOrchestrator`：流程编排
- `EventDispatcher`：事件分发
- `ErrorCollector`：错误收集
- `StrategyExecutor`：策略执行
- `TypeRegistry`：类型注册与缓存

### 2. 开放封闭原则 (OCP)

**解决方案**：

- 定义清晰的扩展点接口
- 使用插件机制扩展功能
- 策略模式支持新验证策略
- 装饰器模式扩展功能

### 3. 里氏替换原则 (LSP)

**解决方案**：

- 所有策略实现 `ValidationStrategy` 接口
- 所有格式化器实现 `ErrorFormatter` 接口
- 接口契约明确，子类可安全替换

### 4. 接口隔离原则 (ISP)

**解决方案**：

- 将大接口拆分为小接口
- `RuleProvider` 只提供规则
- `BusinessValidator` 只做业务验证
- `LifecycleHook` 只管理生命周期

### 5. 依赖倒置原则 (DIP)

**解决方案**：

- 高层模块依赖抽象接口，不依赖具体实现
- 通过构造函数注入依赖
- 使用工厂模式创建对象

## 核心组件设计

### 1. ValidatorFacade（门面）

职责：提供统一的验证入口，隐藏内部复杂性

```go
type ValidatorFacade interface {
    Validate(target any, scene Scene) error
    ValidateFields(target any, scene Scene, fields ...string) error
    ValidateStruct(target any) error
}
```

### 2. ValidationOrchestrator（编排器）

职责：编排验证流程，协调各组件

```go
type ValidationOrchestrator interface {
    Orchestrate(req *ValidationRequest) (*ValidationResult, error)
}
```

### 3. StrategyExecutor（策略执行器）

职责：执行验证策略，处理异常

```go
type StrategyExecutor interface {
    Execute(strategy ValidationStrategy, req *ValidationRequest) error
}
```

### 4. EventDispatcher（事件分发器）

职责：分发验证事件给监听器

```go
type EventDispatcher interface {
    Dispatch(event ValidationEvent)
    Subscribe(listener ValidationListener)
}
```

### 5. ErrorCollector（错误收集器）

职责：收集和管理验证错误

```go
type ErrorCollector interface {
    Add(err *FieldError) bool
    GetAll() []*FieldError
    HasErrors() bool
    Clear()
}
```

## 设计模式应用

### 1. 门面模式 (Facade Pattern)

`ValidatorFacade` 为复杂的验证系统提供简单统一的接口

### 2. 策略模式 (Strategy Pattern)

不同的验证策略可灵活切换和组合

### 3. 观察者模式 (Observer Pattern)

通过事件监听器观察验证过程

### 4. 工厂模式 (Factory Pattern)

通过 Builder 创建配置复杂的验证器

### 5. 模板方法模式 (Template Method Pattern)

定义验证流程骨架，子类实现具体步骤

### 6. 责任链模式 (Chain of Responsibility)

验证策略按优先级依次执行

### 7. 对象池模式 (Object Pool Pattern)

复用验证上下文对象，减少内存分配

## 扩展机制

### 1. 插件接口

```go
type Plugin interface {
    Name() string
    Init(config map[string]any) error
    BeforeValidate(ctx *ValidationContext) error
    AfterValidate(ctx *ValidationContext) error
}
```

### 2. 中间件

```go
type Middleware func(next ValidationHandler) ValidationHandler
```

### 3. 钩子

- BeforeValidation
- AfterValidation
- OnError
- OnSuccess

## 性能优化

1. **缓存优化**：类型信息缓存、字段访问器缓存
2. **对象池**：ValidationContext、ErrorCollector 对象复用
3. **延迟初始化**：按需初始化组件
4. **位运算**：场景匹配使用位运算

## 测试策略

1. **单元测试**：每个组件独立测试
2. **集成测试**：组件协作测试
3. **性能测试**：基准测试和性能分析
4. **模拟测试**：使用 mock 对象隔离依赖

## 向后兼容

提供适配器层，兼容 v5 API：

```go
type V5Adapter struct {
    facade ValidatorFacade
}

func (a *V5Adapter) Validate(target any, scene Scene) *ValidationError {
    // 适配到 v6
}
```

## 使用示例

```go
// 创建验证器
validator := v6.NewBuilder().
    WithStrategies(
        strategy.NewRuleStrategy(),
        strategy.NewBusinessStrategy(),
    ).
    WithPlugins(
        plugin.NewLoggingPlugin(),
        plugin.NewMetricsPlugin(),
    ).
    WithErrorFormatter(formatter.NewI18nFormatter("zh")).
    Build()

// 执行验证
if err := validator.Validate(user, SceneCreate); err != nil {
    // 处理错误
}
```

## 文件结构

```
v6/
├── ARCHITECTURE.md          # 架构文档
├── README.md                # 使用文档
├── core/                    # 核心接口和类型
│   ├── interface.go         # 核心接口定义
│   ├── types.go             # 核心类型定义
│   ├── scene.go             # 场景定义
│   └── error.go             # 错误类型
├── facade/                  # 门面层
│   ├── validator.go         # ValidatorFacade 实现
│   └── builder.go           # 构建器
├── orchestrator/            # 编排层
│   ├── orchestrator.go      # ValidationOrchestrator 实现
│   ├── executor.go          # StrategyExecutor 实现
│   └── dispatcher.go        # EventDispatcher 实现
├── strategy/                # 策略层
│   ├── interface.go         # 策略接口
│   ├── rule.go              # 规则验证策略
│   ├── business.go          # 业务验证策略
│   └── nested.go            # 嵌套验证策略
├── collector/               # 错误收集器
│   └── error_collector.go
├── registry/                # 类型注册表
│   └── type_registry.go
├── matcher/                 # 场景匹配器
│   └── scene_matcher.go
├── formatter/               # 错误格式化器
│   ├── interface.go
│   ├── default.go
│   └── i18n.go
├── plugin/                  # 插件
│   ├── interface.go
│   ├── logging.go
│   └── metrics.go
├── context/                 # 验证上下文
│   └── context.go
├── pool/                    # 对象池
│   └── pool.go
└── adapter/                 # v5 适配器
    └── v5_adapter.go
```

## 迁移指南

详见 [MIGRATION.md](./MIGRATION.md)

## 总结

v6 相比 v5 的改进：

| 方面 | v5 | v6 | 改进 |
|-----|----|----|------|
| 单一职责 | ⚠️ Engine 职责过多 | ✅ 职责清晰分离 | ⭐⭐⭐⭐⭐ |
| 可扩展性 | ⚠️ 扩展点有限 | ✅ 插件+中间件机制 | ⭐⭐⭐⭐⭐ |
| 可测试性 | ⚠️ 组件耦合 | ✅ 完全解耦 | ⭐⭐⭐⭐⭐ |
| 可维护性 | ⚠️ 文件较大 | ✅ 模块化清晰 | ⭐⭐⭐⭐⭐ |
| 依赖倒置 | ⚠️ 部分硬编码 | ✅ 完全依赖接口 | ⭐⭐⭐⭐⭐ |
| 代码量 | 850 行 | ~1200 行 | 为了更好的架构 |

