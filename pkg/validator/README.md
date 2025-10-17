# Validator 验证器

一个功能强大、灵活且高性能的 Go 验证器库，基于 `go-playground/validator/v10` 封装，提供场景化验证、嵌套验证、自定义验证等多种验证方式。

## 目录

- [概述](#概述)
- [核心特性](#核心特性)
- [快速开始](#快速开始)
- [验证场景](#验证场景)
- [验证接口](#验证接口)
- [Map 验证](#map-验证)
- [嵌套验证](#嵌套验证)
- [自动注册](#自动注册)
- [性能优化](#性能优化)
- [最佳实践](#最佳实践)
- [封装设计](#封装设计)

## 概述

本验证器提供了一套完整的验证解决方案，支持：

- **场景化验证**：针对不同业务场景（创建、更新、删除等）使用不同的验证规则
- **接口驱动**：通过实现接口来定义验证规则，灵活且易于维护
- **嵌套验证**：自动递归验证嵌套的结构体字段
- **自动注册**：首次使用时自动注册验证规则，无需手动调用
- **高性能**：使用类型缓存和规则缓存机制，提升验证性能
- **线程安全**：可在多个 goroutine 中并发使用

## 核心特性

### 1. 场景化验证

通过 `ValidateScene` 定义不同的验证场景，同一个模型在不同场景下可以有不同的验证规则。

```go
const (
    SceneCreate ValidateScene = "create" // 创建场景
    SceneUpdate ValidateScene = "update" // 更新场景
)

type User struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

func (u *User) ValidateRules() map[ValidateScene]map[string]string {
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
err := validator.Validate(user, validator.SceneCreate)
```

### 2. 多种验证接口

#### Validatable - 定义验证规则
```go
type Validatable interface {
    ValidateRules() map[ValidateScene]map[string]string
}
```

#### CustomValidatable - 自定义验证逻辑
```go
type CustomValidatable interface {
    CustomValidate(scene ValidateScene) []*FieldError
}
```

#### NestedValidatable - 嵌套对象验证
```go
type NestedValidatable interface {
    ValidateNested(scene ValidateScene) []*FieldError
}
```

#### StructLevelValidatable - 结构体级别验证（自动注册）
```go
type StructLevelValidatable interface {
    StructLevelValidation(sl StructLevel)
}
```

#### MapRulesValidatable - Map 规则验证（自动注册）
```go
type MapRulesValidatable interface {
    ValidationMapRules() map[string]string
}
```

### 3. 自定义错误消息

通过实现 `ErrorMessageProvider` 接口自定义错误消息：

```go
func (u *User) GetErrorMessage(fieldName, tag, param string) string {
    switch fieldName {
    case "username":
        switch tag {
        case "required":
            return "用户名不能为空"
        case "min":
            return fmt.Sprintf("用户名长度不能少于%s个字符", param)
        }
    }
    return ""
}
```

## 快速开始

### 基础使用

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

func (u *User) ValidateRules() map[validator.ValidateScene]map[string]string {
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

    // 验证
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

### 使用默认验证器

```go
// 使用默认的全局验证器实例
errs := validator.Validate(user, validator.SceneCreate)

// 或者创建独立的验证器实例
v := validator.New()
errs := v.Validate(user, validator.SceneCreate)
```

## Map 验证

### 基础 Map 验证

使用 `MapValidator` 进行灵活的 Map 数据验证：

```go
// 创建 Map 验证器
mv := validator.NewMapValidator().
    WithRequiredKeys("name", "email").                     // 必填键
    WithAllowedKeys("name", "email", "age", "phone").     // 允许的键
    WithKeyValidator("email", func(value interface{}) error {
        email, ok := value.(string)
        if !ok || !strings.Contains(email, "@") {
            return fmt.Errorf("无效的邮箱格式")
        }
        return nil
    })

// 验证数据
data := map[string]any{
    "name":  "John",
    "email": "john@example.com",
    "age":   25,
}

err := mv.Validate(data)
```

### 便捷验证函数

```go
// 验证必填键
err := validator.ValidateMapMustHaveKeys(extras, "brand", "warranty")

// 验证字符串键
err := validator.ValidateMapStringKey(extras, "brand", 2, 50) // 长度 2-50

// 验证整数键
err := validator.ValidateMapIntKey(extras, "warranty", 1, 60) // 值 1-60

// 验证浮点数键
err := validator.ValidateMapFloatKey(extras, "price", 0.01, 99999.99)

// 自定义键验证
err := validator.ValidateMapKey(extras, "size", func(value interface{}) error {
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

### 在模型中使用 Map 验证

```go
type Product struct {
    Name   string              `json:"name"`
    Extras map[string]any      `json:"extras"`
}

func (p *Product) CustomValidate(scene ValidateScene) []*FieldError {
    if p.Extras == nil {
        return nil
    }

    // 电子产品额外属性验证
    if err := ValidateMapMustHaveKeys(p.Extras, "brand", "warranty"); err != nil {
        return []*FieldError{NewFieldError("extras", err.Error(), nil, nil)}
    }

    if err := ValidateMapStringKey(p.Extras, "brand", 2, 50); err != nil {
        return []*FieldError{NewFieldError("extras.brand", err.Error(), nil, nil)}
    }

    return nil
}
```

## 嵌套验证

验证器会自动递归验证嵌套的结构体字段，包括嵌入字段。

### 示例：产品模型嵌套验证

```go
type BaseModel struct {
    ID     int64        `json:"id"`
    Extras types.Extras `json:"extras,omitempty"`
}

type Product struct {
    BaseModel              // 嵌入字段会被自动验证
    Name     string       `json:"name"`
    Price    float64      `json:"price"`
}

func (p *Product) ValidateRules() map[ValidateScene]map[string]string {
    return map[ValidateScene]map[string]string{
        SceneCreate: {
            "Name":  "required,min=2,max=100",
            "Price": "required,gt=0",
        },
    }
}

func (p *Product) CustomValidate(scene ValidateScene) []*FieldError {
    // 验证 Extras 中的额外属性
    if p.Extras != nil {
        if err := ValidateMapMustHaveKeys(p.Extras, "brand"); err != nil {
            return []*FieldError{NewFieldError("extras", err.Error(), nil, nil)}
        }
    }
    return nil
}

// 使用
product := &Product{
    BaseModel: BaseModel{
        Extras: types.Extras{"brand": "Apple"},
    },
    Name:  "iPhone 15",
    Price: 999.99,
}

errs := validator.Validate(product, SceneCreate)
```

### 嵌套验证流程

验证器会按以下顺序进行验证：

1. 执行结构体标签验证（基于 `Validatable` 接口的规则）
2. 递归验证嵌套的结构体字段（包括嵌入字段）
3. 执行自定义验证逻辑（`CustomValidatable` 接口）
4. 验证实现了 `NestedValidatable` 接口的嵌套字段

## 自动注册

实现 `StructLevelValidatable` 或 `MapRulesValidatable` 接口的类型会在首次验证时自动注册，无需手动调用注册方法。

### StructLevelValidatable 自动注册

用于跨字段验证和复杂的业务逻辑验证：

```go
type UserWithAutoRegister struct {
    Username        string `json:"username"`
    Password        string `json:"password"`
    ConfirmPassword string `json:"confirm_password"`
}

// 实现 StructLevelValidatable 接口，会自动注册
func (u UserWithAutoRegister) StructLevelValidation(sl validator.StructLevel) {
    // 验证密码和确认密码是否一致
    if u.Password != u.ConfirmPassword {
        sl.ReportError(u.ConfirmPassword, "ConfirmPassword", "confirmPassword", "eqfield", "Password")
    }
}

// 直接使用，无需手动注册
user := UserWithAutoRegister{
    Username:        "john",
    Password:        "pass123",
    ConfirmPassword: "pass123",
}

v := validator.New()
err := v.ValidateStruct(user) // 首次验证时自动注册
```

### MapRulesValidatable 自动注册

用于简单的字段验证规则定义：

```go
type Product struct {
    Name  string  `json:"name"`
    Price float64 `json:"price"`
}

// 实现 MapRulesValidatable 接口，会自动注册
func (p Product) ValidationMapRules() map[string]string {
    return map[string]string{
        "Name":  "required,min=3,max=100",
        "Price": "required,gt=0",
    }
}

// 直接使用，无需手动注册
product := Product{Name: "iPhone", Price: 999.99}
v := validator.New()
err := v.ValidateStruct(product) // 首次验证时自动注册
```

### 同时实现两个接口

可以同时使用 `MapRulesValidatable`（字段规则）和 `StructLevelValidatable`（跨字段验证）：

```go
type Order struct {
    Quantity int     `json:"quantity"`
    Price    float64 `json:"price"`
    Total    float64 `json:"total"`
}

// 字段规则
func (o Order) ValidationMapRules() map[string]string {
    return map[string]string{
        "Quantity": "required,gt=0",
        "Price":    "required,gt=0",
        "Total":    "required,gt=0",
    }
}

// 跨字段验证
func (o Order) StructLevelValidation(sl validator.StructLevel) {
    expectedTotal := float64(o.Quantity) * o.Price
    if o.Total != expectedTotal {
        sl.ReportError(o.Total, "Total", "total", "invalid_total", "")
    }
}
```

### 自动注册 vs 手动注册

**自动注册的优点：**
- 无需手动调用注册方法
- 代码更简洁
- 首次使用时懒加载

**手动注册的优点（在 init 函数中）：**
- 应用启动时一次性注册所有规则
- 避免首次验证的注册开销
- 适合高性能场景

```go
func init() {
    // 应用启动时注册所有验证规则
    _ = validator.RegisterStructValidation(func(sl validator.StructLevel) {
        // 验证逻辑
    }, UserRegistration{})
}
```

## 性能优化

### 1. 类型缓存

验证器会缓存类型信息，避免重复的类型断言：

```go
// 首次验证会缓存类型信息
v := validator.New()
v.Validate(user1, "create") // 缓存 User 类型信息
v.Validate(user2, "create") // 使用缓存，性能更好
```

### 2. 规则缓存

自动注册的验证规则只会注册一次，后续验证会复用已注册的规则。

### 3. 并发安全

验证器是线程安全的，可以在多个 goroutine 中并发使用：

```go
v := validator.New()

// 并发验证
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

### 4. 性能基准

```
BenchmarkValidate_TypeCaching-8              500000     2500 ns/op
BenchmarkAutoRegister_Cached-8              1000000     1200 ns/op
BenchmarkMapValidator_AllowedKeys-8         2000000      800 ns/op
BenchmarkValidate_Parallel-8                3000000      500 ns/op
```

## 最佳实践

### 1. 使用场景化验证

根据不同的业务场景定义不同的验证规则：

```go
const (
    SceneCreate ValidateScene = "create"
    SceneUpdate ValidateScene = "update"
    SceneDelete ValidateScene = "delete"
)
```

### 2. 合理使用验证接口

- `Validatable`：基础字段验证
- `CustomValidatable`：复杂业务逻辑验证
- `NestedValidatable`：嵌套对象验证
- `StructLevelValidatable`：跨字段验证（自动注册）
- `MapRulesValidatable`：简单规则定义（自动注册）

### 3. 自定义错误消息

为用户提供清晰、友好的错误消息：

```go
func (u *User) GetErrorMessage(fieldName, tag, param string) string {
    // 根据字段和标签返回自定义消息
    return "友好的错误提示"
}
```

### 4. 提前注册（高性能场景）

在应用启动时注册所有验证规则：

```go
func init() {
    initValidationRules()
}

func initValidationRules() {
    _ = validator.RegisterStructValidation(...)
    _ = validator.RegisterStructValidationMapRules(...)
}
```

### 5. Map 验证使用链式调用

```go
mv := validator.NewMapValidator().
    WithRequiredKeys("name", "email").
    WithAllowedKeys("name", "email", "age").
    WithKeyValidator("email", emailValidator)
```

## 封装设计

本验证器对 `go-playground/validator/v10` 进行了封装，隐藏了第三方库的实现细节。

### 设计原则

1. **接口隔离**：不直接暴露第三方库的类型
2. **易用性**：提供更简洁的 API
3. **可扩展性**：支持自定义验证逻辑
4. **向后兼容**：方便未来替换底层实现

### StructLevel 接口封装

```go
// 封装的 StructLevel 接口
type StructLevel interface {
    Validator() *Validator
    Top() reflect.Value
    Parent() reflect.Value
    Current() reflect.Value
    ExtractType(field reflect.Value) (reflect.Value, reflect.Kind, bool)
    ReportError(field interface{}, fieldName, structFieldName, tag, param string)
    ReportValidationErrors(relativeNamespace, relativeActualNamespace string, errs ValidationErrors)
}

// 使用示例
func (u User) StructLevelValidation(sl validator.StructLevel) {
    // 使用封装的接口，不依赖第三方库类型
    sl.ReportError(u.Password, "Password", "password", "eqfield", "ConfirmPassword")
}
```

### 获取底层验证器（高级用法）

如果需要直接访问底层验证器（不推荐），可以使用：

```go
underlying := v.GetUnderlyingValidator()
// 使用 go-playground/validator 的原生 API
```

## API 参考

### 全局函数

```go
// 使用默认验证器验证
func Validate(obj any, scene ValidateScene) []*FieldError

// 获取默认验证器实例
func Default() *Validator

// 清除类型缓存
func ClearTypeCache()

// 注册自定义验证标签
func RegisterValidation(tag string, fn validator.Func) error

// 注册结构体验证
func RegisterStructValidation(fn validator.StructLevelFunc, types ...interface{}) error

// 注册 Map 规则
func RegisterStructValidationMapRules(rules map[string]string, types ...interface{}) error

// 验证结构体（使用 struct tag）
func ValidateStruct(obj any) error
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
// 创建 Map 验证器
func NewMapValidator() *MapValidator

// 验证 Map
func ValidateMap(data map[string]any, validator *MapValidator) error

// 验证必填键
func ValidateMapMustHaveKeys(data map[string]any, keys ...string) error

// 验证字符串键
func ValidateMapStringKey(data map[string]any, key string, minLen, maxLen int) error

// 验证整数键
func ValidateMapIntKey(data map[string]any, key string, min, max int) error

// 验证浮点数键
func ValidateMapFloatKey(data map[string]any, key string, min, max float64) error

// 自定义键验证
func ValidateMapKey(data map[string]any, key string, validatorFunc func(value interface{}) error) error
```

## 常见验证标签

```
required     - 必填
omitempty    - 可选
min=N        - 最小长度/值
max=N        - 最大长度/值
len=N        - 长度等于
gte=N        - 大于等于
lte=N        - 小于等于
gt=N         - 大于
lt=N         - 小于
email        - 邮箱格式
url          - URL 格式
alpha        - 只包含字母
alphanum     - 只包含字母和数字
numeric      - 只包含数字
eqfield=F    - 等于某个字段
nefield=F    - 不等于某个字段
```

更多标签请参考：https://pkg.go.dev/github.com/go-playground/validator/v10

## 许可证

本项目遵循项目根目录的许可证。

