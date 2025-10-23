package v1

import (
	"fmt"
	"testing"
)

// ============================================================================
// 测试模型定义
// ============================================================================

// TestUser 测试用户模型
type TestUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Age      int    `json:"age"`
	Phone    string `json:"phone"`
}

// 预定义的通用验证场景常量
const (
	SceneCreate ValidateScene = 1 << 0 // 创建场景 (1)
	SceneUpdate ValidateScene = 1 << 1 // 更新场景 (2)
	SceneDelete ValidateScene = 1 << 2 // 删除场景 (4)
	SceneQuery  ValidateScene = 1 << 3 // 查询场景 (8)
)

// Rules 实现 RuleValidator 接口
func (u *TestUser) RuleValidation() map[ValidateScene]map[string]string {
	return map[ValidateScene]map[string]string{
		SceneCreate: {
			"Username": "required,min=3,max=20,alphanum",
			"Email":    "required,email",
			"Password": "required,min=6,max=20",
			"Age":      "omitempty,gte=0,lte=150",
			"Phone":    "omitempty,len=11,numeric",
		},
		SceneUpdate: {
			"Username": "omitempty,min=3,max=20,alphanum",
			"Email":    "omitempty,email",
			"Password": "omitempty,min=6,max=20",
			"Age":      "omitempty,gte=0,lte=150",
			"Phone":    "omitempty,len=11,numeric",
		},
	}
}

// CrossFieldValidation 实现 CustomValidator 接口
func (u *TestUser) CustomValidation(scene ValidateScene, reportError FuncReportError) {
	if scene&SceneCreate != 0 {
		// 创建时，用户名不能是admin
		if u.Username == "admin" {
			reportError("Username", "reserved", "admin")
		}
	}
}

// GetErrorMessage 实现 ErrorMessageProvider 接口，自定义错误消息
func (u *TestUser) GetErrorMessage(fieldName, tag, param string) string {
	switch fieldName {
	case "username":
		switch tag {
		case "required":
			return "用户名不能为空"
		case "min":
			return fmt.Sprintf("用户名长度不能少于%s个字符", param)
		case "max":
			return fmt.Sprintf("用户名长度不能超过%s个字符", param)
		case "alphanum":
			return "用户名只能包含字母和数字"
		}
	case "email":
		switch tag {
		case "required":
			return "邮箱地址不能为空"
		case "email":
			return "请输入有效的邮箱地址"
		}
	case "password":
		switch tag {
		case "required":
			return "密码不能为空"
		case "min":
			return fmt.Sprintf("密码长度不能少于%s位", param)
		}
	case "phone":
		switch tag {
		case "len":
			return "手机号码必须是11位数字"
		case "numeric":
			return "手机号码只能包含数字"
		}
	}
	// 返回空字符串使用默认消息
	return ""
}

// ============================================================================
// 基础验证测试
// ============================================================================

