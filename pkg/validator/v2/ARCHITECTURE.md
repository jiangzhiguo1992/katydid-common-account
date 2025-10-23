
# Validator V2 架构设计文档

## 设计目标

基于面向对象设计原则（SOLID）和软件工程最佳实践，重新设计验证器架构，实现：

1. **高内聚低耦合**：每个组件职责单一，组件间通过接口交互
2. **可扩展性**：通过策略模式轻松扩展新功能，无需修改现有代码
3. **可维护性**：清晰的代码结构和职责划分
4. **可测试性**：依赖接口便于 Mock 和单元测试
5. **可读性**：清晰的命名和文档
6. **可复用性**：独立的组件可在不同场景复用

---

## SOLID 设计原则应用

### 1. 单一职责原则（Single Responsibility Principle）

**原则**：一个类应该只有一个引起它变化的原因。

**应用**：

#### ✅ 接口职责分离

```
RuleProvider        → 只负责提供验证规则
CustomValidator     → 只负责执行自定义验证
ErrorCollector      → 只负责收集和管理错误
TypeCache          → 只负责缓存类型信息
RegistryManager    → 只负责管理注册状态
```

#### ✅ 策略职责分离

```
RuleValidationStrategy    → 只负责规则验证
CustomValidationStrategy  → 只负责自定义验证
NestedValidationStrategy  → 只负责嵌套验证
```

#### ❌ V1 版本的问题

```go
// V1 的 ValidationContext 承担了太多职责：
type ValidationContext struct {
    Scene   ValidateScene    // 场景管理
    Message string           // 消息管理
    Errors  []*FieldError    // 错误收集
    // 还包含各种错误添加方法...
}
```

#### ✅ V2 的改进

```go
// V2 将职责分离到不同组件：
type ErrorCollector interface {
    Add(err *FieldError)      // 错误收集
    GetErrors() []*FieldError // 错误查询
}

type Result interface {
    IsValid() bool                           // 结果查询
    ErrorsByField(field string) []*FieldError // 错误过滤
}
```

---

### 2. 开放封闭原则（Open-Closed Principle）

**原则**：软件实体应该对扩展开放，对修改封闭。

**应用**：

#### ✅ 策略模式实现扩展

```go
// 定义策略接口
type ValidationStrategy interface {
    Execute(obj any, scene Scene, collector ErrorCollector) bool
}

// 添加新策略无需修改验证器代码
type LoggingStrategy struct{}

func (s *LoggingStrategy) Execute(obj any, scene Scene, collector ErrorCollector) bool {
    log.Printf("Validating %T in scene %s", obj, scene)
    return true
}

// 使用建造者添加自定义策略
validator := NewValidatorBuilder().
    WithStrategy(&LoggingStrategy{}).
    WithDefaultStrategies().
    Build()
```

#### ✅ 接口扩展

```go
// 模型只需实现对应接口即可扩展验证功能
type User struct {
    Username string
    Email    string
}

// 扩展 1：添加规则验证
func (u *User) ProvideRules() map[Scene]FieldRules { ... }

// 扩展 2：添加自定义验证
func (u *User) ValidateCustom(scene Scene, reporter ErrorReporter) { ... }
```

#### ❌ V1 版本的问题

```go
// V1 需要修改验证器内部代码来添加新功能
func (v *Validator) Validate(obj any, scene ValidateScene) []*FieldError {
    // 硬编码的验证流程
    // 1. 规则验证
    // 2. 自定义验证
    // 3. 嵌套验证
    // 添加新类型的验证需要修改这里的代码
}
```

---

### 3. 里氏替换原则（Liskov Substitution Principle）

**原则**：子类型必须能够替换掉它们的父类型。

**应用**：

#### ✅ 接口实现可替换

