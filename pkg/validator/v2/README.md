# Validator V2 - 重构版验证器

## 📋 概述

`validator/v2` 是对原有验证器的完全重构版本，严格遵循 **SOLID 原则**和**设计模式最佳实践**，提供了更清晰的架构和更好的可扩展性。

---

## 🎯 设计原则应用

### 1. ✅ 单一职责原则（SRP）

每个组件只负责一个功能：

| 组件 | 职责 | 文件 |
|------|------|------|
| `Validator` | 协调验证流程 | `validator.go` |
| `ErrorCollector` | 收集和管理错误 | `collector.go` |
| `TypeInfoCache` | 缓存类型元数据 | `cache.go` |
| `ValidationStrategy` | 执行具体验证 | `strategy.go` |

### 2. ✅ 开放封闭原则（OCP）

通过策略模式实现扩展：

```go
// 定义验证策略接口
type ValidationStrategy interface {
    Execute(obj any, scene ValidateScene, collector ErrorCollector)
}

// 轻松添加新策略，无需修改核心代码
type customStrategy struct{}
func (s *customStrategy) Execute(obj any, scene ValidateScene, collector ErrorCollector) {
    // 自定义验证逻辑
}
```

### 3. ✅ 里氏替换原则（LSP）

所有策略实现可以互相替换：

```go
var strategy ValidationStrategy
strategy = NewRuleStrategy(v)
strategy = NewBusinessStrategy()
strategy = NewCompositeStrategy(s1, s2) // 组合策略
// 统一调用
strategy.Execute(obj, scene, collector)
```

### 4. ✅ 接口隔离原则（ISP）

细化的专用接口：

```go
// 规则提供者接口
type RuleProvider interface {
    GetRules() map[ValidateScene]map[string]string
}

// 业务验证器接口
type BusinessValidator interface {
    ValidateBusiness(scene ValidateScene) []ValidationError
}

// 模型只需实现需要的接口
```

### 5. ✅ 依赖倒置原则（DIP）

依赖抽象而非具体实现：

```go
type Validator struct {
    typeCache TypeInfoCache        // 依赖接口
    strategy  ValidationStrategy   // 依赖接口
}

// 可以注入自定义实现
validator := NewValidator(Config{
    TypeCache: myCustomCache,
    Strategy:  myCustomStrategy,
})
```

---

## 🎨 设计模式应用

| 设计模式 | 应用场景 | 优势 |
|---------|---------|------|
| **策略模式** | 验证策略 | 易于扩展新验证类型 |
| **工厂方法** | 对象创建 | 统一的创建接口 |
| **组合模式** | 策略组合 | 灵活组合多个策略 |
| **依赖注入** | 配置验证器 | 提升可测试性 |

---

## 🚀 快速开始

### 1. 基本使用

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
    Age      int    `json:"age"`
}

// 实现 RuleProvider 接口（字段规则验证）
func (u *User) GetRules() map[v2.ValidateScene]map[string]string {
    return map[v2.ValidateScene]map[string]string{
        v2.SceneCreate: {
            "username": "required,min=3,max=20",
            "email":    "required,email",
            "age":      "omitempty,gte=0,lte=150",
        },
    }
}

// 实现 BusinessValidator 接口（业务逻辑验证）
func (u *User) ValidateBusiness(scene v2.ValidateScene) []v2.ValidationError {
    var errors []v2.ValidationError
    
    if u.Username == "admin" {
        errors = append(errors, v2.NewFieldError(
            "username",
            "reserved",
            "用户名是保留字",
        ))
    }
    
    return errors
}

func main() {
    // 创建验证器
    validator := v2.NewValidator()
    
    // 创建用户
    user := &User{
        Username: "john",
        Email:    "john@example.com",
        Age:      25,
    }
    
    // 验证
    errors := validator.Validate(user, v2.SceneCreate)
    
    // 处理结果
    if len(errors) > 0 {
        fmt.Println("验证失败:")
        for _, err := range errors {
            fmt.Printf("- %s: %s\n", err.Field(), err.Message())
        }
    } else {
        fmt.Println("验证通过!")
    }
}
```

---

## 📚 核心接口

### RuleProvider - 字段规则验证

```go
type RuleProvider interface {
    GetRules() map[ValidateScene]map[string]string
}