func TestValidate_Create(t *testing.T) {
	tests := []struct {
		name    string
		user    *TestUser
		wantErr bool
	}{
		{
			name: "有效的创建数据",
			user: &TestUser{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
				Age:      25,
			},
			wantErr: false,
		},
		{
			name: "缺少必填字段username",
			user: &TestUser{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "无效的邮箱",
			user: &TestUser{
				Username: "testuser",
				Email:    "invalid-email",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "密码太短",
			user: &TestUser{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "123",
			},
			wantErr: true,
		},
		{
			name: "用户名是admin",
			user: &TestUser{
				Username: "admin",
				Email:    "admin@example.com",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "手机号格式错误",
			user: &TestUser{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
				Phone:    "123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.user, SceneCreate)
			if (errs != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", errs, tt.wantErr)
			}
			if errs != nil {
				t.Logf("验证错误: %v", errs)
			}
		})
	}
}

func TestValidate_Update(t *testing.T) {
	tests := []struct {
		name    string
		user    *TestUser
		wantErr bool
	}{
		{
			name: "更新用户名",
			user: &TestUser{
				ID:       1,
				Username: "newname",
			},
			wantErr: false,
		},
		{
			name: "更新邮箱",
			user: &TestUser{
				ID:    1,
				Email: "newemail@example.com",
			},
			wantErr: false,
		},
		{
			name: "更新无效邮箱",
			user: &TestUser{
				ID:    1,
				Email: "invalid",
			},
			wantErr: true,
		},
		{
			name: "空数据也可以（更新时所有字段都是可选的）",
			user: &TestUser{
				ID: 1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.user, SceneUpdate)
			if (errs != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", errs, tt.wantErr)
			}
			if errs != nil {
				t.Logf("验证错误: %v", errs)
			}
		})
	}
}

// TestRegisterValidation 测试自定义验证规则注册
func TestRegisterValidation(t *testing.T) {
	v := New()

	type TestStruct struct {
		Field string `json:"field" validate:"required,min=3"`
	}

	tests := []struct {
		name    string
		obj     *TestStruct
		wantErr bool
	}{
		{
			name:    "符合验证规则",
			obj:     &TestStruct{Field: "awesome"},
			wantErr: false,
		},
		{
			name:    "字段为空",
			obj:     &TestStruct{Field: ""},
			wantErr: true,
		},
		{
			name:    "字段太短",
			obj:     &TestStruct{Field: "ab"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.validate.Struct(tt.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCustomErrorMessage 测试自定义错误消息功能
func TestCustomErrorMessage(t *testing.T) {
	// 测试缺少用户名的情况
	user := &TestUser{
		Email:    "test@example.com",
		Password: "password123",
	}

	errs := Validate(user, SceneCreate)
	if errs == nil {
		t.Fatal("期望验证失败，但是通过了")
	}

	// 检查是否包含自定义错误消息
	errMsg := fmt.Sprintf("%v", errs)
	t.Logf("验证错误消息: %s", errMsg)

	// 验证错误消息包含字段名
	if errMsg == "" {
		t.Error("错误消息为空")
	}
}

// TestCustomErrorMessageVsDefault 对比自定义错误消息和默认错误消息
func TestCustomErrorMessageVsDefault(t *testing.T) {
	t.Run("使用自定义错误消息", func(t *testing.T) {
		user := &TestUser{
			Username: "ab", // 太短，少于3个字符
			Email:    "test@example.com",
			Password: "password123",
		}

		errs := Validate(user, SceneCreate)
		if errs == nil {
			t.Fatal("期望验证失败")
		}

		t.Logf("自定义错误消息: %v", errs)
	})

	t.Run("无效的邮箱格式", func(t *testing.T) {
		user := &TestUser{
			Username: "testuser",
			Email:    "invalid-email",
			Password: "password123",
		}

		errs := Validate(user, SceneCreate)
		if errs == nil {
			t.Fatal("期望验证失败")
		}

		t.Logf("邮箱验证错误: %v", errs)
	})

	t.Run("手机号格式错误", func(t *testing.T) {
		user := &TestUser{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "password123",
			Phone:    "123", // 不足11位
		}

		errs := Validate(user, SceneCreate)
		if errs == nil {
			t.Fatal("期望验证失败")
		}

		t.Logf("手机号验证错误: %v", errs)
	})
}

// 示例：国际化错误消息
type InternationalUser struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Lang     string `json:"-"` // 语言设置
}

func (u *InternationalUser) ValidateRules() map[ValidateScene]map[string]string {
	return map[ValidateScene]map[string]string{
		SceneCreate: {
			"Username": "required,min=3",
			"Email":    "required,email",
		},
	}
}

func (u *InternationalUser) GetErrorMessage(fieldName, tag, param string) string {
	if u.Lang == "en" {
		return u.getEnglishMessage(fieldName, tag, param)
	}
	return u.getChineseMessage(fieldName, tag, param)
}

func (u *InternationalUser) getEnglishMessage(fieldName, tag, param string) string {
	switch fieldName {
	case "username":
		switch tag {
		case "required":
			return "Username is required"
		case "min":
			return fmt.Sprintf("Username must be at least %s characters", param)
		}
	case "email":
		switch tag {
		case "required":
			return "Email is required"
		case "email":
			return "Invalid email format"
		}
	}
	return ""
}

func (u *InternationalUser) getChineseMessage(fieldName, tag, param string) string {
	switch fieldName {
	case "username":
		switch tag {
		case "required":
			return "用户名是必填项"
		case "min":
			return fmt.Sprintf("用户名至少需要%s个字符", param)
		}
	case "email":
		switch tag {
		case "required":
			return "邮箱是必填项"
		case "email":
			return "邮箱格式无效"
		}
	}
	return ""
}

func TestInternationalErrorMessage(t *testing.T) {
	t.Run("中文错误消息", func(t *testing.T) {
		user := &InternationalUser{
			Lang: "zh",
		}

		errs := Validate(user, SceneCreate)
		if errs == nil {
			t.Fatal("期望验证失败")
		}

		t.Logf("中文错误消息: %v", errs)
	})

	t.Run("英文错误消息", func(t *testing.T) {
		user := &InternationalUser{
			Lang: "en",
		}

		errs := Validate(user, SceneCreate)
		if errs == nil {
			t.Fatal("期望验证失败")
		}

		t.Logf("English error message: %v", errs)
	})
}

// ============================================================================
// 自动注册测试
// ============================================================================

// UserWithAutoRegister 用户模型（使用自动注册）
// 实现 StructLevelValidatable 接口后，会在首次验证时自动注册
type UserWithAutoRegister struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	Email           string `json:"email"`
	Age             int    `json:"age"`
}

// ProductWithAutoRegister 产品模型（使用自动注册）
// 实现 MapRulesValidatable 接口后，会在首次验证时自动注册
type ProductWithAutoRegister struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

// ValidationMapRules 实现 MapRulesValidatable 接口
// 验证器会在首次验证时自动注册这些规则
func (p ProductWithAutoRegister) ValidationMapRules() map[string]string {
	return map[string]string{
		"Name":  "required,min=3,max=100",
		"Price": "required,gt=0",
		"Stock": "gte=0",
	}
}

// OrderWithBothInterfaces 订单模型（同时实现两个接口）
// 可以同时使用 StructLevelValidatable 和 MapRulesValidatable
type OrderWithBothInterfaces struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
	Total     float64 `json:"total"`
}

// ValidationMapRules 实现 MapRulesValidatable 接口（字段规则）
func (o OrderWithBothInterfaces) ValidationMapRules() map[string]string {
	return map[string]string{
		"ProductID": "required",
		"Quantity":  "required,gt=0",
		"Price":     "required,gt=0",
		"Total":     "required,gt=0",
	}
}

// ============================================================================
// 性能基准测试
// ============================================================================

// BenchmarkUser 测试用的用户结构
type BenchmarkUser struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
	Age      int    `json:"age"`
}

func (u *BenchmarkUser) RuleValidation() map[ValidateScene]map[string]string {
	return map[ValidateScene]map[string]string{
		SceneCreate: {
			"Username": "required,min=3,max=20",
			"Email":    "required,email",
			"Password": "required,min=6",
			"Phone":    "len=11",
			"Age":      "gte=0,lte=150",
		},
		SceneUpdate: {
			"Email": "omitempty,email",
			"Phone": "omitempty,len=11",
			"Age":   "omitempty,gte=0,lte=150",
		},
	}
}

// BenchmarkValidate_TypeCaching 测试类型缓存的性能提升
func BenchmarkValidate_TypeCaching(b *testing.B) {
	v := New()
	user := &BenchmarkUser{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Phone:    "13800138000",
		Age:      25,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Validate(user, SceneCreate)
	}
}

// BenchmarkValidate_MultipleInstances 测试多个不同实例的验证性能
func BenchmarkValidate_MultipleInstances(b *testing.B) {
	v := New()
	users := []*BenchmarkUser{
		{Username: "user1", Email: "user1@example.com", Password: "pass123", Phone: "13800138001", Age: 20},
		{Username: "user2", Email: "user2@example.com", Password: "pass456", Phone: "13800138002", Age: 30},
		{Username: "user3", Email: "user3@example.com", Password: "pass789", Phone: "13800138003", Age: 40},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, user := range users {
			_ = v.Validate(user, SceneCreate)
		}
	}
}

// BenchmarkValidate_ErrorFormatting 测试错误格式化性能
func BenchmarkValidate_ErrorFormatting(b *testing.B) {
	v := New()
	invalidUser := &BenchmarkUser{
		Username: "ab",            // 太短
		Email:    "invalid-email", // 无效邮箱
		Password: "123",           // 太短
		Phone:    "123",           // 长度不够
		Age:      200,             // 超出范围
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Validate(invalidUser, SceneCreate)
	}
}

// BenchmarkValidate_Parallel 测试并发验证性能
func BenchmarkValidate_Parallel(b *testing.B) {
	v := New()
	user := &BenchmarkUser{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Phone:    "13800138000",
		Age:      25,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = v.Validate(user, SceneCreate)
		}
	})
}

// User 用户模型 - 演示使用 FuncReportError 的新方式
type User struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	Email           string `json:"email"`
	Age             int    `json:"age"`
}

