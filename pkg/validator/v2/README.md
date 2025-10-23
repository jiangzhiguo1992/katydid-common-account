# Validator V2 - 优化架构设计

## 概述

Validator V2 是基于 SOLID 设计原则和面向对象最佳实践重新设计的验证器架构，相比 V1 版本提供了更好的：

- ✅ **可扩展性**：通过策略模式轻松扩展新的验证逻辑
- ✅ **可维护性**：职责清晰，每个组件只负责单一功能
- ✅ **可测试性**：依赖接口而非具体实现，便于单元测试
- ✅ **可读性**：清晰的接口定义和命名规范
- ✅ **可复用性**：组件独立，可在不同场景下复用

---

## 架构设计原则

### 1. 单一职责原则（SRP）

每个组件只负责一项职责：

- **RuleProvider**: 只提供验证规则
- **CustomValidator**: 只执行自定义验证逻辑
- **ErrorCollector**: 只收集和管理错误
- **TypeCache**: 只缓存类型信息
- **ValidationStrategy**: 每个策略只负责一种验证逻辑

### 2. 开放封闭原则（OCP）

对扩展开放，对修改封闭：

- 通过**策略模式**扩展新的验证逻辑，无需修改核心验证器
- 通过**接口隔离**实现不同的验证行为
- 通过**建造者模式**灵活配置验证器

### 3. 里氏替换原则（LSP）

所有接口实现都可以互相替换：

- `TypeCache` 接口的不同实现可以互换
- `ValidationStrategy` 的不同策略可以互换
- `ErrorCollector` 的不同实现可以互换

### 4. 依赖倒置原则（DIP）

依赖抽象而非具体实现：

- 验证器依赖 `ValidationStrategy` 接口，而非具体策略
- 策略依赖 `TypeCache` 接口，而非具体缓存实现
- 所有组件都通过接口交互

### 5. 接口隔离原则（ISP）

接口细粒度设计，客户端不依赖不需要的接口：

- `ErrorReporter`: 只提供报告错误的方法
- `ErrorCollector`: 扩展 ErrorReporter，增加错误管理功能
- `Result`: 只提供结果查询功能
- `Validator`: 只提供验证功能

---

## 核心架构

### 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        Validator                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         Strategy Pattern (策略模式)                  │  │
│  │  ┌────────────────┐  ┌──────────────┐  ┌──────────┐ │  │
│  │  │ RuleValidation │  │   Custom     │  │  Nested  │ │  │
│  │  │   Strategy     │→ │  Validation  │→ │Validation│ │  │
│  │  │                │  │   Strategy   │  │ Strategy │ │  │
│  │  └────────────────┘  └──────────────┘  └──────────┘ │  │
│  └──────────────────────────────────────────────────────┘  │
│         ↓                  ↓                    ↓            │
│  ┌──────────┐      ┌──────────────┐    ┌──────────────┐   │
│  │TypeCache │      │ErrorCollector│    │RegistryMgr   │   │
│  └──────────┘      └──────────────┘    └──────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              ↓
                      ┌───────────────┐
                      │    Result     │
                      │  (验证结果)    │
                      └───────────────┘
```

### 核心接口

#### 1. Validator - 验证器接口

```go
type Validator interface {
    Validate(obj any, scene Scene) Result
}
```

**职责**：执行验证并返回结果

#### 2. RuleProvider - 规则提供者接口

```go
type RuleProvider interface {
    ProvideRules() map[Scene]FieldRules
}
```

**职责**：为模型提供场景化的字段验证规则

#### 3. CustomValidator - 自定义验证器接口

```go
type CustomValidator interface {
    ValidateCustom(scene Scene, reporter ErrorReporter)
}
```

**职责**：执行复杂的业务逻辑验证

#### 4. ValidationStrategy - 验证策略接口

```go
type ValidationStrategy interface {
    Execute(obj any, scene Scene, collector ErrorCollector) bool
}
```

**职责**：定义可插拔的验证策略

---

## 快速开始

### 基础使用

```go
package main

import (
    "fmt"
    "katydid-common-account/pkg/validator/v2"
)

// 定义模型
type User struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
    Age      int    `json:"age"`
}

