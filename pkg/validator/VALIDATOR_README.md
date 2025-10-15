# Validator 验证器

## 概述

`validator` 包提供了一个强大且灵活的数据验证框架，基于 `go-playground/validator/v10` 构建，支持场景化验证、嵌套验证和自定义验证逻辑。

## 核心特性

### 1. 场景化验证
支持针对不同业务场景（创建、更新、删除、查询）定义不同的验证规则。

### 2. 多层验证机制
- **结构体标签验证**：基于 `validate` 标签的字段验证
- **嵌套结构体验证**：自动递归验证嵌套的结构体字段
- **自定义验证逻辑**：支持业务级别的复杂验证
- **接口化设计**：通过接口实现灵活的验证扩展

### 3. 友好的错误信息
自动将字段名转换为 JSON 标签名，提供中文错误提示。

## 核心类型

### ValidateScene 验证场景

```go
type ValidateScene string

const (
    SceneCreate ValidateScene = "create" // 创建场景
    SceneUpdate ValidateScene = "update" // 更新场景
    SceneDelete ValidateScene = "delete" // 删除场景
    SceneQuery  ValidateScene = "query"  // 查询场景
)
```

### Validator 验证器

```go
type Validator struct {
    validate *validator.Validate
    mu       sync.RWMutex
}
```

主验证器类型，封装了 `go-playground/validator` 并提供扩展功能。

## 核心接口

### Validatable 接口

实现此接口以定义场景化的验证规则：

```go
type Validatable interface {
    ValidateRules() map[ValidateScene]map[string]string
}
```

**使用示例：**

```go
type User struct {
    ID       int64  `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

func (u *User) ValidateRules() map[ValidateScene]map[string]string {
    return map[ValidateScene]map[string]string{
        SceneCreate: {
            "Username": "required,min=3,max=20,alphanum",
            "Email":    "required,email",
            "Password": "required,min=6,max=20",
        },
        SceneUpdate: {
            "Username": "omitempty,min=3,max=20,alphanum",
            "Email":    "omitempty,email",
        },
    }
}
```

### CustomValidatable 接口

实现此接口以添加自定义业务验证逻辑：

```go
type CustomValidatable interface {
    CustomValidate(scene ValidateScene) error
}
```

**使用示例：**

```go
func (u *User) CustomValidate(scene ValidateScene) error {
    if scene == SceneCreate {
        // 创建时，用户名不能是保留字
        if u.Username == "admin" || u.Username == "root" {
            return fmt.Errorf("用户名 '%s' 是保留字，不能使用", u.Username)
        }
    }
    return nil
}
```

### NestedValidatable 接口

用于验证嵌套的复杂结构（如嵌套的 map、slice 等）：

```go
type NestedValidatable interface {
    ValidateNested(scene ValidateScene) error
}
```

## 核心方法

### 创建验证器

```go
// 使用默认验证器（推荐）
v := validator.Default()

// 创建新的验证器实例
v := validator.New()
```

### 执行验证

```go
// 场景化验证（推荐）
err := validator.Validate(obj, validator.SceneCreate)

// 简单结构体验证（不区分场景）
err := validator.ValidateStruct(obj)

// 使用验证器实例
v := validator.Default()
err := v.Validate(obj, validator.SceneUpdate)
```

### 注册自定义验证规则

```go
err := validator.RegisterValidation("myvalidation", func(fl validator.FieldLevel) bool {
    // 自定义验证逻辑
    return true
})
```

## 验证流程

验证器按以下顺序执行验证：

1. **结构体标签验证**
   - 如果实现了 `Validatable` 接口，根据场景规则验证
   - 否则使用默认的 validator 验证所有字段

2. **递归验证嵌套结构体**
   - 自动发现并验证嵌套的结构体字段
   - 包括嵌入的 BaseModel 等

3. **自定义验证逻辑**
   - 执行 `CustomValidate` 方法（如果实现）

4. **嵌套验证**
   - 执行 `ValidateNested` 方法（如果实现）

## 完整使用示例

### 基础场景验证

```go
package main

import (
    "fmt"
    "katydid-common-account/pkg/validator"
)

type Product struct {
    ID    int64   `json:"id"`
    Name  string  `json:"name"`
    Price float64 `json:"price"`
    Stock int     `json:"stock"`
}

func (p *Product) ValidateRules() map[validator.ValidateScene]map[string]string {
    return map[validator.ValidateScene]map[string]string{
        validator.SceneCreate: {
            "Name":  "required,min=2,max=100",
            "Price": "required,gt=0",
            "Stock": "required,gte=0",
        },
        validator.SceneUpdate: {
            "Name":  "omitempty,min=2,max=100",
            "Price": "omitempty,gt=0",
            "Stock": "omitempty,gte=0",
        },
    }
}

func main() {
    product := &Product{
        Name:  "iPhone",
        Price: 999.99,
        Stock: 100,
    }

    // 创建场景验证
    if err := validator.Validate(product, validator.SceneCreate); err != nil {
        fmt.Printf("验证失败: %v\n", err)
        return
    }

    fmt.Println("验证通过")
}
```

### 自定义验证逻辑

```go
func (p *Product) CustomValidate(scene validator.ValidateScene) error {
    if scene == validator.SceneCreate {
        // 创建时价格不能低于成本
        if p.Price < 10.0 {
            return fmt.Errorf("产品价格不能低于最低成本 10.0")
        }
        
        // 库存必须是10的倍数
        if p.Stock%10 != 0 {
            return fmt.Errorf("库存必须是10的倍数，便于管理")
        }
    }
    return nil
}
```

### 嵌套结构验证

```go
type Order struct {
    ID       int64    `json:"id"`
    UserID   int64    `json:"user_id"`
    Product  *Product `json:"product"`  // 嵌套的产品
}

