# Validator v5_refactored - 企业级验证器框架（重构版）

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.18-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

一个严格遵循 SOLID 原则、完全解耦、高度可扩展的 Go 验证器框架。

---

## 🎯 核心特性

### ✅ SOLID 原则

- **单一职责 (SRP)**：每个组件只负责一件事
- **开放封闭 (OCP)**：通过接口扩展，无需修改代码
- **里氏替换 (LSP)**：所有实现可互相替换
- **接口隔离 (ISP)**：细粒度接口，避免臃肿
- **依赖倒置 (DIP)**：完全面向接口编程

### ✅ 设计模式

- **策略模式**：灵活的验证策略
- **观察者模式**：事件驱动架构
- **工厂模式**：统一创建逻辑
- **建造者模式**：流畅的 API
- **责任链模式**：策略链执行
- **对象池模式**：性能优化

### ✅ 架构特点

- **高内聚低耦合**：组件独立，依赖接口
- **事件驱动**：通过事件总线解耦
- **可测试**：所有组件可独立测试
- **可扩展**：通过接口轻松扩展
- **高性能**：对象池、多级缓存

---

## 🚀 快速开始

### 基础使用

```go
package main

import (
    "fmt"
    v5 "your-project/pkg/validator/v5_refactored"
)

// 定义模型
type User struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
    Age      int    `json:"age"`
}

// 实现 RuleProvider 接口
func (u *User) GetRules(scene v5.Scene) map[string]string {
    switch scene {
    case v5.SceneCreate:
        return map[string]string{
            "username": "required,min=3,max=20",
            "email":    "required,email",
            "password": "required,min=6",
            "age":      "required,min=18",
        }
    case v5.SceneUpdate:
        return map[string]string{
            "username": "omitempty,min=3,max=20",
            "email":    "omitempty,email",
            "password": "omitempty,min=6",
        }
    default:
        return nil
    }
}

func main() {
    user := &User{
        Username: "john",
        Email:    "john@example.com",
        Password: "123456",
        Age:      25,
    }

    // 使用默认验证器
    if err := v5.Validate(user, v5.SceneCreate); err != nil {
        fmt.Printf("验证失败: %v\n", err)
        return
    }

    fmt.Println("验证通过！")
}
```

### 高级用法 - 自定义配置

```go
package main

import (
    v5 "your-project/pkg/validator/v5_refactored"
)

func main() {
    // 使用建造者模式创建自定义验证器
    validator := v5.NewBuilder().
        WithEventBus(v5.NewAsyncEventBus(4, 100)).         // 异步事件总线
        WithRegistry(v5.NewMultiLevelTypeRegistry(100)).    // 多级缓存
        WithErrorFormatter(v5.NewChineseErrorFormatter()). // 中文错误
        WithMaxErrors(50).                                  // 最大错误数
        WithMaxDepth(10).                                   // 最大嵌套深度
        Build()

    // 使用自定义验证器
    user := &User{Username: "test"}
    if err := validator.Validate(user, v5.SceneCreate); err != nil {
        fmt.Printf("验证失败: %v\n", err)
    }
}
```

### 业务验证

```go
// 实现 BusinessValidator 接口
func (u *User) ValidateBusiness(scene v5.Scene, ctx *v5.ValidationContext, collector v5.ErrorCollector) error {
    // 复杂的业务逻辑验证
    if u.Username == "admin" {
        collector.Add(v5.NewFieldError("username", "reserved").
            WithMessage("用户名 'admin' 已被保留"))
    }

    // 跨字段验证
    if u.Age < 18 && u.Password == "" {
        collector.Add(v5.NewFieldError("password", "required").
            WithMessage("未成年用户必须设置密码"))
    }

    // 数据库检查
    if u.checkUsernameExists() {
        collector.Add(v5.NewFieldError("username", "unique").
            WithMessage("用户名已存在"))
    }

    return nil
}

func (u *User) checkUsernameExists() bool {
    // 数据库查询逻辑
    return false
}
```

### 生命周期钩子

