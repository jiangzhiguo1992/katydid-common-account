# Nested Validation 嵌套验证

## 概述

嵌套验证是 validator 包的核心功能之一，支持自动递归验证嵌套的结构体字段、嵌入的基础模型以及复杂的动态数据结构。这使得验证器能够处理真实业务场景中的复杂数据模型。

## 核心概念

### 什么是嵌套验证？

嵌套验证是指在验证一个对象时，自动验证其内部嵌套的子对象。包括：

1. **嵌入的结构体**（如 BaseModel）
2. **嵌套的结构体字段**（如 Product 中的 Category 对象）
3. **动态数据结构**（如 map[string]any 类型的 Extras 字段）
4. **指针类型的嵌套字段**

## NestedValidatable 接口

用于验证复杂的嵌套数据结构（特别是非结构化数据）。

```go
type NestedValidatable interface {
    ValidateNested(scene ValidateScene) error
}
```

### 使用场景

当模型包含需要特殊处理的嵌套数据（如 map、slice、动态字段）时，实现此接口。

## 验证流程

验证器对嵌套结构的处理顺序：

1. **主对象标签验证**
   - 验证主对象的字段（通过 ValidateRules）

2. **递归验证嵌套结构体**
   - 自动发现并验证所有嵌套的结构体字段
   - 包括嵌入的 BaseModel
   - 递归处理多层嵌套

3. **主对象自定义验证**
   - 执行主对象的 CustomValidate

4. **嵌套可验证对象**
   - 执行实现了 NestedValidatable 接口的字段的 ValidateNested

## 自动嵌套验证

### 示例 1: 嵌入的 BaseModel

```go
type BaseModel struct {
    ID     int64        `json:"id"`
    Status types.Status `json:"status"`
    Extras types.Extras `json:"extras,omitempty"` // map[string]any
}

func (m *BaseModel) ValidateRules() map[ValidateScene]map[string]string {
    return map[ValidateScene]map[string]string{
        SceneUpdate: {
            "ID": "omitempty,gt=0",
        },
    }
}

type Product struct {
    BaseModel  // 嵌入 BaseModel
    Name       string  `json:"name"`
    Price      float64 `json:"price"`
}

func (p *Product) ValidateRules() map[ValidateScene]map[string]string {
    return map[ValidateScene]map[string]string{
        SceneCreate: {
            "Name":  "required,min=2,max=100",
            "Price": "required,gt=0",
        },
    }
}

// 验证时，BaseModel 的规则会自动被应用
product := &Product{
    BaseModel: BaseModel{ID: 123},
    Name:      "iPhone",
    Price:     999.99,
}

err := validator.Validate(product, validator.SceneUpdate)
// 会同时验证 Product 和 BaseModel 的规则
```

### 示例 2: 嵌套的结构体字段

```go
type Category struct {
    ID   int64  `json:"id"`
    Name string `json:"name"`
}

func (c *Category) ValidateRules() map[ValidateScene]map[string]string {
    return map[ValidateScene]map[string]string{
        SceneCreate: {
            "Name": "required,min=2,max=50",
        },
    }
}

type Product struct {
    ID       int64     `json:"id"`
    Name     string    `json:"name"`
    Category *Category `json:"category"` // 嵌套的 Category
}

func (p *Product) ValidateRules() map[ValidateScene]map[string]string {
    return map[ValidateScene]map[string]string{
        SceneCreate: {
            "Name": "required,min=2,max=100",
        },
    }
}

// 验证时，Category 会自动被验证
product := &Product{
    Name: "iPhone",
    Category: &Category{
        Name: "电子产品",
    },
}

err := validator.Validate(product, validator.SceneCreate)
// 会同时验证 Product 和 Category
```

## 手动嵌套验证

### 使用 CustomValidate 验证动态数据

对于 map、slice 等动态数据结构，通常在 CustomValidate 中手动验证。

