package v2_test

import (
	"fmt"
	"testing"

	v2 "katydid-common-account/pkg/validator/v2"
)

// ============================================================================
// 示例 1: 基本验证
// ============================================================================

type User struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
	Password string `json:"password"`
}

// 实现 RuleValidator 接口
func (u *User) ValidateRules() map[v2.Scene]v2.FieldRules {
	return map[v2.Scene]v2.FieldRules{
		"create": {
			"username": "required,min=3,max=20",
			"email":    "required,email",
			"age":      "required,gte=18",
			"password": "required,min=8",
		},
		"update": {
			"username": "omitempty,min=3,max=20",
			"email":    "omitempty,email",
			"age":      "omitempty,gte=18",
		},
	}
}

// 实现 CustomValidator 接口
func (u *User) ValidateCustom(scene v2.Scene, reporter v2.ErrorReporter) {
	// 创建场景的特殊验证
	if scene == "create" && u.Age < 18 {
		reporter.ReportMsg("User.Age", "min_age", "18", "用户必须年满18岁才能注册")
	}
}

func ExampleValidate_Basic() {
	user := &User{
		Username: "john",
		Email:    "john@example.com",
		Age:      25,
		Password: "password123",
	}

	// 验证
	result := v2.Validate(user, "create")

	if result.IsValid() {
		fmt.Println("验证通过")
	} else {
		fmt.Println("验证失败:")
		for _, err := range result.Errors() {
			fmt.Printf("  - %s\n", err.Error())
		}
	}

	// Output: 验证通过
}

// ============================================================================
// 示例 2: 部分字段验证
// ============================================================================

func ExampleValidateFields() {
	user := &User{
		Username: "jo", // 太短，不符合规则
		Email:    "valid@example.com",
		Age:      25,
	}

	// 只验证 username 和 email
	result := v2.ValidateFields(user, "update", "username", "email")

	if !result.IsValid() {
		fmt.Println("部分字段验证失败:")
		for _, err := range result.Errors() {
			fmt.Printf("  - 字段 %s: %s\n", err.Field, err.Error())
		}
	}
}

// ============================================================================
// 示例 3: 排除字段验证
// ============================================================================

func ExampleValidateExcept() {
	user := &User{
		Username: "john",
		Email:    "john@example.com",
		Age:      25,
		// 没有设置 password，但我们要排除它
	}

	// 验证除了 password 外的所有字段
	result := v2.ValidateExcept(user, "create", "password")

	if result.IsValid() {
		fmt.Println("验证通过（排除了 password）")
	}

	// Output: 验证通过（排除了 password）
}

// ============================================================================
// 示例 4: Map 验证
// ============================================================================

func ExampleMapValidator() {
	// 创建 Map 验证器
	validator := v2.NewMapValidator().
		WithNamespace("User.Extras").
		WithRequiredKeys("phone", "address").
		WithAllowedKeys("phone", "address", "avatar", "bio").
		WithKeyValidator("phone", v2.StringValidator(11, 11)).
		WithKeyValidator("address", v2.StringValidator(5, 200))

	// 测试数据
	data := map[string]any{
		"phone":   "13800138000",
		"address": "Beijing, China",
		"avatar":  "https://example.com/avatar.jpg",
	}

	// 验证
	errors := validator.Validate(data)

	if len(errors) == 0 {
		fmt.Println("Map 验证通过")
	} else {
		fmt.Println("Map 验证失败:")
		for _, err := range errors {
			fmt.Printf("  - %s\n", err.Error())
		}
	}

	// Output: Map 验证通过
}

// ============================================================================
// 示例 5: 场景化 Map 验证
// ============================================================================

func ExampleSceneMapValidators() {
	// 为不同场景创建不同的验证器
	createValidator := v2.NewMapValidator().
		WithNamespace("User.Profile").
		WithRequiredKeys("phone", "address", "realName").
		WithKeyValidator("phone", v2.StringValidator(11, 11))

	updateValidator := v2.NewMapValidator().
		WithNamespace("User.Profile").
		// 更新时不要求必填
		WithKeyValidator("phone", v2.StringValidator(11, 11))

	// 创建场景化验证器
	validators := v2.NewSceneMapValidators().
		WithScene("create", createValidator).
		WithScene("update", updateValidator)

	// 创建场景验证
	data := map[string]any{
		"phone":    "13800138000",
		"address":  "Beijing",
		"realName": "张三",
	}

	errors := validators.Validate("create", data)
	if len(errors) == 0 {
		fmt.Println("创建场景验证通过")
	}

	// Output: 创建场景验证通过
}