// 实现 RuleProvider 接口 - 提供字段验证规则
func (u *User) ProvideRules() map[v2.Scene]v2.FieldRules {
    return map[v2.Scene]v2.FieldRules{
        v2.SceneCreate: {
            "Username": "required,min=3,max=20",
            "Email":    "required,email",
            "Password": "required,min=6",
            "Age":      "omitempty,gte=0,lte=150",
        },
        v2.SceneUpdate: {
            "Username": "omitempty,min=3,max=20",
            "Email":    "omitempty,email",
        },
    }
}

func main() {
    user := &User{
        Username: "john",
        Email:    "john@example.com",
        Password: "password123",
        Age:      25,
    }
    
    // 使用全局验证器
    result := v2.Validate(user, v2.SceneCreate)
    
    if !result.IsValid() {
        for _, err := range result.Errors() {
            fmt.Printf("字段 %s 验证失败: %s\n", err.Field, err.Message)
        }
        return
    }
    
    fmt.Println("验证通过！")
}
```

---

## 高级功能

### 1. 自定义验证逻辑

实现 `CustomValidator` 接口来添加复杂的业务逻辑验证：

```go
func (u *User) ValidateCustom(scene v2.Scene, reporter v2.ErrorReporter) {
    // 跨字段验证
    if u.Password != "" && u.Password != u.ConfirmPassword {
        reporter.ReportWithMessage(
            "User.ConfirmPassword",
            "password_mismatch",
            "",
            "密码和确认密码不一致",
        )
    }
    
    // 场景化验证
    if scene == v2.SceneCreate && u.Age < 18 {
        reporter.ReportWithMessage(
            "User.Age",
            "min_age",
            "18",
            "创建用户时年龄必须大于等于 18 岁",
        )
    }
    
    // 业务规则验证
    if u.Username == "admin" || u.Username == "root" {
        reporter.ReportWithMessage(
            "User.Username",
            "reserved_word",
            "",
            "用户名是保留字，不能使用",
        )
    }
}
```

### 2. Map 字段验证

验证动态 map 字段（如 Extras、Metadata）：

```go
type Product struct {
    Name     string         `json:"name"`
    Category string         `json:"category"`
    Extras   map[string]any `json:"extras"`
}

func (p *Product) ValidateCustom(scene v2.Scene, reporter v2.ErrorReporter) {
    if p.Extras == nil {
        return
    }
    
    // 根据分类验证不同的 Extras 字段
    switch p.Category {
    case "electronics":
        // 验证必填字段
        if err := v2.ValidateMapRequired(p.Extras, "brand", "warranty"); err != nil {
            reporter.ReportWithMessage("Product.Extras", "required_keys", "", err.Error())
        }
        
        // 验证字段类型和范围
        if err := v2.ValidateMapString(p.Extras, "brand", 2, 50); err != nil {
            reporter.ReportWithMessage("Product.Extras.brand", "invalid", "", err.Error())
        }
        
        if err := v2.ValidateMapInt(p.Extras, "warranty", 12, 60); err != nil {
            reporter.ReportWithMessage("Product.Extras.warranty", "invalid", "", err.Error())
        }
        
    case "clothing":
        if err := v2.ValidateMapRequired(p.Extras, "size", "color"); err != nil {
            reporter.ReportWithMessage("Product.Extras", "required_keys", "", err.Error())
        }
        
        // 自定义验证
        if err := v2.ValidateMapKey(p.Extras, "size", func(value any) error {
            size, ok := value.(string)
            if !ok {
                return fmt.Errorf("size 必须是字符串")
            }
            validSizes := map[string]bool{"S": true, "M": true, "L": true, "XL": true}
            if !validSizes[size] {
                return fmt.Errorf("size 必须是 S, M, L, XL 之一")
            }
            return nil
        }); err != nil {
            reporter.ReportWithMessage("Product.Extras.size", "invalid", "", err.Error())
        }
    }
}
```

### 3. 使用 MapValidator 进行结构化验证

```go
func (p *Product) ValidateCustom(scene v2.Scene, reporter v2.ErrorReporter) {
    if p.Extras == nil {
        return
    }
    
    // 创建 Map 验证器
    validator := v2.NewMapValidator().
        WithNamespace("Product.Extras").
        WithRequiredKeys("brand", "warranty").
        WithAllowedKeys("brand", "warranty", "color", "model").
        WithKeyValidator("warranty", func(value any) error {
            warranty, ok := value.(int)
            if !ok {
                return fmt.Errorf("warranty 必须是整数")
            }
            if warranty < 12 || warranty > 60 {
                return fmt.Errorf("warranty 必须在 12 到 60 个月之间")
            }
            return nil
        })
    
    // 执行验证
    errors := validator.Validate(p.Extras)
    
    // 添加错误到报告器
    for _, err := range errors {
        reporter.ReportWithMessage(err.Namespace, err.Tag, err.Param, err.Message)
    }
}
```

### 4. 建造者模式自定义验证器

```go
// 创建自定义配置的验证器
validator := v2.NewValidatorBuilder().
    WithMaxDepth(50).                    // 设置最大嵌套深度
    WithTypeCache(customCache).          // 使用自定义缓存
    WithDefaultStrategies().             // 使用默认策略
    Build()

