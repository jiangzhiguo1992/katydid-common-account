package main

import (
	"fmt"

	v2 "katydid-common-account/pkg/validator/v2"
)

// ============================================================================
// 示例 1: 基础使用 - 简单用户模型
// ============================================================================

type User struct {
	Username        string `json:"username"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	Age             int    `json:"age"`
}

// 实现 RuleProvider 接口 - 提供字段验证规则
func (u *User) ProvideRules() map[v2.Scene]v2.FieldRules {
	return map[v2.Scene]v2.FieldRules{
		v2.SceneCreate: {
			"Username": "required,min=3,max=20,alphanum",
			"Email":    "required,email",
			"Password": "required,min=6,max=20",
			"Age":      "omitempty,gte=0,lte=150",
		},
		v2.SceneUpdate: {
			"Username": "omitempty,min=3,max=20,alphanum",
			"Email":    "omitempty,email",
			"Password": "omitempty,min=6,max=20",
		},
	}
}

// 实现 CustomValidator 接口 - 自定义验证逻辑
func (u *User) ValidateCustom(scene v2.Scene, reporter v2.ErrorReporter) {
	// 跨字段验证：密码和确认密码必须一致
	if u.Password != "" && u.Password != u.ConfirmPassword {
		reporter.ReportWithMessage(
			"User.ConfirmPassword",
			"password_mismatch",
			"",
			"密码和确认密码不一致",
		)
	}

	// 场景化验证：创建时年龄必须大于等于 18
	if scene == v2.SceneCreate && u.Age > 0 && u.Age < 18 {
		reporter.ReportWithMessage(
			"User.Age",
			"min_age",
			"18",
			"创建用户时年龄必须大于等于 18 岁",
		)
	}

	// 业务规则验证：用户名不能是保留字
	reservedNames := map[string]bool{
		"admin":  true,
		"root":   true,
		"system": true,
	}

	if reservedNames[u.Username] {
		reporter.ReportWithMessage(
			"User.Username",
			"reserved",
			"",
			"用户名是系统保留字，不能使用",
		)
	}
}

func exampleBasicUsage() {
	fmt.Println("========== 示例 1: 基础使用 ==========")

	// 创建用户
	user := &User{
		Username:        "john",
		Email:           "john@example.com",
		Password:        "password123",
		ConfirmPassword: "password123",
		Age:             25,
	}

	// 使用全局验证器验证
	result := v2.Validate(user, v2.SceneCreate)

	if result.IsValid() {
		fmt.Println("✓ 验证通过")
	} else {
		fmt.Println("✗ 验证失败:")
		for _, err := range result.Errors() {
			fmt.Printf("  - %s: %s\n", err.Field, err.Message)
		}
	}

	fmt.Println()
}

// ============================================================================
// 示例 2: 场景化验证
// ============================================================================

func exampleSceneValidation() {
	fmt.Println("========== 示例 2: 场景化验证 ==========")

	user := &User{
		Username:        "john",
		Email:           "john@example.com",
		Password:        "", // 更新时密码可选
		ConfirmPassword: "",
		Age:             16, // 创建时会失败，更新时会通过
	}

	// 创建场景验证
	fmt.Println("创建场景验证:")
	createResult := v2.Validate(user, v2.SceneCreate)
	if !createResult.IsValid() {
		for _, err := range createResult.Errors() {
			fmt.Printf("  - %s: %s\n", err.Field, err.Message)
		}
	}

	// 更新场景验证
	fmt.Println("\n更新场景验证:")
	updateResult := v2.Validate(user, v2.SceneUpdate)
	if updateResult.IsValid() {
		fmt.Println("  ✓ 验证通过（更新场景没有年龄限制）")
	} else {
		for _, err := range updateResult.Errors() {
			fmt.Printf("  - %s: %s\n", err.Field, err.Message)
		}
	}

	fmt.Println()
}

// ============================================================================
// 示例 3: Map 字段验证
// ============================================================================

type Product struct {
	Name     string         `json:"name"`
	Category string         `json:"category"`
	Price    float64        `json:"price"`
	Extras   map[string]any `json:"extras"`
}

func (p *Product) ProvideRules() map[v2.Scene]v2.FieldRules {
	return map[v2.Scene]v2.FieldRules{
		v2.SceneCreate: {
			"Name":     "required,min=3,max=100",
			"Category": "required,oneof=electronics clothing food",
			"Price":    "required,gt=0",
		},
	}
}

func (p *Product) ValidateCustom(scene v2.Scene, reporter v2.ErrorReporter) {
	if p.Extras == nil {
		return
	}

	// 根据分类验证不同的 Extras 字段
	switch p.Category {
	case "electronics":
		// 验证必填字段
		if err := v2.ValidateMapRequired(p.Extras, "brand", "warranty"); err != nil {
			reporter.ReportWithMessage("Product.Extras", "required_keys", "", err.Error())
		}

		// 验证字符串字段
		if err := v2.ValidateMapString(p.Extras, "brand", 2, 50); err != nil {
			reporter.ReportWithMessage("Product.Extras.brand", "invalid", "", err.Error())
		}

		// 验证整数字段
		if err := v2.ValidateMapInt(p.Extras, "warranty", 12, 60); err != nil {
			reporter.ReportWithMessage("Product.Extras.warranty", "invalid", "", err.Error())
		}

	case "clothing":
		// 验证必填字段
		if err := v2.ValidateMapRequired(p.Extras, "size", "color"); err != nil {
			reporter.ReportWithMessage("Product.Extras", "required_keys", "", err.Error())
		}

		// 自定义验证：尺码必须是指定值之一
		if err := v2.ValidateMapKey(p.Extras, "size", func(value any) error {
			size, ok := value.(string)
			if !ok {
				return fmt.Errorf("尺码必须是字符串类型")
			}
			validSizes := map[string]bool{"S": true, "M": true, "L": true, "XL": true}
			if !validSizes[size] {
				return fmt.Errorf("尺码必须是 S, M, L, XL 之一，当前值: %s", size)
			}
			return nil
		}); err != nil {
			reporter.ReportWithMessage("Product.Extras.size", "invalid", "", err.Error())
		}
	}
}

func exampleMapValidation() {
	fmt.Println("========== 示例 3: Map 字段验证 ==========")

	// 电子产品
	electronics := &Product{
		Name:     "iPhone 15",
		Category: "electronics",
		Price:    6999.00,
		Extras: map[string]any{
			"brand":    "Apple",
			"warranty": 24,
			"color":    "Black",
		},
	}

	fmt.Println("验证电子产品:")
	result := v2.Validate(electronics, v2.SceneCreate)
	if result.IsValid() {
		fmt.Println("  ✓ 验证通过")
	} else {
		for _, err := range result.Errors() {
			fmt.Printf("  - %s: %s\n", err.Field, err.Message)
		}
	}

	// 服装产品
	clothing := &Product{
		Name:     "T-Shirt",
		Category: "clothing",
		Price:    99.00,
		Extras: map[string]any{
			"size":  "M",
			"color": "Blue",
		},
	}

	fmt.Println("\n验证服装产品:")
	result = v2.Validate(clothing, v2.SceneCreate)
	if result.IsValid() {
		fmt.Println("  ✓ 验证通过")
	} else {
		for _, err := range result.Errors() {
			fmt.Printf("  - %s: %s\n", err.Field, err.Message)
		}
	}

	// 无效的服装产品
	invalidClothing := &Product{
		Name:     "T-Shirt",
		Category: "clothing",
		Price:    99.00,
		Extras: map[string]any{
			"size":  "XXL", // 无效的尺码
			"color": "Blue",
		},
	}

	fmt.Println("\n验证无效的服装产品:")
	result = v2.Validate(invalidClothing, v2.SceneCreate)
	if !result.IsValid() {
		for _, err := range result.Errors() {
			fmt.Printf("  - %s: %s\n", err.Field, err.Message)
		}
	}

	fmt.Println()
}

// ============================================================================
// 示例 4: 使用 MapValidator 进行结构化验证
// ============================================================================

func exampleMapValidator() {
	fmt.Println("========== 示例 4: 使用 MapValidator ==========")

	extras := map[string]any{
		"brand":    "Sony",
		"warranty": 36,
		"color":    "Silver",
		"weight":   500, // 不在允许列表中
	}

	// 创建 MapValidator
	validator := v2.NewMapValidator().
		WithNamespace("Product.Extras").
		WithRequiredKeys("brand", "warranty").
		WithAllowedKeys("brand", "warranty", "color").
		WithKeyValidator("warranty", func(value any) error {
			warranty, ok := value.(int)
			if !ok {
				return fmt.Errorf("warranty 必须是整数类型")
			}
			if warranty < 12 || warranty > 60 {
				return fmt.Errorf("warranty 必须在 12 到 60 个月之间，当前值: %d", warranty)
			}
			return nil
		})

	// 验证
	errors := validator.Validate(extras)

	if len(errors) == 0 {
		fmt.Println("  ✓ 验证通过")
	} else {
		fmt.Println("  ✗ 验证失败:")
		for _, err := range errors {
			fmt.Printf("    - %s: %s\n", err.Field, err.Message)
		}
	}

	fmt.Println()
}

// ============================================================================
// 示例 5: 自定义验证器配置（建造者模式）
// ============================================================================

func exampleCustomValidator() {
	fmt.Println("========== 示例 5: 自定义验证器配置 ==========")

	// 使用建造者创建自定义配置的验证器
	validator := v2.NewValidatorBuilder().
		WithMaxDepth(50). // 设置最大嵌套深度
		WithDefaultStrategies().
		Build()

	user := &User{
		Username:        "john",
		Email:           "john@example.com",
		Password:        "password123",
		ConfirmPassword: "password123",
		Age:             25,
	}

	result := validator.Validate(user, v2.SceneCreate)

	if result.IsValid() {
		fmt.Println("  ✓ 验证通过")
	} else {
		for _, err := range result.Errors() {
			fmt.Printf("  - %s: %s\n", err.Field, err.Message)
		}
	}

	fmt.Println()
}

// ============================================================================
// 示例 6: 验证结果的多种查询方式
// ============================================================================

func exampleResultQuery() {
	fmt.Println("========== 示例 6: 验证结果查询 ==========")

	user := &User{
		Username:        "ab",      // 太短
		Email:           "invalid", // 无效邮箱
		Password:        "123",     // 太短
		ConfirmPassword: "456",     // 不匹配
		Age:             16,        // 创建时年龄不够
	}

	result := v2.Validate(user, v2.SceneCreate)

	fmt.Printf("验证是否通过: %v\n", result.IsValid())
	fmt.Printf("错误总数: %d\n", len(result.Errors()))

	// 获取第一个错误
	if firstErr := result.FirstError(); firstErr != nil {
		fmt.Printf("\n第一个错误:\n  字段: %s\n  标签: %s\n  消息: %s\n",
			firstErr.Field, firstErr.Tag, firstErr.Message)
	}

	// 按字段筛选错误
	if usernameErrors := result.ErrorsByField("username"); len(usernameErrors) > 0 {
		fmt.Printf("\nusername 字段的错误:\n")
		for _, err := range usernameErrors {
			fmt.Printf("  - %s: %s\n", err.Tag, err.Message)
		}
	}

	// 按标签筛选错误
	if requiredErrors := result.ErrorsByTag("required"); len(requiredErrors) > 0 {
		fmt.Printf("\n必填项错误:\n")
		for _, err := range requiredErrors {
			fmt.Printf("  - %s\n", err.Field)
		}
	}

	fmt.Println()
}

// ============================================================================
// 主函数
// ============================================================================

func main() {
	fmt.Println("Validator V2 使用示例")
	fmt.Println("=" + string(make([]byte, 50)) + "\n")

	exampleBasicUsage()
	exampleSceneValidation()
	exampleMapValidation()
	exampleMapValidator()
	exampleCustomValidator()
	exampleResultQuery()

	fmt.Println("所有示例运行完成！")
}
