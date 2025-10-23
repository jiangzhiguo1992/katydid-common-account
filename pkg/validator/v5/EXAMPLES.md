# Validator v5 使用示例

## 基础使用

### 1. 最简单的使用方式

```go
package main

import (
    "fmt"
    "validator/v5"
)

type User struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

// 实现 RuleProvider 接口
func (u *User) GetRules(scene v5.Scene) map[string]string {
    if scene == v5.SceneCreate {
        return map[string]string{
            "Username": "required,min=3,max=20",
            "Email":    "required,email",
            "Password": "required,min=6",
        }
    }
    return nil
}

func main() {
    user := &User{
        Username: "john",
        Email:    "john@example.com",
        Password: "password123",
    }

    // 使用默认验证器
    err := v5.Validate(user, v5.SceneCreate)
    if err != nil {
        fmt.Printf("验证失败: %v\n", err)
        return
    }

    fmt.Println("验证通过")
}
```

### 2. 场景化验证

```go
type Product struct {
    Name  string  `json:"name"`
    Price float64 `json:"price"`
    Stock int     `json:"stock"`
}

func (p *Product) GetRules(scene v5.Scene) map[string]string {
    switch scene {
    case v5.SceneCreate:
        return map[string]string{
            "Name":  "required,min=2,max=100",
            "Price": "required,gt=0",
            "Stock": "required,gte=0",
        }
    case v5.SceneUpdate:
        return map[string]string{
            "Name":  "omitempty,min=2,max=100",
            "Price": "omitempty,gt=0",
            "Stock": "omitempty,gte=0",
        }
    default:
        return nil
    }
}

func main() {
    product := &Product{Name: "iPhone", Price: 999.99}

    // 创建场景：所有字段必填
    err := v5.Validate(product, v5.SceneCreate)
    if err != nil {
        fmt.Printf("创建验证失败: %v\n", err)
    }

    // 更新场景：字段可选
    err = v5.Validate(product, v5.SceneUpdate)
    if err == nil {
        fmt.Println("更新验证通过")
    }
}
```

### 3. 业务逻辑验证

```go
type Order struct {
    OrderNo    string  `json:"order_no"`
    TotalPrice float64 `json:"total_price"`
    Discount   float64 `json:"discount"`
}

func (o *Order) GetRules(scene v5.Scene) map[string]string {
    return map[string]string{
        "OrderNo":    "required",
        "TotalPrice": "required,gt=0",
        "Discount":   "gte=0",
    }
}

// 实现 BusinessValidator 接口
func (o *Order) ValidateBusiness(ctx *v5.ValidationContext) error {
    // 业务规则：折扣不能大于总价
    if o.Discount > o.TotalPrice {
        ctx.AddError(v5.NewFieldError("Order.Discount", "Discount", "invalid_discount").
            WithMessage("折扣金额不能大于总价"))
    }

    // 业务规则：订单号格式检查
    if len(o.OrderNo) < 10 {
        ctx.AddError(v5.NewFieldError("Order.OrderNo", "OrderNo", "invalid_format").
            WithMessage("订单号格式不正确"))
    }

    return nil
}

func main() {
    order := &Order{
        OrderNo:    "ORD20231023001",
        TotalPrice: 100.0,
        Discount:   150.0, // 错误：折扣大于总价
    }

    err := v5.Validate(order, v5.SceneCreate)
    if err != nil {
        if ve, ok := err.(*v5.ValidationError); ok {
            for _, e := range ve.Errors {
                fmt.Printf("错误：%s\n", e.Message)
            }
        }
    }
}
```

## 高级使用

### 4. 使用构建器创建自定义验证器

```go
func main() {
    // 创建自定义验证器
    validator := v5.NewValidatorBuilder().
        WithRuleStrategy().
        WithBusinessStrategy().
        WithMaxDepth(50).
        WithMaxErrors(100).
        Build()

    user := &User{Username: "john"}
    err := validator.Validate(user, v5.SceneCreate)
    if err != nil {
        fmt.Printf("验证失败: %v\n", err)
    }
}
```