result := validator.Validate(user, v2.SceneCreate)
```

### 5. 添加自定义策略

```go
// 定义自定义策略
type LoggingStrategy struct{}

func (s *LoggingStrategy) Execute(obj any, scene v2.Scene, collector v2.ErrorCollector) bool {
    fmt.Printf("Validating object of type %T in scene %s\n", obj, scene)
    return true // 继续执行后续策略
}

// 使用自定义策略
validator := v2.NewValidatorBuilder().
    WithStrategy(&LoggingStrategy{}).
    WithDefaultStrategies().
    Build()
```

---

## 验证结果处理

### Result 接口方法

```go
result := v2.Validate(user, v2.SceneCreate)

// 检查是否验证通过
if result.IsValid() {
    fmt.Println("验证通过")
}

// 获取所有错误
errors := result.Errors()

// 获取第一个错误
firstError := result.FirstError()

// 按字段筛选错误
usernameErrors := result.ErrorsByField("username")

// 按标签筛选错误
requiredErrors := result.ErrorsByTag("required")

// 实现 error 接口
fmt.Println(result.Error())
```

---

## 场景化验证

V2 支持灵活的场景定义：

```go
const (
    SceneCreate v2.Scene = "create"
    SceneUpdate v2.Scene = "update"
    SceneDelete v2.Scene = "delete"
    SceneQuery  v2.Scene = "query"
    
    // 自定义场景
    SceneImport v2.Scene = "import"
    SceneExport v2.Scene = "export"
)

func (u *User) ProvideRules() map[v2.Scene]v2.FieldRules {
    return map[v2.Scene]v2.FieldRules{
        SceneCreate: {
            "Username": "required,min=3,max=20",
            "Email":    "required,email",
            "Password": "required,min=6",
        },
        SceneUpdate: {
            "Username": "omitempty,min=3,max=20",
            "Email":    "omitempty,email",
        },
        SceneImport: {
            "Username": "required",
            "Email":    "required,email",
            // 导入时可能不需要密码
        },
    }
}
```

---

## 性能优化

### 1. 类型缓存

类型信息会被自动缓存，避免重复的反射操作：

```go
validator := v2.NewValidator()

// 第一次验证：构建类型信息并缓存
result1 := validator.Validate(user1, v2.SceneCreate)

// 后续验证：使用缓存，性能提升
result2 := validator.Validate(user2, v2.SceneCreate)
```

### 2. 并发安全

V2 验证器是线程安全的，可以在多个 goroutine 中并发使用：

```go
validator := v2.NewValidator()

var wg sync.WaitGroup
for _, user := range users {
    wg.Add(1)
    go func(u *User) {
        defer wg.Done()
        result := validator.Validate(u, v2.SceneCreate)
        // 处理结果...
    }(user)
}
wg.Wait()
```

### 3. 清除缓存

在测试或需要重新加载类型信息时：

```go
// 清除默认验证器的缓存
v2.ClearCache()