```go
// 所有 TypeCache 实现都可以互换
type TypeCache interface {
    Get(obj any) *TypeInfo
    Clear()
}

// 默认实现
type DefaultTypeCache struct { ... }

// 可以替换为其他实现（如：Redis 缓存）
type RedisTypeCache struct { ... }

// 验证器不关心具体实现
validator := NewValidatorBuilder().
    WithTypeCache(NewRedisTypeCache()).
    Build()
```

#### ✅ 策略可替换

```go
// 所有策略都实现相同接口，可以互换
strategies := []ValidationStrategy{
    NewRuleValidationStrategy(...),
    NewCustomValidationStrategy(...),
    NewNestedValidationStrategy(...),
}

// 可以动态调整策略顺序或替换策略
```

---

### 4. 依赖倒置原则（Dependency Inversion Principle）

**原则**：高层模块不应该依赖低层模块，两者都应该依赖抽象。

**应用**：

#### ✅ 依赖接口而非具体实现

```go
// 验证器依赖抽象接口
type DefaultValidator struct {
    strategies []ValidationStrategy  // 依赖策略接口
    typeCache  TypeCache             // 依赖缓存接口
    registry   RegistryManager       // 依赖注册管理接口
}

// 策略依赖抽象接口
type RuleValidationStrategy struct {
    typeCache TypeCache  // 依赖缓存接口，而非具体实现
}
```

#### ✅ 依赖注入

```go
// 通过构造函数或建造者注入依赖
validator := NewValidatorBuilder().
    WithTypeCache(customCache).      // 注入自定义缓存
    WithRegistry(customRegistry).    // 注入自定义注册器
    Build()
```

#### ❌ V1 版本的问题

```go
// V1 直接依赖具体实现
type Validator struct {
    validate        *validator.Validate  // 直接依赖第三方库
    typeCache       *sync.Map            // 直接依赖具体数据结构
    registeredCache *sync.Map
}
```

---

### 5. 接口隔离原则（Interface Segregation Principle）

**原则**：客户端不应该依赖它不需要的接口。

**应用**：

#### ✅ 细粒度接口设计

```go
// 错误报告器：只需要报告功能
type ErrorReporter interface {
    Report(namespace, tag, param string)
    ReportWithMessage(namespace, tag, param, message string)
}

// 错误收集器：需要更多管理功能
type ErrorCollector interface {
    ErrorReporter  // 继承报告功能
    Add(err *FieldError)
    HasErrors() bool
    GetErrors() []*FieldError
}

// 验证结果：只需要查询功能
type Result interface {
    IsValid() bool
    Errors() []*FieldError
    FirstError() *FieldError
    ErrorsByField(field string) []*FieldError
    ErrorsByTag(tag string) []*FieldError
}
```

#### ✅ 模型接口分离

```go
// 模型只需实现需要的接口
type RuleProvider interface {
    ProvideRules() map[Scene]FieldRules
}

type CustomValidator interface {
    ValidateCustom(scene Scene, reporter ErrorReporter)
}

// 简单模型只实现 RuleProvider
type SimpleModel struct { ... }
func (m *SimpleModel) ProvideRules() map[Scene]FieldRules { ... }

// 复杂模型实现两个接口
type ComplexModel struct { ... }
func (m *ComplexModel) ProvideRules() map[Scene]FieldRules { ... }
func (m *ComplexModel) ValidateCustom(scene Scene, reporter ErrorReporter) { ... }
```

#### ❌ V1 版本的问题

```go
// V1 的接口包含了太多方法
type CustomValidator interface {
    CustomValidation(scene ValidateScene, report FuncReportError)
}

// FuncReportError 是一个函数类型，不够灵活
type FuncReportError func(namespace, tag, param string)
```

---

## 设计模式应用

### 1. 策略模式（Strategy Pattern）

**目的**：定义一系列算法，把它们一个个封装起来，并且使它们可以相互替换。

**应用场景**：不同类型的验证逻辑