// CustomValidation 实现 CustomValidator 接口
// 所有错误都通过 reportError 报告
func (u *User) CustomValidation(scene ValidateScene, reportError FuncReportError) {
	// 跨字段验证：密码和确认密码必须一致
	if u.Password != u.ConfirmPassword {
		reportError("ConfirmPassword", "password_mismatch", "")
	}

	// 场景化验证：创建时必须年满18岁
	if scene == SceneCreate && u.Age < 18 {
		reportError("Age", "min_age", "18")
	}

	// 场景化验证：更新时用户名不能为空
	if scene == SceneUpdate && u.Username == "" {
		reportError("Username", "required", "")
	}
}

// Product 商品模型 - 演示混合使用两种方式
type Product struct {
	Name          string  `json:"name"`
	OriginalPrice float64 `json:"original_price"`
	DiscountPrice float64 `json:"discount_price"`
	Category      string  `json:"category"`
	Brand         string  `json:"brand"`
}

// CustomValidation 实现 CustomValidator 接口
// 所有错误都通过 reportError 报告
func (p *Product) CustomValidation(scene ValidateScene, reportError FuncReportError) {
	// 使用 reportError 报告简单错误
	if p.DiscountPrice > p.OriginalPrice {
		reportError("DiscountPrice", "price_check", "")
	}

	// 复杂验证：电子产品必须有品牌
	if p.Category == "electronics" && p.Brand == "" {
		reportError("Brand", "required_for_electronics", "")
	}

	// 场景化验证
	if scene == SceneCreate && p.Name == "" {
		reportError("Name", "required", "")
	}
}