### 5. 添加验证监听器

```go
type MyLogger struct{}

func (l *MyLogger) Debug(msg string, args ...any) {
    fmt.Printf("[DEBUG] %s %v\n", msg, args)
}

func (l *MyLogger) Info(msg string, args ...any) {
    fmt.Printf("[INFO] %s %v\n", msg, args)
}

func (l *MyLogger) Warn(msg string, args ...any) {
    fmt.Printf("[WARN] %s %v\n", msg, args)
}

func (l *MyLogger) Error(msg string, args ...any) {
    fmt.Printf("[ERROR] %s %v\n", msg, args)
}

func main() {
    // 创建带日志的验证器
    logger := &MyLogger{}
    listener := v5.NewLoggingListener(logger)

    validator := v5.NewValidatorBuilder().
        WithRuleStrategy().
        WithBusinessStrategy().
        WithListener(listener).
        Build()

    user := &User{Username: "john"}
    _ = validator.Validate(user, v5.SceneCreate)
    // 会输出验证过程的日志
}
```

### 6. 使用指标监听器

```go
func main() {
    // 创建指标监听器
    metrics := v5.NewMetricsListener()

    validator := v5.NewValidatorBuilder().
        WithRuleStrategy().
        WithBusinessStrategy().
        WithListener(metrics).
        Build()

    // 执行多次验证
    for i := 0; i < 100; i++ {
        user := &User{Username: "john"}
        _ = validator.Validate(user, v5.SceneCreate)
    }

    // 获取指标
    validationCount, errorCount := metrics.GetMetrics()
    fmt.Printf("验证次数: %d, 错误数: %d\n", validationCount, errorCount)
}
```

### 7. 部分字段验证

```go
func main() {
    user := &User{
        Username: "ab",                    // 太短（不验证）
        Email:    "valid@example.com",    // 有效（要验证）
        Password: "123",                   // 太短（不验证）
    }

    // 只验证 Email 字段
    err := v5.ValidateFields(user, v5.SceneCreate, "Email")
    if err == nil {
        fmt.Println("Email 验证通过")
    }
}
```

### 8. 排除字段验证

```go
func main() {
    user := &User{
        Username: "john",
        Email:    "john@example.com",
        Password: "", // 密码为空，但我们要排除它
    }

    // 验证除了 Password 外的所有字段
    err := v5.ValidateExcept(user, v5.SceneCreate, "Password")
    if err == nil {
        fmt.Println("验证通过（已排除 Password）")
    }
}
```

### 9. 生命周期钩子

```go
type Article struct {
    Title   string `json:"title"`
    Content string `json:"content"`
}

func (a *Article) GetRules(scene v5.Scene) map[string]string {
    return map[string]string{
        "Title":   "required,min=5",
        "Content": "required,min=10",
    }
}

// 实现 LifecycleHooks 接口
func (a *Article) BeforeValidation(ctx *v5.ValidationContext) error {
    // 验证前的预处理
    a.Title = strings.TrimSpace(a.Title)
    a.Content = strings.TrimSpace(a.Content)
    fmt.Println("验证前：清理空白字符")
    return nil
}

func (a *Article) AfterValidation(ctx *v5.ValidationContext) error {
    // 验证后的处理
    if !ctx.HasErrors() {
        fmt.Println("验证后：数据有效，可以保存")
    }
    return nil
}

func main() {
    article := &Article{
        Title:   "  Hello World  ",
        Content: "  This is content  ",
    }

    err := v5.Validate(article, v5.SceneCreate)
    if err == nil {
        fmt.Printf("标题: '%s'\n", article.Title)
    }
}
```

### 10. 自定义验证策略