```go
type Product struct {
    BaseModel
    Name     string  `json:"name"`
    Category string  `json:"category"`
    Extras   types.Extras `json:"extras,omitempty"` // map[string]any
}

func (p *Product) CustomValidate(scene ValidateScene) error {
    if scene == SceneCreate {
        // 根据类别验证不同的 extras 字段
        switch p.Category {
        case "electronics":
            return p.validateElectronicsExtras()
        case "clothing":
            return p.validateClothingExtras()
        }
    }
    return nil
}

func (p *Product) validateElectronicsExtras() error {
    if p.Extras == nil {
        return fmt.Errorf("电子产品必须提供额外属性")
    }
    
    // 使用 MapValidator
    validator := &MapValidator{
        RequiredKeys: []string{"brand", "warranty"},
        AllowedKeys:  []string{"brand", "warranty", "model"},
        KeyValidators: map[string]func(value interface{}) error{
            "brand": func(value interface{}) error {
                str, ok := value.(string)
                if !ok || len(str) < 2 {
                    return fmt.Errorf("品牌名称长度必须至少2个字符")
                }
                return nil
            },
            "warranty": func(value interface{}) error {
                months, ok := value.(int)
                if !ok || months < 1 || months > 60 {
                    return fmt.Errorf("保修期必须在1-60个月之间")
                }
                return nil
            },
        },
    }
    
    return ValidateMap(p.Extras, validator)
}

func (p *Product) validateClothingExtras() error {
    if p.Extras == nil {
        return fmt.Errorf("服装必须提供额外属性")
    }
    
    // 检查必填键
    if err := ValidateMapMustHaveKeys(p.Extras, "size", "color"); err != nil {
        return err
    }
    
    // 验证 size 枚举
    if err := ValidateMapKey(p.Extras, "size", func(value interface{}) error {
        size, ok := value.(string)
        if !ok {
            return fmt.Errorf("尺码必须是字符串类型")
        }
        validSizes := map[string]bool{
            "XS": true, "S": true, "M": true, 
            "L": true, "XL": true, "XXL": true,
        }
        if !validSizes[size] {
            return fmt.Errorf("尺码必须是 XS, S, M, L, XL, XXL 之一")
        }
        return nil
    }); err != nil {
        return err
    }
    
    // 验证 color 长度
    return ValidateMapStringKey(p.Extras, "color", 2, 20)
}
```

## 多层嵌套验证

### 示例：订单 -> 产品 -> 基础模型

```go
type Order struct {
    BaseModel
    UserID   int64     `json:"user_id"`
    Products []*Product `json:"products"`
    Total    float64   `json:"total"`
}

func (o *Order) ValidateRules() map[ValidateScene]map[string]string {
    return map[ValidateScene]map[string]string{
        SceneCreate: {
            "UserID": "required,gt=0",
            "Total":  "required,gt=0",
        },
    }
}

func (o *Order) CustomValidate(scene ValidateScene) error {
    if scene == SceneCreate {
        // 验证至少有一个产品
        if len(o.Products) == 0 {
            return fmt.Errorf("订单至少需要一个产品")
        }
        
        // 手动验证每个产品（也可以依赖自动嵌套验证）
        for i, product := range o.Products {
            if err := validator.Validate(product, scene); err != nil {
                return fmt.Errorf("产品 #%d 验证失败: %w", i+1, err)
            }
        }
        
        // 验证总价
        var sum float64
        for _, product := range o.Products {
            sum += product.Price
        }
        if sum != o.Total {
            return fmt.Errorf("订单总价不匹配")
        }
    }
    return nil
}
```

## 使用 NestedValidatable 接口

### 示例：用户资料验证

```go
type UserProfile struct {
    BaseModel
    UserID      int64        `json:"user_id"`
    Bio         string       `json:"bio"`
    SocialLinks types.Extras `json:"social_links,omitempty"` // map[string]any
}

func (up *UserProfile) ValidateRules() map[ValidateScene]map[string]string {
    return map[ValidateScene]map[string]string{
        SceneCreate: {
            "UserID": "required,gt=0",
            "Bio":    "omitempty,max=500",
        },
    }
}

// 实现 NestedValidatable 接口
func (up *UserProfile) ValidateNested(scene ValidateScene) error {
    if up.SocialLinks == nil {
        return nil
    }
    
    // 验证社交媒体链接
    validator := &MapValidator{
        AllowedKeys: []string{"twitter", "github", "linkedin", "website"},
        KeyValidators: map[string]func(value interface{}) error{
            "twitter": func(value interface{}) error {
                url, ok := value.(string)
                if !ok {
                    return fmt.Errorf("twitter 链接必须是字符串")
                }
                if !strings.HasPrefix(url, "https://twitter.com/") {
                    return fmt.Errorf("twitter 链接格式不正确")
                }
                return nil
            },
            "github": func(value interface{}) error {
                url, ok := value.(string)
                if !ok {
                    return fmt.Errorf("github 链接必须是字符串")
                }
                if !strings.HasPrefix(url, "https://github.com/") {
                    return fmt.Errorf("github 链接格式不正确")
                }
                return nil
            },
        },
    }
    
    return ValidateMap(up.SocialLinks, validator)
}
```

## 完整示例：电商产品模型

