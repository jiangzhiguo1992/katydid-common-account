# Validator v5 架构设计文档

## 设计原则

本版本严格遵循以下设计原则：

### 1. SOLID 原则

#### 单一职责原则 (SRP)
- **ValidatorEngine**: 只负责协调验证流程
- **RuleExecutor**: 只负责执行规则验证
- **ErrorCollector**: 只负责收集和管理错误
- **TypeRegistry**: 只负责类型信息的注册和缓存
- **SceneMatcher**: 只负责场景匹配逻辑

#### 开放封闭原则 (OCP)
- 通过 `ValidationStrategy` 接口支持扩展新的验证策略
- 通过 `ErrorFormatter` 接口支持自定义错误格式化
- 通过 `CacheStrategy` 接口支持不同的缓存实现

#### 里氏替换原则 (LSP)
- 所有 `ValidationStrategy` 的实现都可以互相替换
- 所有 `ErrorCollector` 的实现都可以互相替换

#### 接口隔离原则 (ISP)
- `RuleProvider`: 只提供规则定义
- `BusinessValidator`: 只处理业务逻辑验证
- `LifecycleHooks`: 只处理生命周期回调
- 避免了 v4 中接口功能过于复杂的问题

#### 依赖倒置原则 (DIP)
- 高层模块 `ValidatorEngine` 依赖抽象接口，不依赖具体实现
- 所有依赖都通过构造函数注入

### 2. 高内聚低耦合

#### 高内聚
- 每个模块专注于单一领域
- 相关功能聚合在同一模块

#### 低耦合
- 模块间通过接口通信
- 使用事件驱动模式解耦组件
- 依赖注入减少硬编码依赖

### 3. 设计模式应用

- **策略模式**: `ValidationStrategy` 支持不同验证策略
- **工厂模式**: `ValidatorFactory` 创建配置好的验证器
- **建造者模式**: `ValidatorBuilder` 构建复杂配置
- **观察者模式**: `ValidationEvent` 支持生命周期监听
- **适配器模式**: `ErrorAdapter` 适配第三方库错误
- **对象池模式**: 内存优化
- **责任链模式**: `ValidationPipeline` 串联多个验证器
- **单例模式**: 全局默认验证器

## 核心架构

```
┌─────────────────────────────────────────────────────────────┐
│                      ValidatorEngine                         │
│  (协调者 - 负责整体验证流程编排)                              │
└─────────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐   ┌──────────────┐
│ TypeRegistry │    │ SceneMatcher │   │ErrorCollector│
│  (类型缓存)   │    │  (场景匹配)   │   │  (错误收集)  │
└──────────────┘    └──────────────┘   └──────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────┐
│                   ValidationStrategy                         │
│              (验证策略接口 - 支持扩展)                         │
└─────────────────────────────────────────────────────────────┘
        │
        ├─────────────┬─────────────┬─────────────┐
        ▼             ▼             ▼             ▼
┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐
│RuleStrategy│ │CustomStrat │ │NestedStrat │ │ MapStrat  │
│ (规则验证) │ │(自定义验证)│ │(嵌套验证)  │ │(Map验证)  │
└────────────┘ └────────────┘ └────────────┘ └────────────┘
```

## 模块职责

### 1. 核心引擎层
- **ValidatorEngine**: 验证流程编排器
- **ValidatorBuilder**: 验证器构建器
- **ValidatorFactory**: 验证器工厂

### 2. 策略执行层
- **ValidationStrategy**: 验证策略接口
- **RuleStrategy**: 规则验证策略
- **CustomStrategy**: 自定义验证策略
- **NestedStrategy**: 嵌套验证策略
- **MapStrategy**: Map 验证策略

### 3. 支持服务层
- **TypeRegistry**: 类型注册与缓存
- **SceneMatcher**: 场景匹配服务
- **ErrorCollector**: 错误收集服务
- **ErrorFormatter**: 错误格式化服务

### 4. 基础设施层
- **ObjectPool**: 对象池管理
- **Cache**: 缓存接口与实现
- **Logger**: 日志接口

## 接口设计

### 验证接口（业务层实现）

```go
// RuleProvider - 提供验证规则
type RuleProvider interface {
    GetRules(scene Scene) map[string]string
}

// BusinessValidator - 业务逻辑验证
type BusinessValidator interface {
    ValidateBusiness(ctx *ValidationContext) error
}

// LifecycleHooks - 生命周期钩子
type LifecycleHooks interface {
    BeforeValidation(ctx *ValidationContext) error
    AfterValidation(ctx *ValidationContext) error
}
```

### 策略接口（框架层实现）

```go
// ValidationStrategy - 验证策略接口
type ValidationStrategy interface {
    Name() string
    Validate(target any, ctx *ValidationContext) error
    Priority() int
}

// ErrorCollector - 错误收集器接口
type ErrorCollector interface {
    AddError(err *FieldError)
    GetErrors() []*FieldError
    HasErrors() bool
    Clear()
}
```

## 扩展点

1. **自定义验证策略**: 实现 `ValidationStrategy` 接口
2. **自定义错误格式**: 实现 `ErrorFormatter` 接口
3. **自定义缓存**: 实现 `CacheStrategy` 接口
4. **生命周期监听**: 实现 `ValidationListener` 接口

## 优势对比

### v4 vs v5

| 特性 | v4 | v5 |
|------|----|----|
| 单一职责 | ❌ 职责混乱 | ✅ 职责清晰 |
| 依赖注入 | ❌ 硬编码依赖 | ✅ 完全依赖注入 |
| 可测试性 | ⚠️ 较差 | ✅ 优秀 |
| 可扩展性 | ⚠️ 有限 | ✅ 高度可扩展 |
| 接口隔离 | ❌ 接口过大 | ✅ 接口精简 |
| 代码复用 | ⚠️ 一般 | ✅ 高度复用 |

## 迁移指南

v4 到 v5 的迁移路径请参考 MIGRATION_GUIDE.md