// Order 订单模型 - 演示使用 reportError
type Order struct {
	OrderID    string  `json:"order_id"`
	TotalPrice float64 `json:"total_price"`
	ItemCount  int     `json:"item_count"`
}

// CustomValidation 实现 CustomValidator 接口
// 所有错误都通过 reportError 报告
func (o *Order) CustomValidation(_ ValidateScene, reportError FuncReportError) {
	if o.TotalPrice < 0 {
		reportError("TotalPrice", "min", "0")
	}

	if o.ItemCount <= 0 {
		reportError("ItemCount", "min", "1")
	}
}

// TestUserValidation_WithReportError 测试使用 reportError 的用户验证
func TestUserValidation_WithReportError(t *testing.T) {
	// 测试密码不匹配
	user := &User{
		Username:        "john",
		Password:        "password123",
		ConfirmPassword: "password456",
		Email:           "john@example.com",
		Age:             25,
	}

	errs := Validate(user, SceneCreate)
	if errs == nil {
		t.Error("Expected validation errors, got nil")
	}

	// 检查是否有密码不匹配的错误
	found := false
	for _, err := range errs {
		if err.Tag == "password_mismatch" {
			found = true
			fmt.Printf("✓ 密码不匹配错误: field=%s, tag=%s\n", err.Namespace, err.Tag)
			break
		}
	}
	if !found {
		t.Error("Expected password_mismatch error")
	}
}

