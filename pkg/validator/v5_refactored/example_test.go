package v5_refactored

import (
	"fmt"
)

// User 用户模型
type User struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Age      int    `json:"age"`
}

// GetRules 实现 RuleProvider 接口
func (u *User) GetRules(scene Scene) map[string]string {
	switch scene {
	case SceneCreate:
		return map[string]string{
			"username": "required,min=3,max=20",
			"email":    "required,email",
			"password": "required,min=6",
			"age":      "required,min=18",
		}
	case SceneUpdate:
		return map[string]string{
			"username": "omitempty,min=3,max=20",
			"email":    "omitempty,email",
		}
	default:
		return nil
	}
}

// ValidateBusiness 实现 BusinessValidator 接口
func (u *User) ValidateBusiness(scene Scene, ctx *ValidationContext, collector ErrorCollector) error {
	// 示例：检查用户名是否为保留字
	if u.Username == "admin" || u.Username == "root" {
		collector.Add(NewFieldError("username", "reserved").
			WithMessage("用户名已被保留，请使用其他用户名"))
	}

	// 示例：跨字段验证
	if u.Age < 18 && len(u.Password) < 8 {
		collector.Add(NewFieldError("password", "min").
			WithMessage("未成年用户密码长度至少 8 位"))
	}

	return nil
}

// BeforeValidation 实现 LifecycleHooks 接口
func (u *User) BeforeValidation(ctx *ValidationContext) error {
	// 数据预处理
	fmt.Println("=== 验证前处理 ===")
	fmt.Printf("用户名: %s\n", u.Username)
	return nil
}

// AfterValidation 实现 LifecycleHooks 接口
func (u *User) AfterValidation(ctx *ValidationContext) error {
	fmt.Println("=== 验证后处理 ===")
	return nil
}

// ValidationLogger 验证日志监听器
type ValidationLogger struct{}

func (l *ValidationLogger) OnEvent(event Event) {
	switch event.Type() {
	case EventValidationStart:
		fmt.Println("📝 开始验证...")
	case EventValidationEnd:
		fmt.Println("✅ 验证完成")
	case EventHookBefore:
		fmt.Println("🔄 执行前置钩子")
	case EventHookAfter:
		fmt.Println("🔄 执行后置钩子")
	}
}

func (l *ValidationLogger) EventTypes() []EventType {
	return nil // 监听所有事件
}

func main() {
	fmt.Println("========================================")
	fmt.Println("  v5_refactored 验证器框架示例")
	fmt.Println("========================================\n")

	// 示例 1: 使用默认验证器
	fmt.Println("【示例 1】基础验证 - 使用默认验证器")
	fmt.Println("----------------------------------------")
	user1 := &User{
		Username: "john",
		Email:    "john@example.com",
		Password: "password123",
		Age:      25,
	}

	if err := Validate(user1, SceneCreate); err != nil {
		fmt.Printf("❌ 验证失败: %v\n", err)
	} else {
		fmt.Println("✅ 验证通过")
	}

	fmt.Println("\n【示例 2】验证失败案例")
	fmt.Println("----------------------------------------")
	user2 := &User{
		Username: "ab",      // 太短
		Email:    "invalid", // 无效邮箱
		Password: "123",     // 太短
		Age:      15,        // 未成年
	}

	if err := Validate(user2, SceneCreate); err != nil {
		fmt.Printf("❌ 验证失败:\n%v\n", err)
	}

	fmt.Println("\n【示例 3】自定义验证器 - 带事件监听")
	fmt.Println("----------------------------------------")

	// 创建事件总线
	eventBus := NewSyncEventBus()
	eventBus.Subscribe(&ValidationLogger{})

	// 使用建造者模式创建自定义验证器
	validator := NewBuilder().
		WithEventBus(eventBus).
		WithErrorFormatter(NewChineseErrorFormatter()).
		WithMaxErrors(10).
		Build()

	user3 := &User{
		Username: "admin", // 保留字
		Email:    "admin@example.com",
		Password: "admin123",
		Age:      20,
	}

	if err := validator.Validate(user3, SceneCreate); err != nil {
		fmt.Printf("\n❌ 验证失败:\n%v\n", err)
	} else {
		fmt.Println("\n✅ 验证通过")
	}

	fmt.Println("\n【示例 4】部分字段验证")
	fmt.Println("----------------------------------------")
	user4 := &User{
		Username: "validuser",
		Email:    "invalid-email", // 故意错误
		Password: "password123",
		Age:      25,
	}

	// 只验证邮箱字段
	if err := ValidateFields(user4, SceneCreate, "email"); err != nil {
		fmt.Printf("❌ 邮箱验证失败: %v\n", err)
	}

	fmt.Println("\n【示例 5】高级配置 - 多级缓存 + 并发执行")
	fmt.Println("----------------------------------------")

	advancedValidator := NewBuilder().
		WithPipeline(NewConcurrentPipelineExecutor(4)). // 并发执行器
		WithRegistry(NewMultiLevelTypeRegistry(100)).   // 多级缓存
		WithErrorFormatter(NewChineseErrorFormatter()). // 中文错误
		WithMaxErrors(20).
		WithMaxDepth(10).
		Build()

	if err := advancedValidator.Validate(user1, SceneCreate); err != nil {
		fmt.Printf("❌ 验证失败: %v\n", err)
	} else {
		fmt.Println("✅ 验证通过")
	}

	fmt.Println("\n========================================")
	fmt.Println("  演示完成")
	fmt.Println("========================================")
}