func (o *Order) ValidateRules() map[validator.ValidateScene]map[string]string {
    return map[validator.ValidateScene]map[string]string{
        validator.SceneCreate: {
            "UserID": "required,gt=0",
        },
    }
}

// Product 会被自动递归验证
order := &Order{
    UserID: 123,
    Product: &Product{
        Name:  "iPhone",
        Price: 999.99,
        Stock: 100,
    },
}

err := validator.Validate(order, validator.SceneCreate)
```

## 支持的验证标签

基于 `go-playground/validator/v10`，支持以下常用标签：

| 标签 | 说明 | 示例 |
|------|------|------|
| `required` | 必填字段 | `validate:"required"` |
| `omitempty` | 空值时跳过验证 | `validate:"omitempty,email"` |
| `email` | 邮箱格式 | `validate:"email"` |
| `url` | URL格式 | `validate:"url"` |
| `min` | 最小值/长度 | `validate:"min=3"` |
| `max` | 最大值/长度 | `validate:"max=100"` |
| `len` | 精确长度 | `validate:"len=11"` |
| `gt` | 大于 | `validate:"gt=0"` |
| `gte` | 大于等于 | `validate:"gte=0"` |
| `lt` | 小于 | `validate:"lt=100"` |
| `lte` | 小于等于 | `validate:"lte=100"` |
| `alpha` | 只包含字母 | `validate:"alpha"` |
| `alphanum` | 字母和数字 | `validate:"alphanum"` |
| `numeric` | 只包含数字 | `validate:"numeric"` |
| `oneof` | 枚举值 | `validate:"oneof=pending approved rejected"` |

完整标签列表请参考：https://pkg.go.dev/github.com/go-playground/validator/v10

## 错误信息

验证器提供中文错误信息，常见错误：

- `字段 'username' 验证失败: 必填项`
- `字段 'email' 验证失败: 必须是有效的邮箱地址`
- `字段 'age' 验证失败: 必须大于等于 0`
- `字段 'Product' 验证失败: 字段 'name' 验证失败: 最小值/长度为 2`

## 最佳实践

### 1. 优先使用场景化验证

```go
// 推荐：使用场景化验证
err := validator.Validate(user, validator.SceneCreate)

// 不推荐：不区分场景
err := validator.ValidateStruct(user)
```

### 2. 合理组织验证规则

```go
func (u *User) ValidateRules() map[validator.ValidateScene]map[string]string {
    // 定义通用规则
    commonRules := map[string]string{
        "Email": "email",
        "Age":   "omitempty,gte=0,lte=150",
    }
    
    // 创建场景：添加必填要求
    createRules := make(map[string]string)
    for k, v := range commonRules {
        createRules[k] = v
    }
    createRules["Username"] = "required,min=3,max=20"
    createRules["Password"] = "required,min=6"
    
    return map[validator.ValidateScene]map[string]string{
        validator.SceneCreate: createRules,
        validator.SceneUpdate: commonRules,
    }
}
```

### 3. 自定义验证处理复杂逻辑

将复杂的业务验证逻辑放在 `CustomValidate` 中：

```go
func (p *Product) CustomValidate(scene validator.ValidateScene) error {
    // 复杂的跨字段验证
    if p.Category == "electronics" && p.Warranty == 0 {
        return fmt.Errorf("电子产品必须提供保修期")
    }
    
    // 调用外部服务验证
    if scene == validator.SceneCreate {
        if exists := checkProductNameExists(p.Name); exists {
            return fmt.Errorf("产品名称已存在")
        }
    }
    
    return nil
}
```

### 4. 使用默认验证器

大多数情况下使用默认验证器即可，无需创建多个实例：

```go
// 推荐
err := validator.Validate(obj, scene)

// 特殊场景才创建实例
v := validator.New()
v.RegisterValidation("custom", customFunc)
err := v.Validate(obj, scene)
```

## 与其他包的集成

### 与 BaseModel 集成

```go
type BaseModel struct {
    ID     int64        `json:"id"`
    Status types.Status `json:"status"`
    Extras types.Extras `json:"extras,omitempty"`
}

type Product struct {
    BaseModel  // 嵌入 BaseModel，会自动验证
    Name  string  `json:"name"`
    Price float64 `json:"price"`
}
```

### 在 HTTP Handler 中使用

```go
func CreateUserHandler(c *gin.Context) {
    var user User
    if err := c.ShouldBindJSON(&user); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 验证
    if err := validator.Validate(&user, validator.SceneCreate); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 处理业务逻辑
    // ...
}
```

## 性能考虑

- 验证器使用单例模式，避免重复创建
- 使用反射进行验证，对性能有一定影响
- 对于高性能要求的场景，可考虑缓存验证结果
- 嵌套验证会递归遍历所有字段，深层嵌套可能影响性能

## 相关文档

- [MAP_VALIDATOR_README.md](./MAP_VALIDATOR_README.md) - Map 验证器文档
- [NESTED_VALIDATION_README.md](./NESTED_VALIDATION_README.md) - 嵌套验证文档
- [go-playground/validator](https://github.com/go-playground/validator) - 底层验证库文档