```go
// 实现 LifecycleHooks 接口
func (u *User) BeforeValidation(ctx *v5.ValidationContext) error {
    // 验证前的预处理
    u.Username = strings.TrimSpace(u.Username)
    u.Email = strings.ToLower(u.Email)
    return nil
}

func (u *User) AfterValidation(ctx *v5.ValidationContext) error {
    // 验证后的处理
    fmt.Println("验证完成")
    return nil
}
```

### 事件监听

```go
// 定义监听器
type ValidationLogger struct{}

func (l *ValidationLogger) OnEvent(event v5.Event) {
    switch event.Type() {
    case v5.EventValidationStart:
        fmt.Println("开始验证")
    case v5.EventValidationEnd:
        fmt.Println("验证结束")
    case v5.EventErrorOccurred:
        fmt.Printf("发生错误: %v\n", event.Data())
    }
}

func (l *ValidationLogger) EventTypes() []v5.EventType {
    // 返回空表示监听所有事件
    return nil
}

// 订阅事件
func main() {
    eventBus := v5.NewSyncEventBus()
    eventBus.Subscribe(&ValidationLogger{})

    validator := v5.NewBuilder().
        WithEventBus(eventBus).
        Build()

    validator.Validate(&User{}, v5.SceneCreate)
}
```

---

## 📊 架构对比

### v5 vs v5_refactored

| 维度 | v5 | v5_refactored | 改进 |
|------|----|--------------|----- |
| **单一职责** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | Engine 职责拆分为 5 个组件 |
| **开放封闭** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 更多扩展点和接口 |
| **接口隔离** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 细粒度接口设计 |
| **依赖倒置** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 完全依赖接口 |
| **可测试性** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 组件完全独立 |
| **可扩展性** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 插件式架构 |
| **事件驱动** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 完整的事件总线 |
| **并发支持** | ❌ | ✅ | 支持并发管道执行器 |

### 核心改进

#### 1. 职责分离

**v5**：
```go
type ValidatorEngine struct {
    validator      *validator.Validate
    sceneMatcher   SceneMatcher
    registry       Registry
    strategies     []ValidationStrategy
    listeners      []ValidationListener  // 职责混杂
    errorFormatter ErrorFormatter
    maxDepth       int
    maxErrors      int
}
```

**v5_refactored**：
```go
type ValidatorEngine struct {
    pipeline         PipelineExecutor    // 策略执行
    eventBus         EventBus            // 事件管理
    hookManager      HookManager         // 钩子管理
    registry         TypeRegistry        // 类型缓存
    collectorFactory ErrorCollectorFactory // 错误收集
}
```

#### 2. 事件驱动

**v5**：直接调用监听器
```go
for _, listener := range e.listeners {
    listener.OnValidationStart(ctx)
}
```

**v5_refactored**：通过事件总线
```go
e.eventBus.Publish(NewBaseEvent(EventValidationStart, ctx))
```

#### 3. 依赖注入

**v5_refactored** 完全通过构造函数注入依赖：
```go
func NewValidatorEngine(
    pipeline PipelineExecutor,      // 接口
    eventBus EventBus,               // 接口
    hookManager HookManager,         // 接口
    registry TypeRegistry,           // 接口
    collectorFactory ErrorCollectorFactory, // 接口
    errorFormatter ErrorFormatter,   // 接口
) *ValidatorEngine
```

---

## 🧪 测试支持

### 组件独立测试

```go
func TestPipelineExecutor(t *testing.T) {
    executor := v5.NewDefaultPipelineExecutor()
    
    // 添加 mock 策略
    executor.AddStrategy(&MockStrategy{})
    
    ctx := v5.AcquireContext(v5.SceneCreate, &User{})
    collector := v5.NewDefaultErrorCollector(10)
    
    err := executor.Execute(&User{}, ctx, collector)
    
    assert.NoError(t, err)
    assert.False(t, collector.HasErrors())
}
```

### Mock 依赖