```go
// 自定义策略：检查敏感词
type SensitiveWordStrategy struct {
    bannedWords []string
}

func NewSensitiveWordStrategy(words []string) *SensitiveWordStrategy {
    return &SensitiveWordStrategy{bannedWords: words}
}

func (s *SensitiveWordStrategy) Name() string {
    return "sensitive_word"
}

func (s *SensitiveWordStrategy) Priority() int {
    return 40 // 在业务验证之后
}

func (s *SensitiveWordStrategy) Validate(target any, ctx *v5.ValidationContext) error {
    // 使用反射检查所有字符串字段
    val := reflect.ValueOf(target)
    if val.Kind() == reflect.Ptr {
        val = val.Elem()
    }

    if val.Kind() != reflect.Struct {
        return nil
    }

    typ := val.Type()
    for i := 0; i < val.NumField(); i++ {
        field := val.Field(i)
        if field.Kind() == reflect.String {
            text := field.String()
            for _, word := range s.bannedWords {
                if strings.Contains(text, word) {
                    ctx.AddError(v5.NewFieldError(
                        typ.Field(i).Name,
                        typ.Field(i).Name,
                        "sensitive_word",
                    ).WithMessage(fmt.Sprintf("包含敏感词: %s", word)))
                }
            }
        }
    }

    return nil
}

func main() {
    // 创建带自定义策略的验证器
    sensitiveStrategy := NewSensitiveWordStrategy([]string{"spam", "scam"})

    validator := v5.NewValidatorBuilder().
        WithRuleStrategy().
        WithBusinessStrategy().
        WithStrategy(sensitiveStrategy).
        Build()

    user := &User{
        Username: "john",
        Email:    "spam@example.com", // 包含敏感词
        Password: "password123",
    }

    err := validator.Validate(user, v5.SceneCreate)
    if err != nil {
        fmt.Printf("验证失败: %v\n", err)
    }
}
```

## 性能优化技巧

### 11. 使用工厂创建验证器

```go
func main() {
    factory := v5.NewValidatorFactory()

    // 默认验证器（功能完整）
    defaultValidator := factory.CreateDefault()

    // 最小验证器（只有规则验证，性能最优）
    minimalValidator := factory.CreateMinimal()

    // 根据场景选择
    validator := minimalValidator // 高性能场景
    // validator := defaultValidator // 功能完整场景

    user := &User{Username: "john"}
    _ = validator.Validate(user, v5.SceneCreate)
}
```

### 12. 清除缓存

```go
func main() {
    // 在测试中清除缓存
    v5.ClearCache()

    // 获取统计信息
    stats := v5.Stats()
    fmt.Printf("统计信息: %+v\n", stats)
}
```

## 迁移指南（从 v4 到 v5）

### 接口变化

**v4:**
```go
type RuleValidator interface {
    RuleValidation() map[ValidateScene]map[string]string
}

type CustomValidator interface {
    CustomValidation(scene ValidateScene, report FuncReportError)
}
```

**v5:**
```go
type RuleProvider interface {
    GetRules(scene Scene) map[string]string
}

type BusinessValidator interface {
    ValidateBusiness(ctx *ValidationContext) error
}
```

### 迁移步骤

1. **重命名接口方法**
   - `RuleValidation()` → `GetRules(scene)`
   - `CustomValidation()` → `ValidateBusiness(ctx)`

2. **调整规则格式**
   - v4: 返回 `map[Scene]map[string]string`
   - v5: 根据 scene 参数返回 `map[string]string`

3. **错误报告方式**
   - v4: 使用 `report(namespace, tag, param)` 函数
   - v5: 使用 `ctx.AddError()` 方法

### 示例对比

**v4 代码:**
```go
func (u *User) CustomValidation(scene ValidateScene, report FuncReportError) {
    if u.Password != u.ConfirmPassword {
        report("User.ConfirmPassword", "password_mismatch", "")
    }
}
```

**v5 代码:**
```go
func (u *User) ValidateBusiness(ctx *ValidationContext) error {
    if u.Password != u.ConfirmPassword {
        ctx.AddError(NewFieldError("User.ConfirmPassword", "ConfirmPassword", "password_mismatch").
            WithMessage("密码不匹配"))
    }
    return nil
}
```