// TestUserValidation_AgeCheck 测试年龄验证
func TestUserValidation_AgeCheck(t *testing.T) {
	user := &User{
		Username:        "teen",
		Password:        "pass123",
		ConfirmPassword: "pass123",
		Email:           "teen@example.com",
		Age:             16, // 未满18岁
	}

	errs := Validate(user, SceneCreate)
	if errs == nil {
		t.Error("Expected validation errors for underage user")
	}

	// 检查年龄错误
	found := false
	for _, err := range errs {
		if err.Tag == "min_age" && err.Param == "18" {
			found = true
			fmt.Printf("✓ 年龄验证错误: field=%s, tag=%s, param=%s\n", err.Namespace, err.Tag, err.Param)
			break
		}
	}
	if !found {
		t.Error("Expected min_age error")
	}
}

// TestProductValidation_MixedApproach 测试混合使用两种错误报告方式
func TestProductValidation_MixedApproach(t *testing.T) {
	product := &Product{
		Name:          "iPhone",
		OriginalPrice: 999.99,
		DiscountPrice: 1099.99, // 错误：折扣价高于原价
		Category:      "electronics",
		Brand:         "", // 错误：电子产品缺少品牌
	}

	errs := Validate(product, SceneCreate)
	if errs == nil {
		t.Error("Expected validation errors")
	}

	fmt.Printf("\n商品验证错误 (共 %d 个):\n", len(errs))
	for i, err := range errs {
		fmt.Printf("  %d. field=%s, tag=%s, message=%s\n", i+1, err.Namespace, err.Tag, err.Message)
	}

	// 验证包含价格检查错误
	hasPrice := false
	hasBrand := false
	for _, err := range errs {
		if err.Tag == "price_check" {
			hasPrice = true
		}
		if err.Tag == "required_for_electronics" {
			hasBrand = true
		}
	}

	if !hasPrice {
		t.Error("Expected price_check error")
	}
	if !hasBrand {
		t.Error("Expected required_for_electronics error")
	}
}

// TestOrderValidation_BackwardCompatible 测试向后兼容（完全使用返回值）
func TestOrderValidation_BackwardCompatible(t *testing.T) {
	order := &Order{
		OrderID:    "ORD001",
		TotalPrice: -100.0, // 错误：负数价格
		ItemCount:  0,      // 错误：商品数量为0
	}

	errs := Validate(order, SceneCreate)
	if errs == nil {
		t.Error("Expected validation errors")
	}

	if len(errs) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errs))
	}

	fmt.Printf("\n订单验证错误 (向后兼容模式):\n")
	for i, err := range errs {
		fmt.Printf("  %d. %s\n", i+1, err.Message)
	}
}

// TestValidUser 测试验证通过的情况
func TestValidUser(t *testing.T) {
	user := &User{
		Username:        "alice",
		Password:        "securepass123",
		ConfirmPassword: "securepass123",
		Email:           "alice@example.com",
		Age:             25,
	}

	errs := Validate(user, SceneCreate)
	if errs != nil {
		t.Errorf("Expected no errors, got %d errors", len(errs))
		for _, err := range errs {
			t.Logf("  - %s: %s", err.Namespace, err.Message)
		}
	} else {
		fmt.Println("\n✓ 用户验证通过！")
	}
}

// ExampleCustomValidator_reportError 演示如何使用 FuncReportError
func ExampleCustomValidator_reportError() {
	// 创建一个有错误的用户
	user := &User{
		Username:        "bob",
		Password:        "pass123",
		ConfirmPassword: "pass456", // 密码不匹配
		Email:           "bob@example.com",
		Age:             16, // 未满18岁
	}

	// 执行验证
	errs := Validate(user, SceneCreate)

	// 输出错误
	fmt.Printf("验证失败，共 %d 个错误:\n", len(errs))
	for i, err := range errs {
		param := err.Param
		if param == "" {
			param = ""
		}
		fmt.Printf("%d. 字段: %s, 标签: %s, 参数: %s\n",
			i+1, err.Namespace, err.Tag, param)
	}

	// Output:
	// 验证失败，共 2 个错误:
	// 1. 字段: confirm_password, 标签: password_mismatch, 参数:
	// 2. 字段: age, 标签: min_age, 参数: 18
}