```go
type MockEventBus struct {
    events []v5.Event
}

func (m *MockEventBus) Publish(event v5.Event) {
    m.events = append(m.events, event)
}

func TestValidatorEngine(t *testing.T) {
    mockBus := &MockEventBus{}
    
    validator := v5.NewValidatorEngine(
        v5.NewDefaultPipelineExecutor(),
        mockBus,  // 注入 Mock
        nil, nil, nil, nil,
    )
    
    validator.Validate(&User{}, v5.SceneCreate)
    
    // 验证事件发布
    assert.Equal(t, 2, len(mockBus.events))
}
```

---

## 📦 组件说明

### 核心组件

1. **ValidatorEngine**：验证引擎，协调各组件
2. **PipelineExecutor**：管道执行器，编排策略
3. **EventBus**：事件总线，发布订阅
4. **HookManager**：钩子管理器，生命周期
5. **ErrorCollector**：错误收集器，错误管理
6. **TypeRegistry**：类型注册表，类型缓存

### 可选组件

- `AsyncEventBus`：异步事件总线
- `ConcurrentPipelineExecutor`：并发管道执行器
- `MultiLevelTypeRegistry`：多级缓存注册表
- `ConcurrentErrorCollector`：并发错误收集器

---

## 🎨 扩展示例

### 自定义验证策略

```go
type CustomStrategy struct{}

func (s *CustomStrategy) Type() v5.StrategyType {
    return v5.StrategyTypeCustom
}

func (s *CustomStrategy) Priority() int8 {
    return 50  // 优先级
}

func (s *CustomStrategy) Name() string {
    return "custom"
}

func (s *CustomStrategy) Validate(target any, ctx *v5.ValidationContext, collector v5.ErrorCollector) error {
    // 自定义验证逻辑
    return nil
}

// 使用自定义策略
pipeline := v5.NewDefaultPipelineExecutor()
pipeline.AddStrategy(&CustomStrategy{})
```

### 自定义事件监听器

```go
type MetricsListener struct {
    validationCount int64
    errorCount      int64
}

func (l *MetricsListener) OnEvent(event v5.Event) {
    switch event.Type() {
    case v5.EventValidationStart:
        atomic.AddInt64(&l.validationCount, 1)
    case v5.EventErrorOccurred:
        atomic.AddInt64(&l.errorCount, 1)
    }
}

func (l *MetricsListener) EventTypes() []v5.EventType {
    return []v5.EventType{
        v5.EventValidationStart,
        v5.EventErrorOccurred,
    }
}
```

---

## 📖 详细文档

- [架构设计](ARCHITECTURE.md) - 完整的架构设计文档
- [使用示例](EXAMPLES.md) - 更多使用示例
- [迁移指南](MIGRATION.md) - 从 v5 迁移到 v5_refactored

---

## ⚡ 性能优化

- ✅ 对象池减少内存分配
- ✅ 多级缓存提升查询速度
- ✅ 并发执行支持
- ✅ 事件异步处理

---

## 🔧 配置建议

### 生产环境

```go
validator := v5.NewBuilder().
    WithPipeline(v5.NewConcurrentPipelineExecutor(8)).     // 并发执行
    WithEventBus(v5.NewAsyncEventBus(4, 1000)).            // 异步事件
    WithRegistry(v5.NewMultiLevelTypeRegistry(200)).       // 多级缓存
    WithErrorCollectorFactory(
        v5.NewDefaultErrorCollectorFactory(true),          // 并发收集器
    ).
    WithMaxErrors(100).
    WithMaxDepth(20).
    Build()
```

### 开发环境

```go
validator := v5.NewBuilder().
    WithEventBus(v5.NewSyncEventBus()).                    // 同步事件（便于调试）
    WithErrorFormatter(v5.NewChineseErrorFormatter()).     // 中文错误
    Build()
```

---

## 📝 总结

v5_refactored 是一个真正意义上的**企业级验证器框架**，具有：

- ✅ 严格遵循 SOLID 原则
- ✅ 完全解耦的组件设计
- ✅ 事件驱动架构
- ✅ 高度可扩展
- ✅ 易于测试
- ✅ 生产就绪

适用于：
- 企业级应用
- 微服务架构
- 复杂业务逻辑
- 长期维护的项目
- 团队协作开发

## 📄 License

MIT License

