# v5 到 v6 迁移指南

## 概述

v6 在 v5 的基础上进行了全面的架构重构，严格遵循 SOLID 原则，提供了更好的可扩展性、可维护性和可测试性。

## 主要变化

### 1. 接口重命名

| v5 | v6 | 说明 |
|----|----|------|
| `RuleValidation` | `RuleProvider` | 更明确地表达"提供规则"的职责 |
| `ValidateRules()` | `GetRules()` | 更符合 Getter 命名规范 |
| `ValidationError` | `ValidationError` | 保持不变 |

### 2. 创建验证器的方式

**v5:**
```go
engine := v5.NewValidatorEngine(
    v5.WithStrategies(...),
    v5.WithMaxErrors(100),
)
```

**v6:**
```go
validator := v6.NewValidator().
    WithStrategies(...).
    WithMaxErrors(100).
    BuildDefault()
```

### 3. 验证方法

**v5:**
```go
err := engine.Validate(user, scene)
if err != nil {
    // 处理错误
}
```

**v6:**
```go
// 方式1：简单用法
err := validator.Validate(user, scene)

// 方式2：高级用法
req := core.NewValidationRequest(user, scene).
    WithFields("name", "email")
result, err := validator.ValidateWithRequest(req)
```

## 逐步迁移

### 第 1 步：更新导入

```go
// 旧的导入
import v5 "katydid-common-account/pkg/validator/v5"

// 新的导入
import (
    "katydid-common-account/pkg/validator/v6"
    "katydid-common-account/pkg/validator/v6/core"
)
```

### 第 2 步：更新接口实现

**v5:**
```go
type User struct {
    Name string
}

func (u *User) ValidateRules() map[v5.Scene]map[string]string {
    return map[v5.Scene]map[string]string{
        SceneCreate: {
            "name": "required",
        },
    }
}
```

**v6:**
```go
type User struct {
    Name string
}

func (u *User) GetRules() map[core.Scene]map[string]string {
    return map[core.Scene]map[string]string{
        SceneCreate: {
            "name": "required",
        },
    }
}
```

### 第 3 步：更新场景定义

**v5:**
```go
const (
    SceneCreate v5.Scene = 1 << iota
    SceneUpdate
)
```

**v6:**
```go
const (
    SceneCreate core.Scene = 1 << iota
    SceneUpdate
)
```

### 第 4 步：更新验证器创建

**v5:**
```go
validator := v5.NewValidatorFactory().CreateDefault()
```

**v6:**
```go
validator := v6.NewValidator().BuildDefault()
```

### 第 5 步：更新验证调用

基本用法无需改动：

```go
err := validator.Validate(user, SceneCreate)
```

## 新功能

### 1. 插件系统

v6 新增了插件系统，可以轻松扩展功能：

```go
import "katydid-common-account/pkg/validator/v6/plugin"

validator := v6.NewValidator().
    WithPlugins(
        plugin.NewLoggingPlugin(),
    ).
    BuildDefault()
```

### 2. 事件监听

```go
type MyListener struct{}

func (l *MyListener) OnEvent(event core.ValidationEvent) {
    // 处理事件
}

validator := v6.NewValidator().
    WithListeners(&MyListener{}).
    BuildDefault()
```

### 3. 高级验证请求

```go
// 只验证指定字段
req := core.NewValidationRequest(user, SceneUpdate).
    WithFields("name", "email")

result, err := validator.ValidateWithRequest(req)

// 排除某些字段
req := core.NewValidationRequest(user, SceneCreate).
    WithExcludeFields("password")
```

## 兼容性说明

### 完全兼容的部分

- ✅ 场景定义（Scene）
- ✅ 基本验证调用
- ✅ 错误处理
- ✅ 验证规则语法

### 需要修改的部分

- ⚠️ 接口名称（`RuleValidation` → `RuleProvider`）
- ⚠️ 方法名称（`ValidateRules()` → `GetRules()`）
- ⚠️ 创建验证器的方式

### 不兼容的部分

- ❌ 直接访问 `ValidatorEngine` 的内部字段
- ❌ 自定义的 v5 特有扩展

## 迁移检查清单

- [ ] 更新导入路径
- [ ] 重命名接口实现（`RuleValidation` → `RuleProvider`）
- [ ] 重命名方法（`ValidateRules()` → `GetRules()`）
- [ ] 更新场景类型（`v5.Scene` → `core.Scene`）
- [ ] 更新验证器创建代码
- [ ] 运行测试确保功能正常
- [ ] 考虑使用新功能（插件、事件监听等）

## 示例：完整迁移

### v5 代码

```go
package main

import (
    v5 "katydid-common-account/pkg/validator/v5"
)

const SceneCreate v5.Scene = 1

type User struct {
    Name string `json:"name"`
}

func (u *User) ValidateRules() map[v5.Scene]map[string]string {
    return map[v5.Scene]map[string]string{
        SceneCreate: {
            "name": "required,min=2",
        },
    }
}

func main() {
    validator := v5.NewValidatorFactory().CreateDefault()
    
    user := &User{Name: "Alice"}
    
    if err := validator.Validate(user, SceneCreate); err != nil {
        panic(err)
    }
}
```

### v6 代码

```go
package main

import (
    "katydid-common-account/pkg/validator/v6"
    "katydid-common-account/pkg/validator/v6/core"
)

const SceneCreate core.Scene = 1

type User struct {
    Name string `json:"name"`
}

func (u *User) GetRules() map[core.Scene]map[string]string {
    return map[core.Scene]map[string]string{
        SceneCreate: {
            "name": "required,min=2",
        },
    }
}

func main() {
    validator := v6.NewValidator().BuildDefault()
    
    user := &User{Name: "Alice"}
    
    if err := validator.Validate(user, SceneCreate); err != nil {
        panic(err)
    }
}
```

## 性能对比

v6 继承了 v5 的所有性能优化，并进一步改进：

| 指标 | v5 | v6 | 说明 |
|-----|----|----|------|
| 字段访问 | O(1) | O(1) | 相同的缓存策略 |
| 内存分配 | 优化 | 更优 | 更好的对象池 |
| 可扩展性 | 有限 | 优秀 | 插件机制 |

## 常见问题

### Q: 是否必须迁移到 v6？

A: 不是必须的。v5 仍然可以继续使用。但如果你的项目需要更好的可扩展性和可维护性，建议迁移到 v6。

### Q: 迁移工作量大吗？

A: 对于大多数项目，迁移工作量很小，主要是重命名接口和方法。核心验证逻辑无需修改。

### Q: v6 性能如何？

A: v6 继承了 v5 的所有性能优化，性能相当或更好。

### Q: 可以同时使用 v5 和 v6 吗？

A: 可以。两者可以在同一个项目中共存，方便渐进式迁移。

## 获取帮助

如有问题，请：

1. 查看 [README.md](./README.md)
2. 查看 [ARCHITECTURE.md](./ARCHITECTURE.md)
3. 提交 Issue

## 总结

v6 是 v5 的全面升级，虽然引入了一些接口变化，但迁移成本较低，收益明显。建议新项目直接使用 v6，现有项目可以考虑渐进式迁移。

