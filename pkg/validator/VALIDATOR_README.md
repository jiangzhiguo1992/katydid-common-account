# Validator 验证器

## 概述

`validator` 包提供了一个强大且灵活的数据验证框架，基于 `go-playground/validator/v10` 构建，支持场景化验证、嵌套验证、自定义验证逻辑和**自定义错误消息**。

## 核心特性

### 1. 场景化验证
支持针对不同业务场景（创建、更新、删除、查询）定义不同的验证规则。

### 2. 多层验证机制
- **结构体标签验证**：基于 `validate` 标签的字段验证
- **嵌套结构体验证**：自动递归验证嵌套的结构体字段
- **自定义验证逻辑**：支持业务级别的复杂验证
- **接口化设计**：通过接口实现灵活的验证扩展

### 3. 友好的错误信息
- 自动将字段名转换为 JSON 标签名
- 支持**模型自定义错误消息**（推荐）
- 提供默认中文错误提示作为后备

### 4. 扩展性设计
- **错误消息由外部模型定义**：避免频繁修改基础包
- 基础包提供基本的默认错误消息
- 业务模型可完全自定义验证错误信息

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

### ErrorMessageProvider 接口 ⭐️ 重要

**实现此接口以自定义验证错误消息（强烈推荐）**

这是验证器最重要的扩展点，允许业务模型完全控制错误消息的内容和格式。

```go
type ErrorMessageProvider interface {
    // GetErrorMessage 获取字段验证失败的错误信息
    // fieldName: 字段名（JSON标签名）
    // tag: 验证标签（如 required, email, min 等）
    // param: 验证参数（如 min=3 中的 3）
    GetErrorMessage(fieldName, tag, param string) string
}
```

**设计理念：**
- ✅ **基础包保持稳定**：不需要为每个新的验证规则修改基础包
- ✅ **业务自定义**：每个模型可以定义符合自己业务的错误消息
- ✅ **灵活性**：支持国际化、多语言、业务术语等
- ✅ **降低耦合**：基础包只提供默认后备消息

**使用示例：**

```go
type User struct {
    ID       int64  `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
    Age      int    `json:"age"`
}

// 实现 ErrorMessageProvider 接口
func (u *User) GetErrorMessage(fieldName, tag, param string) string {
    // 根据字段和验证标签返回自定义错误消息
    switch fieldName {
    case "username":
        switch tag {
        case "required":
            return "用户名不能为空"
        case "min":
            return fmt.Sprintf("用户名长度不能少于%s个字符", param)
        case "max":
            return fmt.Sprintf("用户名长度不能超过%s个字符", param)
        case "alphanum":
            return "用户名只能包含字母和数字"
        }
    case "email":
        switch tag {
        case "required":
            return "邮箱地址不能为空"
        case "email":
            return "请输入有效的邮箱地址"
        }
    case "password":
        switch tag {
        case "required":
            return "密码不能为空"
        case "min":
            return fmt.Sprintf("密码长度不能少于%s位", param)
        case "max":
            return fmt.Sprintf("密码长度不能超过%s位", param)
        }
    case "age":
        switch tag {
        case "gte":
            return fmt.Sprintf("年龄必须大于等于%s岁", param)
        case "lte":
            return fmt.Sprintf("年龄必须小于等于%s岁", param)
        }
    }
    
    // 返回空字符串使用默认消息
    return ""
}
```

**高级用法 - 支持国际化：**

```go
type Product struct {
    Name  string `json:"name"`
    Price float64 `json:"price"`
    Lang  string `json:"-"` // 语言标识
}

func (p *Product) GetErrorMessage(fieldName, tag, param string) string {
    // 根据语言返回不同的错误消息
    if p.Lang == "en" {
        return p.getEnglishMessage(fieldName, tag, param)
    }
    return p.getChineseMessage(fieldName, tag, param)
}

func (p *Product) getEnglishMessage(fieldName, tag, param string) string {
    if fieldName == "name" && tag == "required" {
        return "Product name is required"
    }
    // ... 其他英文消息
    return ""
}

