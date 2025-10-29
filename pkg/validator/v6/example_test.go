package v6_test

import (
	"fmt"
	v6 "katydid-common-account/pkg/validator/v6"
	"katydid-common-account/pkg/validator/v6/core"
)

// 定义场景
const (
	SceneCreate core.Scene = 1 << iota // 1
	SceneUpdate                         // 2
	SceneDelete                         // 4
)

// User 用户模型
type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Age      int    `json:"age"`
}

// ValidateRules 实现 IRuleValidator 接口
func (u *User) ValidateRules(scene core.Scene) map[string]string {
	switch scene {
	case SceneCreate:
		return map[string]string{
			"username": "required,min=3,max=20",
			"email":    "required,email",
			"password": "required,min=6",
			"age":      "required,gte=18,lte=120",
		}
	case SceneUpdate:
		return map[string]string{
			"username": "omitempty,min=3,max=20",
			"email":    "omitempty,email",
			"age":      "omitempty,gte=18,lte=120",
		}
	default:
		return nil
	}
}

// ValidateBusiness 实现 IBusinessValidator 接口
func (u *User) ValidateBusiness(scene core.Scene, collector core.IErrorCollector) {
	// 跨字段验证
	if scene == SceneCreate && u.Password == "123456" {
		collector.Collect(v6.NewFieldError("User.Password", "password", "weak",
			v6.WithMessage("密码过于简单")))
	}

	// 模拟数据库检查
	if scene == SceneCreate && u.Username == "admin" {
		collector.Collect(v6.NewFieldError("User.Username", "username", "duplicate",
			v6.WithMessage("用户名已存在")))
	}
}

// BeforeValidation 实现 LifecycleHooks 接口
func (u *User) BeforeValidation(ctx core.IContext) error {
	// 数据预处理
	// u.Username = strings.TrimSpace(u.Username)
	fmt.Printf("验证前处理: scene=%v\n", ctx.Scene())
	return nil
}

// AfterValidation 实现 LifecycleHooks 接口
func (u *User) AfterValidation(ctx core.IContext) error {
	fmt.Printf("验证后处理: scene=%v\n", ctx.Scene())
	return nil
}

// ============================================================================
// 示例代码
// ============================================================================

// Example_basic 基本使用示例
func Example_basic() {
	user := &User{
		Username: "jo",  // 太短
		Email:    "bad", // 格式错误
		Password: "123", // 太短
		Age:      15,    // 年龄不够
	}

	// 使用默认验证器
	if err := v6.Validate(user, SceneCreate); err != nil {
		fmt.Printf("验证失败:\n")
		for _, msg := range err.Errors() {
			fmt.Printf("  - %s\n", msg)
		}
	}

	// Output:
	// 验证失败:
	//   - Field 'username' failed validation on tag 'min' with param '3'
	//   - Field 'email' failed validation on tag 'email'
	//   - Field 'password' failed validation on tag 'min' with param '6'
	//   - Field 'age' failed validation on tag 'gte' with param '18'
}

// Example_builder 使用构建器自定义验证器
func Example_builder() {
	// 创建自定义验证器
	validator := v6.NewBuilder().
		WithLRUCache(1000).
		WithRuleStrategy(10).
		WithBusinessStrategy(20).
		WithMaxErrors(50).
		Build()

	user := &User{
		Username: "john",
		Email:    "john@example.com",
		Password: "password123",
		Age:      25,
	}

	if err := validator.Validate(user, SceneCreate); err != nil {
		fmt.Printf("验证失败: %v\n", err)
	} else {
		fmt.Println("验证通过")
	}

	// Output:
	// 验证通过
}

// Example_interceptor 使用拦截器
func Example_interceptor() {
	// 创建带拦截器的验证器
	validator := v6.NewBuilder().
		WithRuleStrategy(10).
		WithInterceptor(v6.InterceptorFunc(func(ctx core.IContext, target any, next func() error) error {
			fmt.Printf("拦截器: 验证开始\n")
			err := next()
			fmt.Printf("拦截器: 验证结束\n")
			return err
		})).
		Build()

	user := &User{
		Username: "john",
		Email:    "john@example.com",
		Password: "password123",
		Age:      25,
	}

	validator.Validate(user, SceneCreate)

	// Output:
	// 拦截器: 验证开始
	// 验证前处理: scene=1
	// 验证后处理: scene=1
	// 拦截器: 验证结束
}

// Example_listener 使用监听器
func Example_listener() {
	// 自定义监听器
	type MyListener struct{}

	func (l *MyListener) OnValidationStart(ctx core.IContext, target any) {
		fmt.Printf("监听器: 验证开始\n")
	}

	func (l *MyListener) OnValidationEnd(ctx core.IContext, target any, err error) {
		fmt.Printf("监听器: 验证结束\n")
	}

	func (l *MyListener) OnError(ctx core.IContext, fieldErr core.IFieldError) {
		fmt.Printf("监听器: 发现错误 - %s\n", fieldErr.Field())
	}

	// 创建带监听器的验证器
	validator := v6.NewBuilder().
		WithRuleStrategy(10).
		WithListener(&MyListener{}).
		Build()

	user := &User{
		Username: "jo",  // 太短
		Email:    "bad", // 格式错误
		Password: "123456",
		Age:      25,
	}

	validator.Validate(user, SceneCreate)

	// Output:
	// 监听器: 验证开始
	// 验证前处理: scene=1
	// 验证后处理: scene=1
	// 监听器: 发现错误 - username
	// 监听器: 发现错误 - email
	// 监听器: 验证结束
}

// Example_facade 使用门面
func Example_facade() {
	user := &User{
		Username: "john",
		Email:    "john@example.com",
		Password: "password123",
		Age:      25,
	}

	// 直接使用全局门面
	if err := v6.Validate(user, SceneCreate); err != nil {
		fmt.Printf("验证失败: %v\n", err)
	} else {
		fmt.Println("验证通过")
	}

	// Output:
	// 验证通过
}
