# Map Validator 映射验证器

## 概述

`MapValidator` 是专门用于验证 `map[string]any` 类型数据的验证工具，特别适合验证动态的、非结构化的键值对数据（如 JSON extras 字段）。

## 核心类型

### MapValidator 结构

```go
type MapValidator struct {
    RequiredKeys  []string                                  // 必填键列表
    AllowedKeys   []string                                  // 允许的键列表（白名单）
    KeyValidators map[string]func(value interface{}) error  // 键值验证器
}
```

## 核心功能

### 1. 必填键验证
验证 map 中必须包含指定的键。

### 2. 键白名单验证
限制 map 只能包含指定的键，防止非法键进入。

### 3. 自定义键值验证
为特定的键提供自定义验证函数。

## 核心函数

### ValidateMap

验证 map 数据的主函数。

```go
func ValidateMap(kvs map[string]any, v *MapValidator) error
```

**验证流程：**
1. 检查所有必填键是否存在
2. 检查是否存在不允许的键
3. 对指定的键执行自定义验证

**使用示例：**

```go
extras := map[string]any{
    "brand":    "Apple",
    "warranty": 24,
    "color":    "Silver",
}

validator := &MapValidator{
    RequiredKeys: []string{"brand", "warranty"},
    AllowedKeys:  []string{"brand", "warranty", "color"},
    KeyValidators: map[string]func(value interface{}) error{
        "brand": func(value interface{}) error {
            str, ok := value.(string)
            if !ok || len(str) < 2 {
                return fmt.Errorf("brand 必须是长度至少2的字符串")
            }
            return nil
        },
    },
}

err := ValidateMap(extras, validator)
```

### ValidateMapKey

验证 map 中特定键的值。

```go
func ValidateMapKey(kvs map[string]any, key string, validatorFunc func(value interface{}) error) error
```

**特点：**
- 键不存在时不报错（可选键）
- 键存在时执行验证函数

**使用示例：**

```go
err := ValidateMapKey(extras, "size", func(value interface{}) error {
    size, ok := value.(string)
    if !ok {
        return fmt.Errorf("size 必须是字符串类型")
    }
    validSizes := map[string]bool{"S": true, "M": true, "L": true, "XL": true}
    if !validSizes[size] {
        return fmt.Errorf("size 必须是 S, M, L, XL 之一")
    }
    return nil
})
```

### ValidateMapMustHaveKey

验证 map 必须包含指定的键。

```go
func ValidateMapMustHaveKey(kvs map[string]any, key string) error
```

**使用示例：**

```go
if err := ValidateMapMustHaveKey(extras, "brand"); err != nil {
    return err // "map 必须包含键: brand"
}
```

### ValidateMapMustHaveKeys

验证 map 必须包含多个指定的键。

```go
func ValidateMapMustHaveKeys(kvs map[string]any, keys ...string) error
```

**使用示例：**

```go
err := ValidateMapMustHaveKeys(extras, "brand", "warranty", "model")
```

### 类型特定验证函数

#### ValidateMapStringKey

验证字符串类型的键值及长度。

```go
func ValidateMapStringKey(kvs map[string]any, key string, minLen, maxLen int) error
```

**参数：**
- `minLen`: 最小长度（0 表示不限制）
- `maxLen`: 最大长度（0 表示不限制）

**使用示例：**

```go
// 验证 color 字段长度在 2-20 之间
err := ValidateMapStringKey(extras, "color", 2, 20)
```

#### ValidateMapIntKey

验证整数类型的键值及范围。

```go
func ValidateMapIntKey(kvs map[string]any, key string, min, max int) error
```

**特点：**
- 自动转换 int、int64、float64 类型

**使用示例：**

```go
// 验证 warranty 在 1-60 个月之间
err := ValidateMapIntKey(extras, "warranty", 1, 60)
```

#### ValidateMapFloatKey

验证浮点数类型的键值及范围。

```go
func ValidateMapFloatKey(kvs map[string]any, key string, min, max float64) error
```

**特点：**
- 支持 float64、float32、int、int64 自动转换

**使用示例：**

```go
// 验证 rating 在 0.0-5.0 之间
err := ValidateMapFloatKey(extras, "rating", 0.0, 5.0)
```

#### ValidateMapBoolKey

验证布尔类型的键值。

```go
func ValidateMapBoolKey(kvs map[string]any, key string) error
```

**使用示例：**

```go
err := ValidateMapBoolKey(extras, "is_featured")
```

## MapValidator 链式构建

### NewMapValidator

创建新的 MapValidator 实例。

```go
func NewMapValidator() *MapValidator
```

### 链式方法

#### WithRequiredKeys

设置必填键列表。

