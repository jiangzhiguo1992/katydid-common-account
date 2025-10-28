# v6 设计原则总结

## SOLID 原则应用

### 1. 单一职责原则 (Single Responsibility Principle)

**定义**: 一个类应该只有一个引起它变化的原因。

**v5 的问题**:
```go
// v5: ValidatorEngine 承担了太多职责
type ValidatorEngine struct {
    // 验证执行
    // 策略管理
    // 错误收集
    // 事件分发
    // 类型注册
    // ...
}
```

**v6 的改进**:
```go
// v6: 职责清晰分离

// ValidatorFacade - 只负责提供统一入口
type ValidatorFacade struct {
    orchestrator ValidationOrchestrator
}

// ValidationOrchestrator - 只负责编排流程
type ValidationOrchestrator interface {
    Orchestrate(req *ValidationRequest) (*ValidationResult, error)
}

// StrategyExecutor - 只负责执行策略
type StrategyExecutor interface {
    Execute(strategy ValidationStrategy, req *ValidationRequest, ctx ValidationContext) error
}

// ErrorCollector - 只负责收集错误
type ErrorCollector interface {
    Add(err *FieldError) bool
    GetAll() []*FieldError
}

// EventDispatcher - 只负责事件分发
type EventDispatcher interface {
    Dispatch(event ValidationEvent)
}

// TypeRegistry - 只负责类型管理
type TypeRegistry interface {
    Register(target any) TypeInfo
}
```

**收益**:
- ✅ 每个组件职责明确
- ✅ 易于理解和维护
- ✅ 修改一个职责不影响其他职责
- ✅ 更容易编写单元测试

---

### 2. 开放封闭原则 (Open-Closed Principle)

**定义**: 软件实体应该对扩展开放，对修改封闭。

**v6 的实现**:

```go
// 核心接口保持稳定
type ValidationStrategy interface {
    Name() string
    Type() StrategyType
    Priority() int
    Validate(req *ValidationRequest, ctx ValidationContext) error
}

// 扩展新策略 - 无需修改现有代码
type CustomStrategy struct{}

func (s *CustomStrategy) Name() string { return "CustomStrategy" }
func (s *CustomStrategy) Type() StrategyType { return StrategyTypeCustom }
func (s *CustomStrategy) Priority() int { return 30 }
func (s *CustomStrategy) Validate(req *ValidationRequest, ctx ValidationContext) error {
    // 自定义验证逻辑
    return nil
}

// 使用
validator := NewBuilder().
    WithStrategies(&CustomStrategy{}).
    Build()
```

**扩展点**:
1. **验证策略**: 实现 `ValidationStrategy` 接口
2. **插件**: 实现 `Plugin` 接口
3. **监听器**: 实现 `ValidationListener` 接口
4. **错误格式化**: 实现 `ErrorFormatter` 接口
5. **场景匹配**: 实现 `SceneMatcher` 接口

**收益**:
- ✅ 新功能通过扩展添加
- ✅ 不破坏现有代码
- ✅ 降低引入 bug 的风险
- ✅ 支持插件化架构

---

### 3. 里氏替换原则 (Liskov Substitution Principle)

**定义**: 子类必须能够替换其基类。

**v6 的实现**:

```go
// 接口契约明确
type Validator interface {
    Validate(target any, scene Scene) error
    ValidateWithRequest(req *ValidationRequest) (*ValidationResult, error)
}

// 所有实现都遵守契约
var v1 Validator = &ValidatorFacade{...}
var v2 Validator = &CustomValidator{...}

// 可以安全替换
func doValidation(validator Validator, user *User) error {
    return validator.Validate(user, SceneCreate)
}
```

**契约保证**:
```go
// ErrorCollector 的契约
type ErrorCollector interface {
    Add(err *FieldError) bool  // 返回 false 表示达到上限
    GetAll() []*FieldError     // 总是返回非 nil 切片
    HasErrors() bool           // 等价于 len(GetAll()) > 0
}

// 所有实现必须遵守这些契约
```

**收益**:
- ✅ 接口可预测
- ✅ 减少意外行为
- ✅ 更容易编写测试
- ✅ 支持 mock 和 stub

---

### 4. 接口隔离原则 (Interface Segregation Principle)

**定义**: 客户端不应该依赖它不需要的接口。