func (p *Product) getChineseMessage(fieldName, tag, param string) string {
    if fieldName == "name" && tag == "required" {
        return "产品名称不能为空"
    }
    // ... 其他中文消息
    return ""
}
```

**最佳实践：**

1. **优先实现 ErrorMessageProvider**：为重要的业务模型实现此接口
2. **返回空字符串使用默认消息**：不需要为每个验证规则都定义消息
3. **使用业务术语**：错误消息应该符合业务场景，而不是技术术语
4. **支持参数化**：使用 `param` 参数使消息更加具体
5. **考虑用户体验**：错误消息应该友好、清晰、可操作

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

### 基础场景验证（带自定义错误消息）

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

// 实现自定义错误消息
func (p *Product) GetErrorMessage(fieldName, tag, param string) string {
    switch fieldName {
    case "name":
        switch tag {
        case "required":
            return "产品名称是必填项"
        case "min":
            return "产品名称至少需要2个字符"
        case "max":
            return "产品名称不能超过100个字符"
        }
    case "price":
        switch tag {
        case "required":
            return "产品价格是必填项"
        case "gt":
            return "产品价格必须大于0"
        }
    case "stock":
        switch tag {
        case "required":
            return "库存数量是必填项"
        case "gte":
            return "库存数量不能为负数"
        }
    }
    return "" // 使用默认消息
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

### 错误消息优先级

验证器按以下优先级获取错误消息：

1. **模型自定义消息**（通过 `ErrorMessageProvider` 接口）⭐️ 最高优先级
2. **基础包默认消息**（作为后备）

### 基础包默认消息

基础包提供以下默认错误消息作为后备：

- `字段 'username' 验证失败: 必填项`
- `字段 'email' 验证失败: 必须是有效的邮箱地址`
- `字段 'age' 验证失败: 必须大于等于 0`
- `字段 'Product' 验证失败: 字段 'name' 验证失败: 最小值/长度为 2`

**重要提示：**
- ⚠️ 基础包的默认消息**仅作为后备**，不应频繁修改
- ✅ 业务模型应该**实现 ErrorMessageProvider 接口**来定义自己的错误消息
- ✅ 这样可以保持基础包稳定，避免因新增验证规则而频繁变更基础包

## 最佳实践

### 1. 优先使用场景化验证

```go
// 推荐：使用场景化验证
err := validator.Validate(user, validator.SceneCreate)

// 不推荐：不区分场景
err := validator.ValidateStruct(user)
```

### 2. 为重要模型实现 ErrorMessageProvider ⭐️

```go
type User struct {
    Username string `json:"username"`
    Email    string `json:"email"`
}

// 强烈推荐：实现自定义错误消息
func (u *User) GetErrorMessage(fieldName, tag, param string) string {
    // 返回业务友好的错误消息
    if fieldName == "username" && tag == "required" {
        return "请输入用户名"
    }
    if fieldName == "email" && tag == "email" {
        return "邮箱格式不正确，请重新输入"
    }
    return "" // 其他情况使用默认消息
}
```

### 3. 合理组织验证规则

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

### 4. 自定义验证处理复杂逻辑

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

### 5. 使用默认验证器

大多数情况下使用默认验证器即可，无需创建多个实例：

```go
// 推荐
err := validator.Validate(obj, scene)

// 特殊场景才创建实例
v := validator.New()
v.RegisterValidation("custom", customFunc)
err := v.Validate(obj, scene)
```

### 6. 错误消息设计原则 ⭐️

**好的错误消息示例：**
```go
✅ "用户名不能为空"
✅ "邮箱格式不正确，请输入有效的邮箱地址"
✅ "密码长度必须在6-20位之间"
✅ "价格必须大于0"
```

**不好的错误消息示例：**
```go
❌ "必填项"  // 太笼统
❌ "不符合规则 'min'"  // 技术术语
❌ "validation failed"  // 不够具体
```

**设计建议：**
- 使用业务语言，不是技术术语
- 告诉用户如何修正错误
- 提供具体的限制条件（如范围、格式等）
- 考虑用户场景和体验

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
