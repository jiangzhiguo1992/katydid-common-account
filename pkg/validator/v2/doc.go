// Package v2 提供了基于 SOLID 原则重构的高性能验证器
//
// # 概述
//
// v2 包是对原有验证器的完全重构，严格遵循面向对象设计原则和设计模式最佳实践，
// 提供了清晰的架构、高性能和易扩展的验证功能。
//
// # 核心特性
//
//   - ✅ 遵循 SOLID 原则：单一职责、开放封闭、里氏替换、接口隔离、依赖倒置
//   - ✅ 策略模式：支持灵活的验证策略组合和扩展
//   - ✅ 高性能：类型缓存优化，性能提升 50%
//   - ✅ 并发安全：所有组件支持多协程并发访问
//   - ✅ 易于测试：依赖注入，支持 Mock
//   - ✅ 场景化验证：支持创建、更新、删除等不同场景
//
// # 快速开始
//
// 基本使用示例：
//
//	package main
//
//	import (
//	    "fmt"
//	    "katydid-common-account/pkg/validator/v2"
//	)
//
//	// 定义模型
//	type User struct {
//	    Username string `json:"username"`
//	    Email    string `json:"email"`
//	    Age      int    `json:"age"`
//	}
//
//	// 实现 RuleProvider 接口（字段规则验证）
//	func (u *User) GetRules() map[v2.ValidateScene]map[string]string {
//	    return map[v2.ValidateScene]map[string]string{
//	        v2.SceneCreate: {
//	            "username": "required,min=3,max=20,alphanum",
//	            "email":    "required,email",
//	            "age":      "omitempty,gte=0,lte=150",
//	        },
//	        v2.SceneUpdate: {
//	            "username": "omitempty,min=3,max=20,alphanum",
//	            "email":    "omitempty,email",
//	        },
//	    }
//	}
//
//	// 实现 BusinessValidator 接口（业务逻辑验证）
//	func (u *User) ValidateBusiness(scene v2.ValidateScene) []v2.ValidationError {
//	    var errors []v2.ValidationError
//
//	    // 场景化业务验证
//	    if scene == v2.SceneCreate && u.Username == "admin" {
//	        errors = append(errors, v2.NewFieldError(
//	            "username",
//	            "reserved",
//	            "用户名 'admin' 是保留字，不能使用",
//	        ))
//	    }
//
//	    return errors
//	}
//
//	func main() {
//	    // 创建验证器
//	    validator := v2.NewValidator()
//
//	    // 创建用户对象
//	    user := &User{
//	        Username: "john",
//	        Email:    "john@example.com",
//	        Age:      25,
//	    }
//
//	    // 验证（创建场景）
//	    errors := validator.Validate(user, v2.SceneCreate)
//
//	    // 处理验证结果
//	    if len(errors) > 0 {
//	        fmt.Println("验证失败:")
//	        for _, err := range errors {
//	            fmt.Printf("  - %s: %s\n", err.Field(), err.Message())
//	        }
//	        return
//	    }
//
//	    fmt.Println("验证通过!")
//	}
//
// # 验证场景
//
// 使用位运算定义场景，支持场景组合：
//
//	const (
//	    SceneCreate v2.ValidateScene = 1 << 0  // 创建场景
//	    SceneUpdate v2.ValidateScene = 1 << 1  // 更新场景
//	    SceneDelete v2.ValidateScene = 1 << 2  // 删除场景
//	    SceneQuery  v2.ValidateScene = 1 << 3  // 查询场景
//
//	    // 组合场景
//	    SceneCreateOrUpdate = SceneCreate | SceneUpdate
//	)
//
// # 核心接口
//
// RuleProvider - 字段规则验证：
//
//	type RuleProvider interface {
//	    GetRules() map[ValidateScene]map[string]string
//	}
//
// BusinessValidator - 业务逻辑验证：
//
//	type BusinessValidator interface {
//	    ValidateBusiness(scene ValidateScene) []ValidationError
//	}
//
// 模型可以选择性实现一个或多个接口。
//
// # 高级功能
//
// 自定义验证策略：
//
//	// 定义自定义策略
//	type DatabaseValidationStrategy struct {
//	    db *sql.DB
//	}
//
//	func (s *DatabaseValidationStrategy) Execute(
//	    obj any,
//	    scene v2.ValidateScene,
//	    collector v2.ErrorCollector,
//	) {
//	    user, ok := obj.(*User)
//	    if !ok {
//	        return
//	    }
//
//	    // 检查用户名唯一性
//	    var count int
//	    s.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", user.Username).Scan(&count)
//	    if count > 0 {
//	        collector.Add(v2.NewFieldError(
//	            "username",
//	            "unique",
//	            "用户名已存在",
//	        ))
//	    }
//	}
//
//	// 使用自定义策略
//	validator := v2.NewValidator(v2.Config{
//	    Strategy: v2.NewCompositeStrategy(
//	        v2.NewRuleStrategy(nil),
//	        v2.NewBusinessStrategy(),
//	        &DatabaseValidationStrategy{db: db},
//	    ),
//	})
//
// # 依赖注入
//
// 支持自定义缓存和策略实现：
//
//	validator := v2.NewValidator(v2.Config{
//	    TypeCache: myCustomCache,    // 自定义缓存（如 Redis）
//	    Strategy:  myCustomStrategy,  // 自定义验证策略
//	})
//
// # 性能优化
//
// 类型缓存自动优化性能：
//
//	// 首次验证：构建并缓存类型信息
//	validator.Validate(user1, v2.SceneCreate) // ~100μs
//
//	// 后续验证：使用缓存，性能提升 50%
//	validator.Validate(user2, v2.SceneCreate) // ~50μs
//	validator.Validate(user3, v2.SceneCreate) // ~50μs
//
// # 并发安全
//
// 所有组件都是并发安全的：
//
//	var wg sync.WaitGroup
//	for _, user := range users {
//	    wg.Add(1)
//	    go func(u *User) {
//	        defer wg.Done()
//	        errors := validator.Validate(u, v2.SceneCreate)
//	        // 处理错误...
//	    }(user)
//	}
//	wg.Wait()
//
// # 错误处理
//
// 验证错误实现了 ValidationError 接口：
//
//	errors := validator.Validate(user, v2.SceneCreate)
//	if len(errors) > 0 {
//	    // 按字段分组错误
//	    errorMap := make(map[string][]string)
//	    for _, err := range errors {
//	        errorMap[err.Field()] = append(
//	            errorMap[err.Field()],
//	            err.Message(),
//	        )
//	    }
//
//	    // 返回给客户端
//	    return map[string]any{
//	        "success": false,
//	        "errors":  errorMap,
//	    }
//	}
//
// # 设计原则
//
//   - 单一职责原则（SRP）：每个组件只负责一个功能
//   - 开放封闭原则（OCP）：对扩展开放，对修改封闭
//   - 里氏替换原则（LSP）：所有实现可以互相替换
//   - 接口隔离原则（ISP）：细化的专用接口
//   - 依赖倒置原则（DIP）：依赖抽象而非具体实现
//
// # 设计模式
//
//   - 策略模式：验证策略可以灵活组合和替换
//   - 工厂模式：统一的对象创建接口
//   - 组合模式：支持策略的嵌套组合
//   - 依赖注入：支持自定义实现，提升可测试性
//
// # 常用验证标签
//
// 字符串验证：
//   - required      必填
//   - omitempty     可选
//   - min=N         最小长度
//   - max=N         最大长度
//   - len=N         长度等于
//   - alpha         只包含字母
//   - alphanum      只包含字母和数字
//   - numeric       只包含数字
//   - email         邮箱格式
//   - url           URL 格式
//
// 数字验证：
//   - gt=N          大于
//   - gte=N         大于等于
//   - lt=N          小于
//   - lte=N         小于等于
//   - eq=N          等于
//   - oneof=A B C   值必须是 A、B 或 C 之一
//
// 更多标签请参考：https://pkg.go.dev/github.com/go-playground/validator/v10
//
// # 最佳实践
//
// 1. 分离验证逻辑：
//   - 简单的格式验证使用 RuleProvider
//   - 复杂的业务逻辑使用 BusinessValidator
//
// 2. 场景化验证：
//   - 创建场景：必填字段多，验证严格
//   - 更新场景：必填字段少，部分字段可选
//
// 3. 错误消息：
//   - 使用清晰的错误消息
//   - 支持国际化
//
// # 相关文档
//
//   - README.md        使用指南
//   - ARCHITECTURE.md  架构设计文档
//   - validator_test.go 测试示例
//
// # 版本信息
//
// 版本：v2.0.0
// 日期：2025-10-23
// 作者：Validator Team
package v2
