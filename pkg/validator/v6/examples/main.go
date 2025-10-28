package main

import (
	"fmt"
	"log"

	"katydid-common-account/pkg/validator/v6"
	"katydid-common-account/pkg/validator/v6/core"
	"katydid-common-account/pkg/validator/v6/plugin"
)

// 定义验证场景
const (
	SceneCreate core.Scene = 1 << iota // 1
	SceneUpdate                         // 2
	SceneDelete                         // 4
)

// User 用户模型
type User struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
	Password string `json:"password"`
}

// GetRules 实现 RuleProvider 接口
func (u *User) GetRules() map[core.Scene]map[string]string {
	return map[core.Scene]map[string]string{
		SceneCreate: {
			"name":     "required,min=2,max=50",
			"email":    "required,email",
			"age":      "required,min=18,max=120",
			"password": "required,min=6,max=32",
		},
		SceneUpdate: {
			"name":     "omitempty,min=2,max=50",
			"email":    "omitempty,email",
			"age":      "omitempty,min=18,max=120",
			"password": "omitempty,min=6,max=32",
		},
		SceneDelete: {
			"id": "required,min=1",
		},
	}
}

// ValidateBusiness 实现 BusinessValidator 接口
func (u *User) ValidateBusiness(scene core.Scene, ctx core.ValidationContext) error {
	// 示例：业务逻辑验证
	switch scene {
	case SceneCreate:
		// 创建时的特殊验证
		if u.Age < 18 {
			ctx.ErrorCollector().Add(
				core.NewFieldError("age", "age_limit").
					WithMessage("创建用户时年龄必须大于等于18岁"),
			)
		}

		// 可以添加更多业务验证，如：
		// - 检查邮箱是否已存在（需要查询数据库）
		// - 检查用户名是否被占用
		// - 验证邀请码是否有效

	case SceneUpdate:
		// 更新时的特殊验证
		if u.ID <= 0 {
			ctx.ErrorCollector().Add(
				core.NewFieldError("id", "required").
					WithMessage("更新时必须提供用户ID"),
			)
		}
	}

	return nil
}

// BeforeValidation 实现 LifecycleHook 接口
func (u *User) BeforeValidation(ctx core.ValidationContext) error {
	fmt.Println("🔍 验证前处理...")
	// 可以做一些预处理，如：
	// - 清理数据（trim 空格）
	// - 数据转换
	// - 日志记录
	return nil
}

// AfterValidation 实现 LifecycleHook 接口
func (u *User) AfterValidation(ctx core.ValidationContext) error {
	if ctx.ErrorCollector().HasErrors() {
		fmt.Println("❌ 验证失败，进行清理...")
	} else {
		fmt.Println("✅ 验证成功，进行后续处理...")
	}
	return nil
}

// Product 产品模型（演示嵌套验证）
type Product struct {
	Name   string `json:"name"`
	Price  float64 `json:"price"`
	Owner  *User  `json:"owner"`  // 嵌套对象
}

func (p *Product) GetRules() map[core.Scene]map[string]string {
	return map[core.Scene]map[string]string{
		SceneCreate: {
			"name":  "required,min=2,max=100",
			"price": "required,min=0",
		},
	}
}

// 示例1：基本用法
func example1BasicUsage() {
	fmt.Println("\n=== 示例1：基本用法 ===")

	// 创建验证器
	validator := v6.NewValidator().BuildDefault()

	// 创建用户
	user := &User{
		Name:     "张三",
		Email:    "zhangsan@example.com",
		Age:      25,
		Password: "secret123",
	}

	// 验证
	if err := validator.Validate(user, SceneCreate); err != nil {
		log.Printf("验证失败: %v\n", err)
		return
	}

	fmt.Println("✅ 验证通过")
}

// 示例2：验证失败
func example2ValidationFailure() {
	fmt.Println("\n=== 示例2：验证失败 ===")

	validator := v6.NewValidator().BuildDefault()

	// 创建一个不合法的用户
	user := &User{
		Name:     "李", // 太短
		Email:    "invalid-email", // 格式错误
		Age:      15, // 年龄不够
		Password: "123", // 密码太短
	}

	if err := validator.Validate(user, SceneCreate); err != nil {
		if validationErr, ok := err.(*core.ValidationError); ok {
			fmt.Printf("❌ 验证失败，共 %d 个错误:\n", validationErr.Count())
			for i, fieldErr := range validationErr.Errors() {
				fmt.Printf("  %d. %s\n", i+1, fieldErr.Error())
			}
		}
		return
	}
}