```go
func (ev *MapValidator) WithRequiredKeys(keys ...string) *MapValidator
```

#### WithAllowedKeys

设置允许的键列表。

```go
func (ev *MapValidator) WithAllowedKeys(keys ...string) *MapValidator
```

#### WithKeyValidator

添加键验证器。

```go
func (ev *MapValidator) WithKeyValidator(key string, validatorFunc func(value interface{}) error) *MapValidator
```

#### AddRequiredKey

添加单个必填键。

```go
func (ev *MapValidator) AddRequiredKey(key string) *MapValidator
```

#### AddAllowedKey

添加单个允许的键。

```go
func (ev *MapValidator) AddAllowedKey(key string) *MapValidator
```

### 链式构建示例

```go
validator := NewMapValidator().
    WithRequiredKeys("brand", "warranty").
    WithAllowedKeys("brand", "warranty", "color", "model").
    WithKeyValidator("brand", func(value interface{}) error {
        str, ok := value.(string)
        if !ok || len(str) < 2 {
            return fmt.Errorf("brand 长度必须至少2个字符")
        }
        return nil
    }).
    WithKeyValidator("warranty", func(value interface{}) error {
        // 自定义验证逻辑
        return nil
    })

err := ValidateMap(extras, validator)
```

## 完整使用示例

### 示例 1: 电子产品验证

```go
func validateElectronicsExtras(extras map[string]any) error {
    // 方式1: 使用 ValidateMap
    validator := &MapValidator{
        RequiredKeys: []string{"brand", "warranty"},
        AllowedKeys:  []string{"brand", "warranty", "model", "color"},
        KeyValidators: map[string]func(value interface{}) error{
            "brand": func(value interface{}) error {
                str, ok := value.(string)
                if !ok || len(str) < 2 || len(str) > 50 {
                    return fmt.Errorf("brand 长度必须在 2-50 之间")
                }
                return nil
            },
            "warranty": func(value interface{}) error {
                months, ok := value.(int)
                if !ok || months < 1 || months > 60 {
                    return fmt.Errorf("warranty 必须在 1-60 个月之间")
                }
                return nil
            },
        },
    }
    return ValidateMap(extras, validator)
}
```

### 示例 2: 服装产品验证

```go
func validateClothingExtras(extras map[string]any) error {
    // 检查必填键
    if err := ValidateMapMustHaveKeys(extras, "size", "color"); err != nil {
        return err
    }
    
    // 验证 size 枚举值
    if err := ValidateMapKey(extras, "size", func(value interface{}) error {
        size, ok := value.(string)
        if !ok {
            return fmt.Errorf("size 必须是字符串类型")
        }
        validSizes := map[string]bool{
            "XS": true, "S": true, "M": true, 
            "L": true, "XL": true, "XXL": true,
        }
        if !validSizes[size] {
            return fmt.Errorf("size 必须是 XS, S, M, L, XL, XXL 之一")
        }
        return nil
    }); err != nil {
        return err
    }
    
    // 验证 color 长度
    if err := ValidateMapStringKey(extras, "color", 2, 20); err != nil {
        return err
    }
    
    // 可选字段: material
    if err := ValidateMapStringKey(extras, "material", 2, 30); err != nil {
        return err
    }
    
    return nil
}
```

### 示例 3: 用户社交媒体链接验证

```go
func validateSocialLinks(extras map[string]any) error {
    validator := NewMapValidator().
        WithAllowedKeys("twitter", "github", "linkedin", "website").
        WithKeyValidator("twitter", func(value interface{}) error {
            url, ok := value.(string)
            if !ok {
                return fmt.Errorf("twitter 必须是字符串类型")
            }
            if !strings.HasPrefix(url, "https://twitter.com/") {
                return fmt.Errorf("twitter 链接格式不正确")
            }
            return nil
        }).
        WithKeyValidator("github", func(value interface{}) error {
            url, ok := value.(string)
            if !ok {
                return fmt.Errorf("github 必须是字符串类型")
            }
            if !strings.HasPrefix(url, "https://github.com/") {
                return fmt.Errorf("github 链接格式不正确")
            }
            return nil
        })
    
    return ValidateMap(extras, validator)
}
```

### 示例 4: 在模型的 CustomValidate 中使用

```go
type Product struct {
    ID       int64        `json:"id"`
    Name     string       `json:"name"`
    Category string       `json:"category"`
    Extras   types.Extras `json:"extras,omitempty"` // map[string]any
}

func (p *Product) CustomValidate(scene validator.ValidateScene) error {
    if scene == validator.SceneCreate {
        // 根据类别验证不同的 extras
        switch p.Category {
        case "electronics":
            return validateElectronicsExtras(p.Extras)
        case "clothing":
            return validateClothingExtras(p.Extras)
        case "food":
            return validateFoodExtras(p.Extras)
        }
    }
    return nil
}
```