```go
package main

import (
    "fmt"
    "katydid-common-account/pkg/validator"
    "katydid-common-account/pkg/types"
)

// BaseModel 基础模型
type BaseModel struct {
    ID     int64        `json:"id"`
    Status types.Status `json:"status"`
    Extras types.Extras `json:"extras,omitempty"`
}

func (m *BaseModel) ValidateRules() map[validator.ValidateScene]map[string]string {
    return map[validator.ValidateScene]map[string]string{
        validator.SceneUpdate: {
            "ID": "omitempty,gt=0",
        },
    }
}

// Product 产品模型
type Product struct {
    BaseModel
    Name     string  `json:"name"`
    Price    float64 `json:"price"`
    Stock    int     `json:"stock"`
    Category string  `json:"category"`
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

func (p *Product) CustomValidate(scene validator.ValidateScene) error {
    if scene == validator.SceneCreate {
        // 根据类别验证 extras
        switch p.Category {
        case "electronics":
            return p.validateElectronicsExtras()
        case "clothing":
            return p.validateClothingExtras()
        }
    }
    return nil
}

func (p *Product) validateElectronicsExtras() error {
    if p.Extras == nil {
        return fmt.Errorf("电子产品必须提供额外属性（品牌、保修期等）")
    }
    
    // 必填键验证
    if err := validator.ValidateMapMustHaveKeys(p.Extras, "brand", "warranty"); err != nil {
        return err
    }
    
    // 品牌验证
    if err := validator.ValidateMapStringKey(p.Extras, "brand", 2, 50); err != nil {
        return err
    }
    
    // 保修期验证
    if err := validator.ValidateMapIntKey(p.Extras, "warranty", 1, 60); err != nil {
        return err
    }
    
    return nil
}

func (p *Product) validateClothingExtras() error {
    if p.Extras == nil {
        return fmt.Errorf("服装必须提供额外属性（尺码、颜色等）")
    }
    
    // 必填键
    if err := validator.ValidateMapMustHaveKeys(p.Extras, "size", "color"); err != nil {
        return err
    }
    
    // 尺码枚举验证
    if err := validator.ValidateMapKey(p.Extras, "size", func(value interface{}) error {
        size, ok := value.(string)
        if !ok {
            return fmt.Errorf("尺码必须是字符串类型")
        }
        validSizes := map[string]bool{
            "XS": true, "S": true, "M": true,
            "L": true, "XL": true, "XXL": true,
        }
        if !validSizes[size] {
            return fmt.Errorf("尺码必须是 XS, S, M, L, XL, XXL 之一")
        }
        return nil
    }); err != nil {
        return err
    }
    
    // 颜色验证
    return validator.ValidateMapStringKey(p.Extras, "color", 2, 20)
}

func main() {
    // 测试电子产品
    electronics := &Product{
        Name:     "iPhone 15",
        Price:    999.99,
        Stock:    100,
        Category: "electronics",
        BaseModel: BaseModel{
            Extras: types.Extras{
                "brand":    "Apple",
                "warranty": 24,
                "model":    "A2846",
            },
        },
    }
    
    if err := validator.Validate(electronics, validator.SceneCreate); err != nil {
        fmt.Printf("电子产品验证失败: %v\n", err)
        return
    }
    fmt.Println("电子产品验证通过")
    
    // 测试服装产品
    clothing := &Product{
        Name:     "T恤",
        Price:    29.99,
        Stock:    200,
        Category: "clothing",
        BaseModel: BaseModel{
            Extras: types.Extras{
                "size":     "L",
                "color":    "蓝色",
                "material": "棉",
            },
        },
    }
    
    if err := validator.Validate(clothing, validator.SceneCreate); err != nil {
        fmt.Printf("服装产品验证失败: %v\n", err)
        return
    }
    fmt.Println("服装产品验证通过")
}
```

## 最佳实践

### 1. 分离验证逻辑

将不同类型的嵌套验证逻辑提取为独立方法。

```go
// 推荐：分离验证方法
func (p *Product) CustomValidate(scene ValidateScene) error {
    switch p.Category {
    case "electronics":
        return p.validateElectronicsExtras()
    case "clothing":
        return p.validateClothingExtras()
    }
    return nil
}

// 不推荐：所有逻辑堆在一起
func (p *Product) CustomValidate(scene ValidateScene) error {
    if p.Category == "electronics" {
        if p.Extras == nil {
            return fmt.Errorf("...")
        }
        // 一大堆验证代���...
    } else if p.Category == "clothing" {
        // 又一大堆验证代码...
    }
    return nil
}
```

### 2. 优先使用自动嵌套验证

对于结构化的嵌套对象，优先依赖自动验证。