**v5 的问题**:
```go
// v5: 臃肿的接口（假设）
type Validator interface {
    Validate(...)
    ValidateFields(...)
    ValidateStruct(...)
    AddStrategy(...)
    RemoveStrategy(...)
    GetStrategies(...)
    SetMaxErrors(...)
    GetMaxErrors(...)
    // ... 更多方法
}
```

**v6 的改进**:
```go
// v6: 精简的接口，职责单一

// 用户接口 - 只提供验证功能
type Validator interface {
    Validate(target any, scene Scene) error
    ValidateWithRequest(req *ValidationRequest) (*ValidationResult, error)
}

// 用户实现的接口 - 只提供规则
type RuleProvider interface {
    GetRules() map[Scene]map[string]string
}

// 用户实现的接口 - 只做业务验证
type BusinessValidator interface {
    ValidateBusiness(scene Scene, ctx ValidationContext) error
}

// 用户实现的接口 - 只管理生命周期
type LifecycleHook interface {
    BeforeValidation(ctx ValidationContext) error
    AfterValidation(ctx ValidationContext) error
}

// 可选实现，按需选择
type User struct {
    Name string
}

// 只需要规则验证
func (u *User) GetRules() map[Scene]map[string]string { ... }

// 需要业务验证就实现这个接口
func (u *User) ValidateBusiness(scene Scene, ctx ValidationContext) error { ... }
```

**收益**:
- ✅ 接口精简，易于实现
- ✅ 降低耦合度
- ✅ 按需实现功能
- ✅ 提高代码可读性

---

### 5. 依赖倒置原则 (Dependency Inversion Principle)

**定义**: 高层模块不应该依赖低层模块，两者都应该依赖抽象。

**v5 的问题**:
```go
// v5: 直接依赖具体实现
type ValidatorEngine struct {
    validator      *validator.Validate      // 具体实现
    sceneMatcher   *SceneBitMatcher        // 具体实现
    registry       *TypeRegistry           // 具体实现
    errorFormatter *LocalizesErrorFormatter // 具体实现
}
```

**v6 的改进**:
```go
// v6: 依赖接口抽象
type OrchestratorImpl struct {
    strategies      []ValidationStrategy  // 接口
    executor        StrategyExecutor     // 接口
    eventDispatcher EventDispatcher      // 接口
    plugins         []Plugin             // 接口
}

// 通过构造函数注入
func NewOrchestrator(opts ...OrchestratorOption) ValidationOrchestrator {
    o := &OrchestratorImpl{}
    for _, opt := range opts {
        opt(o)
    }
    // 如果没有提供，使用默认实现
    if o.executor == nil {
        o.executor = NewStrategyExecutor()
    }
    return o
}

// 选项模式支持依赖注入
func WithExecutor(executor StrategyExecutor) OrchestratorOption {
    return func(o *OrchestratorImpl) {
        o.executor = executor
    }
}

// 使用
orchestrator := NewOrchestrator(
    WithExecutor(customExecutor),
    WithEventDispatcher(customDispatcher),
)
```

**依赖关系图**:
```
高层 (Facade)
    ↓ 依赖接口
中层 (Orchestrator)
    ↓ 依赖接口
低层 (Strategy, Collector, Registry)
    ↓ 实现接口
接口层 (Interfaces)
```

**收益**:
- ✅ 降低模块间耦合
- ✅ 易于替换实现
- ✅ 易于编写单元测试（mock）
- ✅ 支持依赖注入容器

---

## 其他设计原则

### 6. 高内聚低耦合

**高内聚**:
```go
// ErrorCollector 模块 - 高内聚
// 所有与错误收集相关的逻辑都在这里
type ErrorCollectorImpl struct {
    errors    []*FieldError
    maxErrors int
}

func (c *ErrorCollectorImpl) Add(err *FieldError) bool { ... }
func (c *ErrorCollectorImpl) GetAll() []*FieldError { ... }
func (c *ErrorCollectorImpl) HasErrors() bool { ... }
func (c *ErrorCollectorImpl) Clear() { ... }
```

**低耦合**:
```go
// 模块间通过接口通信，降低耦合
type ValidationOrchestrator interface {
    Orchestrate(req *ValidationRequest) (*ValidationResult, error)
}

type StrategyExecutor interface {
    Execute(strategy ValidationStrategy, ...) error
}

// Orchestrator 不需要知道 Executor 的具体实现
```