// ============================================================================
// 示例 6: 注册别名
// ============================================================================

func ExampleRegisterAlias() {
	// 注册常用的验证规则别名
	v2.RegisterAlias("password", "required,min=8,max=50,containsany=!@#$%^&*()")
	v2.RegisterAlias("phone", "required,len=11,numeric")
	v2.RegisterAlias("username", "required,min=3,max=20,alphanum")

	type Account struct {
		Username string `json:"username"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}

	// 使用别名
	account := &Account{}
	account.Username = "john123"
	account.Phone = "13800138000"
	account.Password = "Pass@123"

	// 在 RuleValidator 中使用别名
	// ...

	fmt.Println("别名注册成功")
	// Output: 别名注册成功
}

// ============================================================================
// 示例 7: 自定义验证策略
// ============================================================================

type IPWhitelistStrategy struct {
	allowedIPs map[string]bool
}

func NewIPWhitelistStrategy(ips []string) *IPWhitelistStrategy {
	allowed := make(map[string]bool)
	for _, ip := range ips {
		allowed[ip] = true
	}
	return &IPWhitelistStrategy{allowedIPs: allowed}
}

func (s *IPWhitelistStrategy) Execute(obj any, scene v2.Scene, collector v2.ErrorCollector) bool {
	// 假设对象有 IP 字段
	type HasIP interface {
		GetIP() string
	}

	if ipObj, ok := obj.(HasIP); ok {
		ip := ipObj.GetIP()
		if !s.allowedIPs[ip] {
			collector.Add(v2.NewFieldError("IP", "ip", "whitelist", "").
				WithMessage(fmt.Sprintf("IP %s is not in whitelist", ip)))
		}
	}

	return true // 继续执行后续策略
}

// ============================================================================
// 示例 8: 结果查询
// ============================================================================

func ExampleResult_Query() {
	user := &User{
		Username: "jo",      // 太短
		Email:    "invalid", // 无效邮箱
		Age:      16,        // 太小
	}

	result := v2.Validate(user, "create")

	// 查询所有错误
	fmt.Printf("总错误数: %d\n", len(result.Errors()))

	// 获取第一个错误
	if firstErr := result.FirstError(); firstErr != nil {
		fmt.Printf("第一个错误: %s\n", firstErr.Field)
	}

	// 按字段查询
	usernameErrors := result.ErrorsByField("username")
	fmt.Printf("username 字段错误数: %d\n", len(usernameErrors))

	// 按标签查询
	requiredErrors := result.ErrorsByTag("required")
	fmt.Printf("required 标签错误数: %d\n", len(requiredErrors))
}

// ============================================================================
// 示例 9: 嵌套结构验证
// ============================================================================

type Profile struct {
	RealName string `json:"realName" validate:"required,min=2"`
	Bio      string `json:"bio" validate:"max=500"`
}

type UserWithProfile struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Profile  *Profile `json:"profile"` // 嵌套结构
}

func (u *UserWithProfile) ValidateRules() map[v2.Scene]v2.FieldRules {
	return map[v2.Scene]v2.FieldRules{
		"create": {
			"username": "required,min=3",
			"email":    "required,email",
		},
	}
}

func ExampleNestedValidation() {
	user := &UserWithProfile{
		Username: "john",
		Email:    "john@example.com",
		Profile: &Profile{
			RealName: "J", // 太短
			Bio:      "Developer",
		},
	}

	result := v2.Validate(user, "create")

	if !result.IsValid() {
		fmt.Println("嵌套验证失败:")
		for _, err := range result.Errors() {
			fmt.Printf("  - %s: %s\n", err.Namespace, err.Error())
		}
	}
}

// ============================================================================
// 基准测试
// ============================================================================

func BenchmarkValidate(b *testing.B) {
	user := &User{
		Username: "john",
		Email:    "john@example.com",
		Age:      25,
		Password: "password123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v2.Validate(user, "create")
	}
}

func BenchmarkValidateFields(b *testing.B) {
	user := &User{
		Username: "john",
		Email:    "john@example.com",
		Age:      25,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v2.ValidateFields(user, "update", "username", "email")
	}
}

func BenchmarkMapValidate(b *testing.B) {
	validator := v2.NewMapValidator().
		WithRequiredKeys("phone", "address").
		WithKeyValidator("phone", v2.StringValidator(11, 11))

	data := map[string]any{
		"phone":   "13800138000",
		"address": "Beijing",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.Validate(data)
	}
}