// 使用示例
func (u *User) GetRules() map[v2.ValidateScene]map[string]string {
    return map[v2.ValidateScene]map[string]string{
        v2.SceneCreate: {
            "username": "required,min=3,max=20,alphanum",
            "email":    "required,email",
        },
        v2.SceneUpdate: {
            "username": "omitempty,min=3,max=20,alphanum",
            "email":    "omitempty,email",
        },
    }
}
```

### BusinessValidator - 业务逻辑验证

```go
type BusinessValidator interface {
    ValidateBusiness(scene ValidateScene) []ValidationError
}

// 使用示例
func (u *User) ValidateBusiness(scene v2.ValidateScene) []v2.ValidationError {
    var errors []v2.ValidationError
    
    // 复杂的业务逻辑验证
    if scene == v2.SceneCreate && u.Age < 18 {
        errors = append(errors, v2.NewFieldError(
            "age",
            "underage",
            "用户必须年满18岁",
        ))
    }
    
    return errors
}
```

---

## 🔧 高级功能

### 1. 自定义验证策略

```go
// 定义自定义策略
type DatabaseValidationStrategy struct {
    db *sql.DB
}

func (s *DatabaseValidationStrategy) Execute(
    obj any, 
    scene v2.ValidateScene, 
    collector v2.ErrorCollector,
) {
    user, ok := obj.(*User)
    if !ok {
        return
    }
    
    // 检查用户名唯一性
    exists := s.checkUsernameExists(user.Username)
    if exists {
        collector.Add(v2.NewFieldError(
            "username",
            "unique",
            "用户名已存在",
        ))
    }
}

// 使用自定义策略
validator := v2.NewValidator(v2.Config{
    Strategy: v2.NewCompositeStrategy(
        v2.NewRuleStrategy(nil),
        v2.NewBusinessStrategy(),
        &DatabaseValidationStrategy{db: db},
    ),
})
```

### 2. 场景组合

```go
// 定义组合场景
const (
    SceneCreateOrUpdate = v2.SceneCreate | v2.SceneUpdate
    SceneAll            = v2.SceneCreate | v2.SceneUpdate | v2.SceneDelete
)

// 使用组合场景
errors := validator.Validate(user, SceneCreateOrUpdate)
```

### 3. 依赖注入

```go
// 注入自定义缓存
validator := v2.NewValidator(v2.Config{
    TypeCache: myCustomCache,
})

// 注入自定义策略
validator := v2.NewValidator(v2.Config{
    Strategy: myCustomStrategy,
})
```

---

## 🧪 测试支持

### Mock ErrorCollector

```go
type MockCollector struct {
    errors []v2.ValidationError
}

func (m *MockCollector) Add(err v2.ValidationError) {
    m.errors = append(m.errors, err)
}

// 其他方法实现...

// 在测试中使用
func TestMyValidator(t *testing.T) {
    collector := &MockCollector{}
    strategy := NewMyStrategy()
    strategy.Execute(obj, scene, collector)
    
    assert.Equal(t, 1, len(collector.errors))
}
```

---

## 📊 性能优化

### 类型缓存

```go
// 第一次验证：缓存类型信息
validator.Validate(user1, v2.SceneCreate)

// 后续验证：使用缓存，性能提升
validator.Validate(user2, v2.SceneCreate)
validator.Validate(user3, v2.SceneCreate)
```

### 并发安全

```go
// ErrorCollector 支持并发安全
var wg sync.WaitGroup
for _, user := range users {
    wg.Add(1)
    go func(u *User) {
        defer wg.Done()
        errors := validator.Validate(u, v2.SceneCreate)
        // 处理错误
    }(user)
}
wg.Wait()
```

---

## 🔄 与原版本对比

| 特性 | 原版本 | V2 版本 |
|------|--------|---------|
| **接口设计** | 回调函数 | 直接返回错误列表 |
| **依赖管理** | 依赖具体实现 | 依赖抽象接口 |
| **可扩展性** | 需修改核心代码 | 通过策略模式扩展 |
| **可测试性** | 难以 Mock | 易于 Mock 和测试 |
| **代码组织** | 单文件多职责 | 多文件单一职责 |
| **并发安全** | 部分支持 | 完全支持 |

---

## 📖 文件结构

```
validator/v2/
├── doc.go           # 包文档
├── types.go         # 基本类型定义
├── interfaces.go    # 核心接口
├── validator.go     # 验证器实现
├── strategy.go      # 验证策略
├── collector.go     # 错误收集器
├── cache.go         # 类型缓存
├── validator_test.go # 单元测试
└── README.md        # 本文档
```

---

## 💡 最佳实践

### 1. 接口实现建议

```go
// ✅ 好的实践：分离验证逻辑
type User struct {
    Username string
    Email    string
}