### 7. 组合优于继承

```go
// v6: 使用组合
type ValidatorFacade struct {
    orchestrator ValidationOrchestrator  // 组合
}

type OrchestratorImpl struct {
    executor        StrategyExecutor     // 组合
    eventDispatcher EventDispatcher      // 组合
}

// 而不是继承
// type ValidatorFacade extends BaseValidator { ... }
```

### 8. 面向接口编程

```go
// 定义接口
type ValidationStrategy interface {
    Validate(req *ValidationRequest, ctx ValidationContext) error
}

// 面向接口编程
func executeStrategy(strategy ValidationStrategy, req *ValidationRequest, ctx ValidationContext) error {
    return strategy.Validate(req, ctx)
}

// 而不是面向具体类型
// func executeStrategy(strategy *RuleStrategy, ...) error
```

### 9. 最小知识原则（迪米特法则）

```go
// ValidatorFacade 只与 Orchestrator 通信
type ValidatorFacade struct {
    orchestrator ValidationOrchestrator
}

func (f *ValidatorFacade) Validate(target any, scene Scene) error {
    req := NewValidationRequest(target, scene)
    result, err := f.orchestrator.Orchestrate(req)  // 只调用 orchestrator
    return result.ToError()
}

// 不直接访问更深层的对象
// f.orchestrator.executor.Execute(...)  ❌
```

### 10. 不要重复自己 (DRY)

```go
// 复用对象池
var validationContextPool = sync.Pool{
    New: func() interface{} {
        return &ValidationContext{...}
    },
}

// 所有地方都使用同一个池
func NewValidationContext(...) ValidationContext {
    ctx := validationContextPool.Get().(*ValidationContext)
    // ...
    return ctx
}
```

## 设计模式应用

### 1. 门面模式 (Facade Pattern)
- `ValidatorFacade` 为复杂系统提供简单接口

### 2. 策略模式 (Strategy Pattern)
- `ValidationStrategy` 支持不同验证策略

### 3. 观察者模式 (Observer Pattern)
- `EventDispatcher` + `ValidationListener`

### 4. 建造者模式 (Builder Pattern)
- `Builder` 构建复杂的验证器

### 5. 工厂模式 (Factory Pattern)
- `NewOrchestrator`, `NewErrorCollector` 等

### 6. 模板方法模式 (Template Method Pattern)
- `ValidationOrchestrator.Orchestrate()` 定义验证流程骨架

### 7. 责任链模式 (Chain of Responsibility)
- 验证策略按优先级依次执行

### 8. 对象池模式 (Object Pool Pattern)
- `validationContextPool` 复用对象

## 总结

v6 相比 v5 的改进：

| 原则/模式 | v5 | v6 | 改进 |
|---------|----|----|------|
| 单一职责 | ⚠️ Engine 职责过多 | ✅ 职责清晰分离 | ⭐⭐⭐⭐⭐ |
| 开放封闭 | ⚠️ 扩展点有限 | ✅ 插件+策略机制 | ⭐⭐⭐⭐⭐ |
| 里氏替换 | ✅ 基本遵循 | ✅ 完全遵循 | ⭐⭐⭐⭐ |
| 接口隔离 | ⚠️ 接口较臃肿 | ✅ 接口精简 | ⭐⭐⭐⭐⭐ |
| 依赖倒置 | ⚠️ 部分硬编码 | ✅ 完全依赖接口 | ⭐⭐⭐⭐⭐ |
| 高内聚 | ⚠️ 一般 | ✅ 高 | ⭐⭐⭐⭐ |
| 低耦合 | ⚠️ 耦合较紧 | ✅ 松耦合 | ⭐⭐⭐⭐⭐ |
| 可测试性 | ⚠️ 较难测试 | ✅ 易于测试 | ⭐⭐⭐⭐⭐ |
| 可扩展性 | ⚠️ 有限 | ✅ 优秀 | ⭐⭐⭐⭐⭐ |
| 可维护性 | ⚠️ 一般 | ✅ 优秀 | ⭐⭐⭐⭐⭐ |

v6 是一个教科书级别的 Go 验证器实现，充分展示了软件工程最佳实践的应用。