// 或清除自定义验证器的缓存
validator.ClearCache()
```

---

## 设计模式应用

### 1. 策略模式（Strategy Pattern）

不同的验证逻辑通过策略实现：

- `RuleValidationStrategy`: 基于规则的验证
- `CustomValidationStrategy`: 自定义业务逻辑验证
- `NestedValidationStrategy`: 嵌套结构验证

### 2. 建造者模式（Builder Pattern）

灵活配置验证器：

```go
validator := v2.NewValidatorBuilder().
    WithMaxDepth(50).
    WithDefaultStrategies().
    Build()
```

### 3. 工厂方法模式（Factory Method Pattern）

创建各种组件：

- `NewValidator()`: 创建验证器
- `NewErrorCollector()`: 创建错误收集器
- `NewMapValidator()`: 创建 Map 验证器

### 4. 单例模式（Singleton Pattern）

全局默认验证器：

```go
result := v2.Validate(user, v2.SceneCreate) // 使用全局单例
```

---

## 与 V1 版本对比

| 特性 | V1 | V2 |
|------|----|----|
| **接口设计** | 混合职责 | 职责分离，符合 ISP |
| **可扩展性** | 通过修改代码扩展 | 通过策略模式扩展 |
| **依赖管理** | 依赖具体实现 | 依赖抽象接口（DIP） |
| **测试性** | 较难 mock | 易于 mock 和测试 |
| **配置灵活性** | 有限 | 建造者模式，高度灵活 |
| **错误处理** | 返回切片 | Result 接口，功能丰富 |
| **代码组织** | 单一大文件 | 按职责分文件 |

---

## 最佳实践

### 1. 接口实现建议

- 简单的格式验证使用 `RuleProvider`
- 复杂的业务逻辑使用 `CustomValidator`
- Map 字段验证使用 `MapValidator` 或便捷函数

### 2. 错误消息建议

```go
// 提供友好的中文错误消息
reporter.ReportWithMessage(
    "User.Email",
    "email",
    "",
    "邮箱格式不正确，请输入有效的邮箱地址",
)
```

### 3. 性能建议

- 对于高频验证，创建独立的验证器实例而非使用全局单例
- 合理设置最大嵌套深度，防止过深的递归
- 定期清理不再使用的类型缓存

### 4. 测试建议

```go
func TestUserValidation(t *testing.T) {
    // 使用独立的验证器实例，避免测试间干扰
    validator := v2.NewValidator()
    
    user := &User{...}
    result := validator.Validate(user, v2.SceneCreate)
    
    if !result.IsValid() {
        t.Errorf("Expected validation to pass, but got: %v", result.Error())
    }
}
```

---

## API 参考

### 便捷函数

```go
// 使用默认验证器
func Validate(obj any, scene Scene) Result

// 清除默认验证器缓存
func ClearCache()
```

### Map 验证便捷函数

```go
func ValidateMapRequired(data map[string]any, keys ...string) error
func ValidateMapString(data map[string]any, key string, minLen, maxLen int) error
func ValidateMapInt(data map[string]any, key string, min, max int) error
func ValidateMapFloat(data map[string]any, key string, min, max float64) error
func ValidateMapBool(data map[string]any, key string) error
func ValidateMapKey(data map[string]any, key string, validator func(value any) error) error
```

---

## 常见验证标签

与 V1 相同，支持 go-playground/validator 的所有标签：

```
required      - 必填
omitempty     - 可选
min=N         - 最小长度/值
max=N         - 最大长度/值
len=N         - 长度等于
email         - 邮箱格式
url           - URL 格式
numeric       - 数字
alpha         - 字母
alphanum      - 字母和数字
gt=N          - 大于
gte=N         - 大于等于
lt=N          - 小于
lte=N         - 小于等于
```

---

## 总结

Validator V2 通过应用 SOLID 原则和设计模式，提供了一个：

- ✅ **高内聚低耦合**的验证架构
- ✅ **易于扩展和维护**的代码结构
- ✅ **便于测试**的接口设计
- ✅ **功能完整**的验证解决方案

同时保持了与 V1 版本相同的功能，平滑迁移路径清晰。

---

## 许可证

本项目遵循项目根目录的许可证。