```go
// 推荐：定义 Category 的验证规则，让自动验证处理
type Category struct {
    ID   int64  `json:"id"`
    Name string `json:"name"`
}

func (c *Category) ValidateRules() map[ValidateScene]map[string]string {
    return map[ValidateScene]map[string]string{
        SceneCreate: {"Name": "required,min=2,max=50"},
    }
}

// 不推荐：手动验证
func (p *Product) CustomValidate(scene ValidateScene) error {
    if p.Category != nil {
        if p.Category.Name == "" {
            return fmt.Errorf("分类名称不能为空")
        }
        // 更多验证...
    }
    return nil
}
```

### 3. 使用 MapValidator 验证动态数据

```go
// 推荐：使用 MapValidator
validator := &MapValidator{
    RequiredKeys: []string{"brand"},
    AllowedKeys:  []string{"brand", "model"},
}
ValidateMap(p.Extras, validator)

// 不推荐：手动检查每个键
if _, ok := p.Extras["brand"]; !ok {
    return fmt.Errorf("缺少 brand")
}
if val, ok := p.Extras["invalid"]; ok {
    return fmt.Errorf("不允许的键: invalid")
}
```

### 4. 提供清晰的错误路径

```go
// 推荐：明确指出哪个嵌套字段出错
func (o *Order) CustomValidate(scene ValidateScene) error {
    for i, product := range o.Products {
        if err := validator.Validate(product, scene); err != nil {
            return fmt.Errorf("产品 #%d 验证失败: %w", i+1, err)
        }
    }
    return nil
}
```

### 5. 合理使用验证时机

```go
// 在 CustomValidate 中验证业务逻辑
func (p *Product) CustomValidate(scene ValidateScene) error {
    if scene == SceneCreate {
        // 创建时的特殊验证
        return p.validateExtrasForCreate()
    }
    return nil
}

// 在 ValidateNested 中验证复杂嵌套结构
func (p *Product) ValidateNested(scene ValidateScene) error {
    // 验证所有嵌套的动态数据
    return p.validateAllNestedData()
}
```

## 常见场景

### 场景 1: 带有动态属性的基础模型

所有模型都继承 BaseModel，每个模型根据业务需求验证不同的 Extras。

```go
type BaseModel struct {
    ID     int64        `json:"id"`
    Extras types.Extras `json:"extras,omitempty"`
}

type Product struct {
    BaseModel
    // 产品特有字段...
}

func (p *Product) CustomValidate(scene ValidateScene) error {
    // 验证产品特定的 extras
    return validateProductExtras(p.Extras)
}
```

### 场景 2: 多对多关系验证

```go
type Article struct {
    ID   int64  `json:"id"`
    Tags []*Tag `json:"tags"`
}

func (a *Article) CustomValidate(scene ValidateScene) error {
    if scene == SceneCreate {
        if len(a.Tags) == 0 {
            return fmt.Errorf("文章至少需要一个标签")
        }
        
        // 验证每个标签
        for i, tag := range a.Tags {
            if err := validator.Validate(tag, scene); err != nil {
                return fmt.Errorf("标签 #%d 验证失败: %w", i+1, err)
            }
        }
    }
    return nil
}
```

### 场景 3: 条件性嵌套验证

根据条件决定是否验证嵌套数据。

```go
func (p *Product) CustomValidate(scene ValidateScene) error {
    // 只有特定类别才验证 extras
    if p.Category == "special" && p.Extras != nil {
        return validateSpecialExtras(p.Extras)
    }
    return nil
}
```

## 性能优化

### 1. 避免重复验证

```go
// 如果已经依赖自动嵌套验证，不要再手动验证
func (o *Order) CustomValidate(scene ValidateScene) error {
    // 不需要：Products 会被自动验证
    // for _, p := range o.Products {
    //     validator.Validate(p, scene)
    // }
    
    // 只验证业务逻辑
    return o.validateBusinessRules()
}
```

### 2. 缓存验证器实例

```go
var (
    electronicsValidator *MapValidator
    clothingValidator    *MapValidator
)

func init() {
    electronicsValidator = NewMapValidator().
        WithRequiredKeys("brand", "warranty").
        WithAllowedKeys("brand", "warranty", "model")
    
    clothingValidator = NewMapValidator().
        WithRequiredKeys("size", "color").
        WithAllowedKeys("size", "color", "material")
}
```

## 相关文档

- [VALIDATOR_README.md](./VALIDATOR_README.md) - 主验证器文档
- [MAP_VALIDATOR_README.md](./MAP_VALIDATOR_README.md) - Map 验证器文档