## 常见使用模式

### 模式 1: 严格白名单模式

只允许特定的键，其他键一律拒绝。

```go
validator := &MapValidator{
    AllowedKeys: []string{"key1", "key2", "key3"},
}
```

### 模式 2: 必填字段模式

某些键必须存在。

```go
validator := &MapValidator{
    RequiredKeys: []string{"name", "email"},
}
```

### 模式 3: 组合验证模式

结合必填、白名单和自定义验证。

```go
validator := &MapValidator{
    RequiredKeys: []string{"name"},
    AllowedKeys:  []string{"name", "age", "email"},
    KeyValidators: map[string]func(value interface{}) error{
        "email": validateEmail,
        "age":   validateAge,
    },
}
```

### 模式 4: 逐个验证模式

不使用 MapValidator，直接调用验证函数。

```go
// 检查必填
if err := ValidateMapMustHaveKey(extras, "name"); err != nil {
    return err
}

// 类型验证
if err := ValidateMapStringKey(extras, "name", 2, 50); err != nil {
    return err
}

// 自定义验证
if err := ValidateMapKey(extras, "status", validateStatus); err != nil {
    return err
}
```

## 最佳实践

### 1. 优先使用类型特定函数

```go
// 推荐：使用类型特定函数
err := ValidateMapStringKey(extras, "name", 2, 50)

// 不推荐：自己写类型检查
err := ValidateMapKey(extras, "name", func(value interface{}) error {
    str, ok := value.(string)
    if !ok {
        return fmt.Errorf("必须是字符串")
    }
    if len(str) < 2 || len(str) > 50 {
        return fmt.Errorf("长度必须在 2-50 之间")
    }
    return nil
})
```

### 2. 使用白名单而非黑名单

```go
// 推荐：白名单模式，明确允许的键
AllowedKeys: []string{"brand", "model", "color"}

// 不推荐：不限制键，容易引入不安全数据
```

### 3. 分离验证逻辑

将复杂的验证逻辑提取为独立函数。

```go
func validateBrand(value interface{}) error {
    str, ok := value.(string)
    if !ok {
        return fmt.Errorf("brand 必须是字符串类型")
    }
    if len(str) < 2 || len(str) > 50 {
        return fmt.Errorf("brand 长度必须在 2-50 之间")
    }
    // 可以添加更多验证逻辑
    return nil
}

// 使用
KeyValidators: map[string]func(value interface{}) error{
    "brand": validateBrand,
}
```

### 4. 提供清晰的错误信息

```go
func validateSize(value interface{}) error {
    size, ok := value.(string)
    if !ok {
        return fmt.Errorf("尺码必须是字符串类型")
    }
    validSizes := []string{"XS", "S", "M", "L", "XL", "XXL"}
    for _, valid := range validSizes {
        if size == valid {
            return nil
        }
    }
    return fmt.Errorf("尺码必须是以下之一: %s，当前值: %s", 
        strings.Join(validSizes, ", "), size)
}
```

## 错误处理

### 常见错误信息

- `map 必须包含键: brand`
- `map 不允许包含键: invalid_key`
- `键 'name' 必须是字符串类型`
- `键 'age' 的值不能小于 0`
- `键 'email' 必须是有效的邮箱地址`

### 错误处理示例

```go
if err := ValidateMap(extras, validator); err != nil {
    // 记录日志
    log.Printf("extras 验证失败: %v", err)
    
    // 返回友好的错误信息给用户
    return fmt.Errorf("产品额外属性验证失败: %v", err)
}
```

## 与 Validator 集成

Map Validator 通常在 `CustomValidate` 方法中使用：

```go
func (p *Product) CustomValidate(scene validator.ValidateScene) error {
    if p.Extras != nil {
        // 使用 MapValidator 验证 extras
        v := NewMapValidator().
            WithRequiredKeys("brand").
            WithAllowedKeys("brand", "model", "color")
        
        if err := ValidateMap(p.Extras, v); err != nil {
            return fmt.Errorf("extras 验证失败: %w", err)
        }
    }
    return nil
}
```

## 性能考虑

- 每次验证都会遍历 map，复杂度为 O(n)
- 自定义验证函数会被多次调用，应避免耗时操作
- 对于大量重复验证，可考虑缓存 MapValidator 实例

## 相关文档

- [VALIDATOR_README.md](./VALIDATOR_README.md) - 主验证器文档
- [NESTED_VALIDATION_README.md](./NESTED_VALIDATION_README.md) - 嵌套验证文档