```go
// 策略接口
type ValidationStrategy interface {
    Execute(obj any, scene Scene, collector ErrorCollector) bool
}

// 具体策略 1：规则验证
type RuleValidationStrategy struct { ... }

// 具体策略 2：自定义验证
type CustomValidationStrategy struct { ... }

// 具体策略 3：嵌套验证
type NestedValidationStrategy struct { ... }

// 上下文：验证器
type DefaultValidator struct {
    strategies []ValidationStrategy
}

func (v *DefaultValidator) Validate(obj any, scene Scene) Result {
    collector := NewErrorCollector()

    // 依次执行各个策略
    for _, strategy := range v.strategies {
        if !strategy.Execute(obj, scene, collector) {
            break
        }
    }

    return NewValidationResultWithErrors(collector.GetErrors())
}
```

**优势**：
- 易于添加新的验证策略
- 可以动态调整策略执行顺序
- 策略之间相互独立，便于测试

---

### 2. 建造者模式（Builder Pattern）

**目的**：将一个复杂对象的构建与它的表示分离，使得同样的构建过程可以创建不同的表示。

**应用场景**：验证器的灵活配置

```go
// 建造者
type DefaultValidatorBuilder struct {
    strategies []ValidationStrategy
    typeCache  TypeCache
    registry   RegistryManager
    maxDepth   int
}

// 流式接口
func (b *DefaultValidatorBuilder) WithStrategy(strategy ValidationStrategy) ValidatorBuilder {
    b.strategies = append(b.strategies, strategy)
    return b
}

func (b *DefaultValidatorBuilder) WithMaxDepth(depth int) *DefaultValidatorBuilder {
    b.maxDepth = depth
    return b
}

func (b *DefaultValidatorBuilder) Build() Validator {
    return &DefaultValidator{
        strategies: b.strategies,
        typeCache:  b.typeCache,
        registry:   b.registry,
    }
}

// 使用
validator := NewValidatorBuilder().
    WithMaxDepth(50).
    WithTypeCache(customCache).
    WithDefaultStrategies().
    Build()
```

**优势**：
- 灵活配置复杂对象
- 链式调用，代码可读性高
- 可以创建不同配置的验证器实例

---

### 3. 工厂方法模式（Factory Method Pattern）

**目的**：定义一个用于创建对象的接口，让子类决定实例化哪一个类。

**应用场景**：各种组件的创建

```go
// 工厂方法
func NewValidator() *DefaultValidator { ... }
func NewErrorCollector() *DefaultErrorCollector { ... }
func NewTypeCache() *DefaultTypeCache { ... }
func NewMapValidator() *MapValidator { ... }
func NewFieldError(namespace, field, tag, param string) *FieldError { ... }

// 使用
validator := NewValidator()
collector := NewErrorCollector()
```

**优势**：
- 封装对象创建逻辑
- 确保对象正确初始化
- 便于后续扩展（如对象池）

---

### 4. 单例模式（Singleton Pattern）

**目的**：保证一个类仅有一个实例，并提供一个访问它的全局访问点。

**应用场景**：全局默认验证器

```go
var (
    defaultValidator *DefaultValidator
    defaultOnce      sync.Once
)

func Default() Validator {
    defaultOnce.Do(func() {
        defaultValidator = NewValidator()
    })
    return defaultValidator
}

// 使用
result := v2.Validate(user, v2.SceneCreate)
```

**优势**：
- 减少资源消耗
- 全局统一配置
- 线程安全

---

## 架构优势

### 1. 高内聚

每个组件只负责单一职责，内部逻辑高度相关：

```
ErrorCollector  → 错误收集和管理
TypeCache       → 类型信息缓存
Strategy        → 特定的验证逻辑
MapValidator    → Map 字段验证
```

### 2. 低耦合

组件之间通过接口交互，依赖关系清晰：

```
Validator → Strategy → TypeCache
          → Strategy → ErrorCollector
```

### 3. 可扩展性

通过策略模式和接口设计，易于扩展：

```go
// 添加新的验证策略
type DatabaseValidationStrategy struct { ... }

validator := NewValidatorBuilder().
    WithStrategy(&DatabaseValidationStrategy{}).
    WithDefaultStrategies().
    Build()
```