// 示例3：使用插件
func example3WithPlugin() {
	fmt.Println("\n=== 示例3：使用插件 ===")

	// 创建带日志插件的验证器
	validator := v6.NewValidator().
		WithPlugins(plugin.NewLoggingPlugin()).
		BuildDefault()

	user := &User{
		Name:     "王五",
		Email:    "wangwu@example.com",
		Age:      30,
		Password: "password123",
	}

	if err := validator.Validate(user, SceneCreate); err != nil {
		log.Printf("验证失败: %v\n", err)
		return
	}
}

// 示例4：高级用法 - 只验证指定字段
func example4ValidateSpecificFields() {
	fmt.Println("\n=== 示例4：只验证指定字段 ===")

	validator := v6.NewValidator().BuildDefault()

	user := &User{
		Name:  "赵六",
		Email: "zhaoliu@example.com",
		// 故意不设置 Age 和 Password
	}

	// 只验证 name 和 email 字段
	req := core.NewValidationRequest(user, SceneCreate).
		WithFields("name", "email")

	result, err := validator.ValidateWithRequest(req)
	if err != nil {
		log.Printf("请求错误: %v\n", err)
		return
	}

	if result.HasErrors() {
		fmt.Printf("❌ 验证失败: %v\n", result.ToError())
		return
	}

	fmt.Println("✅ 指定字段验证通过")
}

// 示例5：排除字段验证
func example5ExcludeFields() {
	fmt.Println("\n=== 示例5：排除字段验证 ===")

	validator := v6.NewValidator().BuildDefault()

	user := &User{
		Name:  "钱七",
		Email: "qianqi@example.com",
		Age:   28,
		// 不设置 Password
	}

	// 排除 password 字段验证
	req := core.NewValidationRequest(user, SceneCreate).
		WithExcludeFields("password")

	result, err := validator.ValidateWithRequest(req)
	if err != nil {
		log.Printf("请求错误: %v\n", err)
		return
	}

	if result.HasErrors() {
		fmt.Printf("❌ 验证失败: %v\n", result.ToError())
		return
	}

	fmt.Println("✅ 排除字段后验证通过")
}

// 示例6：场景组合
func example6SceneCombination() {
	fmt.Println("\n=== 示例6：场景组合 ===")

	validator := v6.NewValidator().BuildDefault()

	// 定义组合场景
	SceneCreateOrUpdate := SceneCreate | SceneUpdate

	user := &User{
		Name:  "孙八",
		Email: "sunba@example.com",
		Age:   35,
	}

	// 使用组合场景验证
	if err := validator.Validate(user, SceneCreateOrUpdate); err != nil {
		log.Printf("验证失败: %v\n", err)
		return
	}

	fmt.Println("✅ 场景组合验证通过")
}

// 示例7：自定义监听器
func example7CustomListener() {
	fmt.Println("\n=== 示例7：自定义监听器 ===")

	// 定义自定义监听器
	type LogListener struct{}

	func (l *LogListener) OnEvent(event core.ValidationEvent) {
		switch event.Type() {
		case core.EventTypeValidationStart:
			fmt.Println("📢 监听器: 验证开始")
		case core.EventTypeValidationEnd:
			ctx := event.Context()
			if ctx.ErrorCollector().HasErrors() {
				fmt.Printf("📢 监听器: 验证结束，发现 %d 个错误\n", ctx.ErrorCollector().Count())
			} else {
				fmt.Println("📢 监听器: 验证结束，无错误")
			}
		}
	}

	// 创建带监听器的验证器
	validator := v6.NewValidator().
		WithListeners(&LogListener{}).
		BuildDefault()

	user := &User{
		Name:     "周九",
		Email:    "zhoujiu@example.com",
		Age:      40,
		Password: "mypassword",
	}

	if err := validator.Validate(user, SceneCreate); err != nil {
		log.Printf("验证失败: %v\n", err)
		return
	}
}

// 示例8：使用全局验证器
func example8GlobalValidator() {
	fmt.Println("\n=== 示例8：使用全局验证器 ===")

	user := &User{
		Name:     "吴十",
		Email:    "wushi@example.com",
		Age:      45,
		Password: "globalpass",
	}

	// 使用全局验证器（便捷方法）
	if err := v6.Validate(user, SceneCreate); err != nil {
		log.Printf("验证失败: %v\n", err)
		return
	}

	fmt.Println("✅ 全局验证器验证通过")
}

func main() {
	fmt.Println("🚀 v6 验证器示例程序")
	fmt.Println("======================")

	// 运行所有示例
	example1BasicUsage()
	example2ValidationFailure()
	example3WithPlugin()
	example4ValidateSpecificFields()
	example5ExcludeFields()
	example6SceneCombination()
	example7CustomListener()
	example8GlobalValidator()

	fmt.Println("\n======================")
	fmt.Println("✨ 所有示例运行完成")
}