// 简单规则 -> RuleProvider
func (u *User) GetRules() map[v2.ValidateScene]map[string]string {
    return map[v2.ValidateScene]map[string]string{
        v2.SceneCreate: {"username": "required,min=3"},
    }
}

// 复杂逻辑 -> BusinessValidator
func (u *User) ValidateBusiness(scene v2.ValidateScene) []v2.ValidationError {
    // 复杂的业务验证
    return nil
}
```

### 2. 错误处理

```go
errors := validator.Validate(user, v2.SceneCreate)
if len(errors) > 0 {
    // 按字段分组
    errorMap := make(map[string][]string)
    for _, err := range errors {
        errorMap[err.Field()] = append(
            errorMap[err.Field()],
            err.Message(),
        )
    }
    
    // 返回给客户端
    return errorMap
}
```

### 3. 场景化验证

```go
// 定义清晰的场景常量
const (
    SceneCreate v2.ValidateScene = 1 << 0
    SceneUpdate v2.ValidateScene = 1 << 1
    SceneDelete v2.ValidateScene = 1 << 2
)

// 场景组合
const SceneCreateOrUpdate = SceneCreate | SceneUpdate
```

---

## 🎓 总结

V2 版本的验证器通过应用 **SOLID 原则**和**设计模式**，实现了：

- ✅ **高内聚低耦合**：每个组件职责明确
- ✅ **易于扩展**：通过策略模式无需修改核心代码
- ✅ **易于测试**：依赖接口，支持 Mock
- ✅ **易于维护**：清晰的代码结构
- ✅ **高性能**：类型缓存优化
- ✅ **并发安全**：支持多协程并发验证

这是一个**生产级别**的验证器实现，适合大型项目使用！
package v2_test

import (
	"fmt"
	"testing"

	"katydid-common-account/pkg/validator/v2"
)

// ============================================================================
// 示例：基本使用
// ============================================================================

// User 用户模型
type User struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
	Password string `json:"password"`
}

// GetRules 实现 RuleProvider 接口
func (u *User) GetRules() map[v2.ValidateScene]map[string]string {
	return map[v2.ValidateScene]map[string]string{
		v2.SceneCreate: {
			"username": "required,min=3,max=20,alphanum",
			"email":    "required,email",
			"age":      "omitempty,gte=0,lte=150",
			"password": "required,min=6",
		},
		v2.SceneUpdate: {
			"username": "omitempty,min=3,max=20,alphanum",
			"email":    "omitempty,email",
			"age":      "omitempty,gte=0,lte=150",
		},
	}
}

// ValidateBusiness 实现 BusinessValidator 接口
func (u *User) ValidateBusiness(scene v2.ValidateScene) []v2.ValidationError {
	var errors []v2.ValidationError
	
	// 场景化业务验证
	if scene == v2.SceneCreate {
		// 检查用户名是否为保留字
		if u.Username == "admin" || u.Username == "root" || u.Username == "system" {
			errors = append(errors, v2.NewFieldError(
				"username",
				"reserved",
				"用户名是保留字，不能使用",
			))
		}
		
		// 检查密码强度（示例）
		if len(u.Password) > 0 && len(u.Password) < 6 {
			errors = append(errors, v2.NewFieldError(
				"password",
				"weak",
				"密码强度不足",
			))
		}
	}
	
	return errors
}

// TestBasicValidation 基本验证测试
func TestBasicValidation(t *testing.T) {
	// 创建验证器
	validator := v2.NewValidator()
	
	// 测试1: 验证成功
	user1 := &User{
		Username: "john",
		Email:    "john@example.com",
		Age:      25,
		Password: "secret123",
	}
	
	errors := validator.Validate(user1, v2.SceneCreate)
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d errors", len(errors))
		for _, err := range errors {
			t.Logf("Error: %s", err.Message())
		}
	}
	
	// 测试2: 验证失败 - 缺少必填字段
	user2 := &User{
		Email: "invalid-email", // 无效的邮箱
	}
	
	errors = validator.Validate(user2, v2.SceneCreate)
	if len(errors) == 0 {
		t.Error("Expected validation errors, got none")
	}
	
	// 测试3: 业务验证失败 - 保留字用户名
	user3 := &User{
		Username: "admin", // 保留字
		Email:    "admin@example.com",
		Password: "admin123",
	}
	
	errors = validator.Validate(user3, v2.SceneCreate)
	hasReservedError := false
	for _, err := range errors {
		if err.Tag() == "reserved" {
			hasReservedError = true
			break
		}
	}
	
	if !hasReservedError {
		t.Error("Expected reserved username error")
	}
}

// ExampleValidator_Validate 使用示例
func ExampleValidator_Validate() {
	// 创建验证器
	validator := v2.NewValidator()
	
	// 创建用户对象
	user := &User{
		Username: "john",
		Email:    "john@example.com",
		Age:      25,
		Password: "secret123",
	}
	
	// 验证创建场景
	errors := validator.Validate(user, v2.SceneCreate)
	
	// 处理验证结果
	if len(errors) > 0 {
		fmt.Println("验证失败:")
		for _, err := range errors {
			fmt.Printf("- %s: %s\n", err.Field(), err.Message())
		}
	} else {
		fmt.Println("验证通过")
	}
	
	// Output:
	// 验证通过
}

// ============================================================================
// 示例：自定义策略
// ============================================================================

// customStrategy 自定义验证策略示例
type customStrategy struct{}

func (s *customStrategy) Execute(obj any, scene v2.ValidateScene, collector v2.ErrorCollector) {
	user, ok := obj.(*User)
	if !ok {
		return
	}
	
	// 自定义验证逻辑：用户名和邮箱前缀不能相同
	if user.Username != "" && user.Email != "" {
		emailPrefix := user.Email[:len(user.Username)]
		if emailPrefix == user.Username {
			collector.Add(v2.NewFieldError(
				"email",
				"conflict",
				"邮箱前缀不能与用户名相同",
			))
		}
	}
}

func TestCustomStrategy(t *testing.T) {
	// 创建带自定义策略的验证器
	validator := v2.NewValidator(v2.Config{
		Strategy: v2.NewCompositeStrategy(
			v2.NewRuleStrategy(nil), // 会在内部创建
			v2.NewBusinessStrategy(),
			&customStrategy{}, // 自定义策略
		),
	})
	
	user := &User{
		Username: "john",
		Email:    "john@example.com", // 邮箱前缀与用户名相同
		Password: "secret123",
	}
	
	errors := validator.Validate(user, v2.SceneCreate)
	
	hasConflictError := false
	for _, err := range errors {
		if err.Tag() == "conflict" {
			hasConflictError = true
			t.Logf("Found conflict error: %s", err.Message())
		}
	}
	
	if !hasConflictError {
		t.Error("Expected conflict error")
	}
}

// ============================================================================
// 示例：场景组合
// ============================================================================

func TestSceneCombination(t *testing.T) {
	validator := v2.NewValidator()
	
	// 定义组合场景
	const SceneCreateOrUpdate = v2.SceneCreate | v2.SceneUpdate
	
	user := &User{
		Username: "john",
		Email:    "john@example.com",
	}
	
	// 使用组合场景验证
	errors := validator.Validate(user, SceneCreateOrUpdate)
	
	t.Logf("Validation with combined scene returned %d errors", len(errors))
}

// ============================================================================
// 性能测试
// ============================================================================

func BenchmarkValidation(b *testing.B) {
	validator := v2.NewValidator()
	
	user := &User{
		Username: "john",
		Email:    "john@example.com",
		Age:      25,
		Password: "secret123",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(user, v2.SceneCreate)
	}
}

func BenchmarkValidationWithCache(b *testing.B) {
	validator := v2.NewValidator()
	
	// 预热缓存
	user := &User{
		Username: "john",
		Email:    "john@example.com",
		Age:      25,
		Password: "secret123",
	}
	_ = validator.Validate(user, v2.SceneCreate)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(user, v2.SceneCreate)
	}
}