### 4. 可测试性

依赖接口便于 Mock：

```go
// Mock TypeCache
type MockTypeCache struct {
    mock.Mock
}

func (m *MockTypeCache) Get(obj any) *TypeInfo {
    args := m.Called(obj)
    return args.Get(0).(*TypeInfo)
}

// 测试中使用
func TestValidator(t *testing.T) {
    mockCache := new(MockTypeCache)
    mockCache.On("Get", mock.Anything).Return(testTypeInfo)

    validator := NewValidatorBuilder().
        WithTypeCache(mockCache).
        Build()

    // 测试...
}
```

---

## 文件组织

```
v2/
├── interface.go          # 核心接口定义
├── types.go             # 数据类型定义
├── error_collector.go   # 错误收集器实现
├── cache.go             # 缓存和注册管理器实现
├── strategy.go          # 验证策略实现
├── validator.go         # 核心验证器实现
├── builder.go           # 建造者实现
├── map_validator.go     # Map 验证器实现
├── validator_test.go    # 验证器测试
├── map_validator_test.go # Map 验证器测试
├── README.md            # 使用文档
└── ARCHITECTURE.md      # 架构设计文档（本文档）
```

**优势**：
- 按职责分离文件
- 每个文件职责清晰
- 易于定位和维护

---

## 性能优化

### 1. 类型信息缓存

避免重复的反射操作：

```go
type DefaultTypeCache struct {
    cache sync.Map // 线程安全的缓存
}

func (c *DefaultTypeCache) Get(obj any) *TypeInfo {
    typ := reflect.TypeOf(obj)

    // 尝试从缓存获取
    if cached, ok := c.cache.Load(typ); ok {
        return cached.(*TypeInfo)
    }

    // 缓存未命中，构建并缓存
    info := c.buildTypeInfo(obj, typ)
    c.cache.LoadOrStore(typ, info)
    return info
}
```

### 2. 线程安全

使用 sync.Map 和 sync.RWMutex 保证并发安全：

```go
type DefaultErrorCollector struct {
    errors []*FieldError
    mu     sync.RWMutex  // 读写锁
}
```

### 3. 策略顺序优化

按照验证成本排序策略：

```
1. RuleValidationStrategy    (快速，基于规则)
2. CustomValidationStrategy  (中等，业务逻辑)
3. NestedValidationStrategy  (较慢，递归验证)
```

---

## 扩展示例

### 添加数据库唯一性验证策略

```go
type DatabaseValidationStrategy struct {
    db        *sql.DB
    typeCache TypeCache
}

func (s *DatabaseValidationStrategy) Execute(obj any, scene Scene, collector ErrorCollector) bool {
    // 检查类型是否实现了 UniqueValidator 接口
    validator, ok := obj.(UniqueValidator)
    if !ok {
        return true
    }

    // 执行唯一性验证
    if err := validator.ValidateUnique(s.db); err != nil {
        collector.Add(NewFieldError("", "", "unique", "").
            WithMessage(err.Error()))
    }

    return true
}

// 使用
validator := NewValidatorBuilder().
    WithStrategy(&DatabaseValidationStrategy{db: db}).
    WithDefaultStrategies().
    Build()
```

---

## 总结

V2 架构通过应用 SOLID 原则和设计模式，实现了：

✅ **单一职责**：每个组件职责清晰
✅ **开放封闭**：易于扩展，无需修改现有代码
✅ **里氏替换**：接口实现可互换
✅ **依赖倒置**：依赖抽象而非具体实现
✅ **接口隔离**：细粒度接口设计

同时应用了：

✅ **策略模式**：灵活的验证逻辑
✅ **建造者模式**：灵活的配置方式
✅ **工厂方法模式**：统一的对象创建
✅ **单例模式**：全局默认验证器

最终实现了一个**高内聚低耦合、易于扩展和维护**的验证器架构。

