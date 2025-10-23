# Validator 验证器

一个功能强大、灵活且高性能的 Go 验证器库，基于 `go-playground/validator/v10` 封装，提供场景化验证、嵌套验证、自定义验证等多种验证方式。

## 目录

- [概述](#概述)
- [核心特性](#核心特性)
- [快速开始](#快速开始)
- [验证接口](#验证接口)
  - [RuleProvider - 字段规则](#ruleprovider---字段规则)
  - [BusinessValidator - 业务验证](#businessvalidator---业务验证)
  - [CrossFieldValidator - 跨字段验证](#crossfieldvalidator---跨字段验证)
- [验证场景](#验证场景)
- [Map 验证](#map-验证)
- [嵌套验证](#嵌套验证)
- [自动注册机制](#自动注册机制)
- [完整使用示例](#完整使用示例)
- [性能优化](#性能优化)
- [最佳实践](#最佳实践)
- [API 参考](#api-参考)
- [常见验证标签](#常见验证标签)

---

## 概述

本验证器提供了一套完整的验证解决方案，支持：

- ✅ **场景化验证**：针对不同业务场景（创建、更新、删除等）使用不同的验证规则
- ✅ **接口驱动**：通过实现接口来定义验证规则，灵活且易于维护
- ✅ **嵌套验证**：自动递归验证嵌套的结构体字段
- ✅ **自动注册**：首次使用时自动注册验证规则，无需手动调用
- ✅ **高性能**：使用类型缓存和规则缓存机制，提升验证性能
- ✅ **线程安全**：可在多个 goroutine 中并发使用
- ✅ **Map 验证**：灵活验证动态 map[string]any 字段

---

## 核心特性

### 1. 场景化验证

通过 `ValidateScene` 定义不同的验证场景，同一个模型在不同场景下可以有不同的验证规则。

```go
const (
    SceneCreate ValidateScene = "create"
    SceneUpdate ValidateScene = "update"
)

type User struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

func (u *User) Rules() map[ValidateScene]map[string]string {
    return map[ValidateScene]map[string]string{
        SceneCreate: {
            "Username": "required,min=3,max=20",
            "Email":    "required,email",
            "Password": "required,min=6",
        },
        SceneUpdate: {
            "Username": "omitempty,min=3,max=20",
            "Email":    "omitempty,email",
            "Password": "omitempty,min=6",
        },
    }
}

// 使用
errs := validator.Validate(user, SceneCreate)
```

### 2. 三种验证接口

验证器提供三种清晰职责的接口，覆盖所有验证场景：

| 接口 | 用途 | 使用场景 |
|-----|------|---------|
| **RuleProvider** | 字段格式验证 | required, min, max, email 等标签验证 |
| **BusinessValidator** | 业务逻辑验证 | 数据库检查、Map验证、复杂条件判断 |
| **CrossFieldValidator** | 跨字段关系验证 | 密码确认、日期范围、字段间依赖 |

---

## 快速开始

### 最简单的示例

```go
package main

import (
    "fmt"
    "katydid-common-account/pkg/validator"
)

type User struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Age      int    `json:"age"`
}

// 实现 RuleProvider 接口
func (u *User) Rules() map[validator.ValidateScene]map[string]string {
    return map[validator.ValidateScene]map[string]string{
        "create": {
            "Username": "required,min=3,max=20",
            "Email":    "required,email",
            "Age":      "omitempty,gte=0,lte=150",
        },
    }
}

func main() {
    user := &User{
        Username: "john",
        Email:    "john@example.com",
        Age:      25,
    }

    errs := validator.Validate(user, "create")
    if errs != nil {
        for _, err := range errs {
            fmt.Printf("字段 %s 验证失败: %s\n", err.Field, err.Message)
        }
        return
    }
    fmt.Println("验证通过！")
}
```

### 使用验证器实例

```go
// 方式1: 使用默认的全局验证器
errs := validator.Validate(user, "create")

// 方式2: 创建独立的验证器实例
v := validator.New()
errs := v.Validate(user, "create")
```

---

## 验证接口

### RuleProvider - 字段规则

**用途**：为模型字段提供基础的格式验证规则（必填、长度、格式等）

**接口定义**：
```go
type RuleProvider interface {
    Rules() map[ValidateScene]map[string]string
}
```

**示例**：
```go
func (u *User) Rules() map[ValidateScene]map[string]string {
    return map[ValidateScene]map[string]string{
        "create": {
            "Username": "required,min=3,max=20,alphanum",
            "Email":    "required,email",
            "Password": "required,min=6,max=20",
        },
        "update": {
            "Username": "omitempty,min=3,max=20,alphanum",
            "Email":    "omitempty,email",
            "Password": "omitempty,min=6,max=20",
        },
    }
}
```

### BusinessValidator - 业务验证

**用途**：实现无法用标签表达的复杂业务验证逻辑

**接口定义**：
```go
type BusinessValidator interface {
    Validate(scene ValidateScene) []*FieldError
}
```

**示例**：
```go
func (p *Product) Validate(scene ValidateScene) []*FieldError {
    var errors []*FieldError
    
    if scene == "create" && p.Category == "electronics" {
        if err := validator.ValidateMapMustHaveKeys(p.Extras, "brand"); err != nil {
            errors = append(errors, validator.NewFieldError(
                "extras.brand", "电子产品必须提供品牌信息", nil, nil,
            ))
        }
    }
    
    if len(errors) > 0 {
        return errors
    }
    return nil
}
```

### CrossFieldValidator - 跨字段验证

**用途**：验证多个字段之间的关系和约束（自动注册）

**接口定义**：
```go
type CrossFieldValidator interface {
    CrossFieldValidation(sl StructLevel)
}
```

**示例**：
```go
func (u *User) CrossFieldValidation(sl validator.StructLevel) {
    // 密码和确认密码必须一致
    if u.Password != u.ConfirmPassword {
        sl.ReportError(u.ConfirmPassword, "ConfirmPassword", 
            "confirm_password", "password_mismatch", "")
    }
    
    // 未成年用户名必须包含 "kid"
    if u.Age > 0 && u.Age < 18 {
        if !strings.Contains(u.Username, "kid") {
            sl.ReportError(u.Username, "Username", 
                "username", "minor_username", "")
        }
    }
}
```

---

## 验证场景

验证场景用于区分不同的业务操作，同一个模型在不同场景下可以有不同的验证规则。

```go
const (
    SceneCreate ValidateScene = "create"
    SceneUpdate ValidateScene = "update"
    SceneDelete ValidateScene = "delete"
    SceneQuery  ValidateScene = "query"
)

func (u *User) Rules() map[ValidateScene]map[string]string {
    return map[ValidateScene]map[string]string{
        SceneCreate: {
            "Username": "required,min=3",
            "Password": "required,min=6",
        },
        SceneUpdate: {
            "Username": "omitempty,min=3",
            "Password": "omitempty,min=6",
        },
        SceneQuery: {
            "Username": "omitempty",
        },
    }
}
```

---

## Map 验证

验证器提供强大的 Map 验证功能，适用于动态扩展字段（如 Extras）。

### 便捷验证函数

```go
// 验证必填键
err := validator.ValidateMapMustHaveKeys(extras, "brand", "warranty")

// 验证字符串键（长度验证）
err := validator.ValidateMapStringKey(extras, "brand", 2, 50)

// 验证整数键（范围验证）
err := validator.ValidateMapIntKey(extras, "warranty", 1, 60)

// 验证浮点数键
err := validator.ValidateMapFloatKey(extras, "price", 0.01, 99999.99)

// 自定义键验证
err := validator.ValidateMapKey(extras, "size", func(value any) error {
    size, ok := value.(string)
    if !ok {
        return fmt.Errorf("size 必须是字符串")
    }
    validSizes := map[string]bool{"S": true, "M": true, "L": true}
    if !validSizes[size] {
        return fmt.Errorf("size 必须是 S, M, L 之一")
    }
    return nil
})
```

### 在模型中使用

```go
type Product struct {
    Name     string         `json:"name"`
    Category string         `json:"category"`
    Extras   map[string]any `json:"extras"`
}

func (p *Product) Validate(scene ValidateScene) []*FieldError {
    if p.Extras == nil {
        return nil
    }
    
    var errors []*FieldError
    
    switch p.Category {
    case "electronics":
        if err := validator.ValidateMapMustHaveKeys(p.Extras, "brand", "warranty"); err != nil {
            errors = append(errors, validator.NewFieldError("extras", err.Error(), nil, nil))
        }
    case "clothing":
        if err := validator.ValidateMapMustHaveKeys(p.Extras, "size", "color"); err != nil {
            errors = append(errors, validator.NewFieldError("extras", err.Error(), nil, nil))
        }
    }
    
    if len(errors) > 0 {
        return errors
    }
    return nil
}
```

---

## 嵌套验证

验证器会自动递归验证嵌套的结构体字段，包括嵌入字段（Anonymous Fields）。

```go
// BaseModel - 基础模型
type BaseModel struct {
    ID        int64          `json:"id"`
    CreatedBy string         `json:"created_by"`
}

func (b *BaseModel) Rules() map[ValidateScene]map[string]string {
    return map[ValidateScene]map[string]string{
        "create": {"CreatedBy": "required,min=3,max=50"},
    }
}

// Address - 地址信息
type Address struct {
    City   string `json:"city"`
    Street string `json:"street"`
}

func (a *Address) Rules() map[ValidateScene]map[string]string {
    return map[ValidateScene]map[string]string{
        "create": {
            "City":   "required",
            "Street": "required,min=5,max=200",
        },
    }
}

// Company - 公司信息
type Company struct {
    BaseModel            // 嵌入字段，会自动验证
    Name    string       `json:"name"`
    Address *Address     `json:"address"` // 嵌套字段，会自动验证
}

func (c *Company) Rules() map[ValidateScene]map[string]string {
    return map[ValidateScene]map[string]string{
        "create": {"Name": "required,min=3,max=100"},
    }
}

// 使用 - 验证器会自动验证所有层级
company := &Company{
    BaseModel: BaseModel{CreatedBy: "admin"},
    Name:      "TechCorp",
    Address:   &Address{City: "Shenzhen", Street: "Nanshan District"},
}

errs := validator.Validate(company, "create")
```

**验证流程**：
1. 验证 BaseModel 的规则
2. 验证 Company 的规则
3. 验证 Address 的规则

---

## 自动注册机制

实现 `CrossFieldValidator` 接口的类型会在首次验证时自动注册到验证器，无需手动调用注册方法。

```go
type Order struct {
    Quantity int     `json:"quantity"`
    Price    float64 `json:"price"`
    Total    float64 `json:"total"`
}

func (o *Order) Rules() map[ValidateScene]map[string]string {
    return map[ValidateScene]map[string]string{
        "create": {
            "Quantity": "required,gt=0",
            "Price":    "required,gt=0",
            "Total":    "required,gt=0",
        },
    }
}

// 实现 CrossFieldValidator - 自动注册
func (o *Order) CrossFieldValidation(sl validator.StructLevel) {
    expectedTotal := float64(o.Quantity) * o.Price
    if o.Total != expectedTotal {
        sl.ReportError(o.Total, "Total", "total", "invalid_total", "")
    }
}

// 直接使用，首次验证时自动注册
order := &Order{Quantity: 5, Price: 99.99, Total: 499.95}
errs := validator.Validate(order, "create")
```

---

## 完整使用示例

### 用户模型（包含三种接口）

```go
package models

import (
    "fmt"
    "regexp"
    "katydid-common-account/pkg/validator"
)

const (
    SceneCreate validator.ValidateScene = "create"
    SceneUpdate validator.ValidateScene = "update"
)

type User struct {
    Username        string `json:"username"`
    Email           string `json:"email"`
    Password        string `json:"password"`
    ConfirmPassword string `json:"confirm_password"`
    Phone           string `json:"phone"`
    Age             int    `json:"age"`
}

// 1. RuleProvider - 字段规则
func (u *User) Rules() map[validator.ValidateScene]map[string]string {
    return map[validator.ValidateScene]map[string]string{
        SceneCreate: {
            "Username": "required,min=3,max=20,alphanum",
            "Email":    "required,email",
            "Password": "required,min=6,max=20",
            "Phone":    "omitempty,len=11,numeric",
            "Age":      "omitempty,gte=0,lte=150",
        },
        SceneUpdate: {
            "Username": "omitempty,min=3,max=20,alphanum",
            "Email":    "omitempty,email",
            "Password": "omitempty,min=6,max=20",
        },
    }
}

// 2. BusinessValidator - 业务验证
func (u *User) Validate(scene validator.ValidateScene) []*validator.FieldError {
    var errors []*validator.FieldError

    if scene == SceneCreate {
        // 用户名不能是保留字
        if u.Username == "admin" || u.Username == "root" {
            errors = append(errors, validator.NewFieldError(
                "username", "用户名是保留字，不能使用", nil, nil,
            ))
        }

        // 验证密码强度
        if u.Password != "" {
            hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(u.Password)
            hasNumber := regexp.MustCompile(`[0-9]`).MatchString(u.Password)
            if !hasLetter || !hasNumber {
                errors = append(errors, validator.NewFieldError(
                    "password", "密码必须包含字母和数字", nil, nil,
                ))
            }
        }
    }

    if len(errors) > 0 {
        return errors
    }
    return nil
}

// 3. CrossFieldValidator - 跨字段验证
func (u *User) CrossFieldValidation(sl validator.StructLevel) {
    if u.Password != "" && u.Password != u.ConfirmPassword {
        sl.ReportError(u.ConfirmPassword, "ConfirmPassword", 
            "confirm_password", "password_mismatch", "")
    }
}
```

---

## 性能优化

### 1. 类型缓存

验证器会缓存类型信息，避免重复的类型断言：

```go
v := validator.New()
v.Validate(user1, "create") // 首次验证，缓存类型信息
v.Validate(user2, "create") // 使用缓存，性能提升
```

### 2. 规则缓存

自动注册的验证规则只会注册一次，后续验证会复用已注册的规则。

### 3. 并发安全

验证器是线程安全的，可以在多个 goroutine 中并发使用：

```go
v := validator.New()

var wg sync.WaitGroup
for _, user := range users {
    wg.Add(1)
    go func(u *User) {
        defer wg.Done()
        _ = v.Validate(u, "create")
    }(user)
}
wg.Wait()
```

---

## 最佳实践

### 1. 接口选择指南

| 验证需求 | 使用接口 |
|---------|---------|
| 字段格式验证 | `RuleProvider` |
| 复杂业务逻辑 | `BusinessValidator` |
| 字段间关系 | `CrossFieldValidator` |
| Map 动态字段 | `BusinessValidator` + Map验证函数 |

### 2. 场景化验证

```go
const (
    SceneCreate ValidateScene = "create"
    SceneUpdate ValidateScene = "update"
    SceneDelete ValidateScene = "delete"
)
```

### 3. 自定义错误消息

```go
func (u *User) GetErrorMessage(fieldName, tag, param string) string {
    messages := map[string]map[string]string{
        "username": {
            "required": "用户名不能为空",
            "min": fmt.Sprintf("用户名长度不能少于%s个字符", param),
        },
    }
    
    if fieldMessages, ok := messages[fieldName]; ok {
        if msg, ok := fieldMessages[tag]; ok {
            return msg
        }
    }
    return ""
}
```

---

## API 参考

### 全局函数

```go
// 使用默认验证器验证
func Validate(obj any, scene ValidateScene) []*FieldError

// 获取默认验证器实例
func Default() *Validator

// 清除类型缓存
func ClearTypeCache()
```

### Validator 方法

```go
// 创建新的验证器
func New() *Validator

// 验证对象
func (v *Validator) Validate(obj any, scene ValidateScene) []*FieldError

// 清除类型缓存
func (v *Validator) ClearTypeCache()

// 获取底层验证器
func (v *Validator) GetUnderlyingValidator() *validator.Validate
```

### Map 验证函数

```go
// 验证必填键
func ValidateMapMustHaveKeys(data map[string]any, keys ...string) error

// 验证字符串键
func ValidateMapStringKey(data map[string]any, key string, minLen, maxLen int) error

// 验证整数键
func ValidateMapIntKey(data map[string]any, key string, min, max int) error

// 验证浮点数键
func ValidateMapFloatKey(data map[string]any, key string, min, max float64) error

// 自定义键验证
func ValidateMapKey(data map[string]any, key string, validatorFunc func(value any) error) error
```

---

## 常见验证标签

### 字符串验证
```
required      - 必填
omitempty     - 可选
min=N         - 最小长度
max=N         - 最大长度
len=N         - 长度等于
alpha         - 只包含字母
alphanum      - 只包含字母和数字
numeric       - 只包含数字
email         - 邮箱格式
url           - URL 格式
```

### 数字验证
```
gt=N          - 大于
gte=N         - 大于等于
lt=N          - 小于
lte=N         - 小于等于
eq=N          - 等于
oneof=A B C   - 值必须是 A、B 或 C 之一
```

### 字段比较
```
eqfield=F     - 等于某个字段
nefield=F     - 不等于某个字段
gtfield=F     - 大于某个字段
```

更多标签请参考：https://pkg.go.dev/github.com/go-playground/validator/v10

---

## 许可证

本项目遵循项目根目录的许可证。

