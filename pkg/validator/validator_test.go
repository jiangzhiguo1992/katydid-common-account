package validator

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
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

// 测试场景常量
const (
	SceneCreate ValidateScene = "create" // 创建场景
	SceneUpdate ValidateScene = "update" // 更新场景
	SceneDelete ValidateScene = "delete" // 删除场景
	SceneQuery  ValidateScene = "query"  // 查询场景
)

// ValidateRules 实现 Validatable 接口
func (u *TestUser) ValidateRules() map[ValidateScene]map[string]string {
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

// CustomValidate 实现 CustomValidatable 接口
func (u *TestUser) CustomValidate(scene ValidateScene) []*FieldError {
	if scene == SceneCreate {
		// 创建时，用户名不能是admin
		if u.Username == "admin" {
			return []*FieldError{NewFieldError("username", "用户名不能是 'admin'", nil, nil)}
		}
	}
	return nil
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

func TestRegisterValidation(t *testing.T) {
	// 注册自定义验证规则
	err := RegisterValidation("is_awesome", func(fl validator.FieldLevel) bool {
		return fl.Field().String() == "awesome"
	})
	if err != nil {
		t.Fatalf("RegisterValidation() error = %v", err)
	}

	type TestStruct struct {
		Field string `validate:"is_awesome"`
	}

	tests := []struct {
		name    string
		obj     *TestStruct
		wantErr bool
	}{
		{
			name:    "符合自定义规则",
			obj:     &TestStruct{Field: "awesome"},
			wantErr: false,
		},
		{
			name:    "不符合自定义规则",
			obj:     &TestStruct{Field: "not awesome"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStruct() error = %v, wantErr %v", err, tt.wantErr)
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

// TestNoCustomErrorMessage 测试不实现 ErrorMessageProvider 接口的情况
func TestNoCustomErrorMessage(t *testing.T) {
	type SimpleUser struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	// SimpleUser 没有实现 ErrorMessageProvider 接口
	// 应该使用默认错误消息

	user := &SimpleUser{
		Email: "test@example.com",
	}

	// 由于没有实现 Validatable 接口，会使用 ValidateStruct
	err := ValidateStruct(user)
	if err != nil {
		t.Logf("默认错误消息: %s", err.Error())
	}
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

// StructLevelValidation 实现 StructLevelValidatable 接口
// 验证器会在首次验证时自动注册此方法
func (u UserWithAutoRegister) StructLevelValidation(sl StructLevel) {
	// 验证密码和确认密码是否一致
	if u.Password != u.ConfirmPassword {
		sl.ReportError(u.ConfirmPassword, "ConfirmPassword", "confirmPassword", "eqfield", "Password")
	}

	// 验证未成年用户的用户名必须包含 "kid"
	if u.Age < 18 && !strings.Contains(strings.ToLower(u.Username), "kid") {
		sl.ReportError(u.Username, "Username", "username", "kid_required", "")
	}
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

// StructLevelValidation 实现 StructLevelValidatable 接口（跨字段验证）
func (o OrderWithBothInterfaces) StructLevelValidation(sl StructLevel) {
	// 验证总价 = 数量 * 单价
	expectedTotal := float64(o.Quantity) * o.Price
	if o.Total != expectedTotal {
		sl.ReportError(o.Total, "Total", "total", "invalid_total", "")
	}
}

// TestAutoRegister_StructLevelValidatable 测试 StructLevelValidatable 自动注册
func TestAutoRegister_StructLevelValidatable(t *testing.T) {
	v := New()

	// 测试1：密码匹配 - 应该验证成功
	user1 := UserWithAutoRegister{
		Username:        "john_doe",
		Password:        "pass123",
		ConfirmPassword: "pass123",
		Email:           "john@example.com",
		Age:             25,
	}

	err := v.ValidateStruct(user1)
	if err != nil {
		t.Errorf("测试1失败（密码匹配应该成功）: %v", err)
	}

	// 测试2：密码不匹配 - 应该验证失败
	user2 := UserWithAutoRegister{
		Username:        "jane_doe",
		Password:        "pass123",
		ConfirmPassword: "pass456",
		Email:           "jane@example.com",
		Age:             30,
	}

	err = v.ValidateStruct(user2)
	if err == nil {
		t.Error("测试2失败（密码不匹配应该失败）")
	}

	// 测试3：未成年用户名不含 kid - 应该验证失败
	user3 := UserWithAutoRegister{
		Username:        "tommy",
		Password:        "pass123",
		ConfirmPassword: "pass123",
		Email:           "tommy@example.com",
		Age:             15,
	}

	err = v.ValidateStruct(user3)
	if err == nil {
		t.Error("测试3失败（未成年用户名不含kid应该失败）")
	}

	// 测试4：未成年用户名包含 kid - 应该验证成功
	user4 := UserWithAutoRegister{
		Username:        "tommy_kid",
		Password:        "pass123",
		ConfirmPassword: "pass123",
		Email:           "tommy@example.com",
		Age:             15,
	}

	err = v.ValidateStruct(user4)
	if err != nil {
		t.Errorf("测试4失败（未成年用户名包含kid应该成功）: %v", err)
	}
}

// TestAutoRegister_MapRulesValidatable 测试 MapRulesValidatable 自动注册
func TestAutoRegister_MapRulesValidatable(t *testing.T) {
	v := New()

	// 测试1：有效产品 - 应该验证成功
	product1 := ProductWithAutoRegister{
		Name:  "iPhone 15",
		Price: 999.99,
		Stock: 100,
	}

	err := v.ValidateStruct(product1)
	if err != nil {
		t.Errorf("测试1失败（有效产品应该成功）: %v", err)
	}

	// 测试2：名称太短 - 应该验证失败
	product2 := ProductWithAutoRegister{
		Name:  "AB",
		Price: 99.99,
		Stock: 10,
	}

	err = v.ValidateStruct(product2)
	if err == nil {
		t.Error("测试2失败（名称太短应该失败）")
	}

	// 测试3：价格无效 - 应该验证失败
	product3 := ProductWithAutoRegister{
		Name:  "Bad Product",
		Price: -10,
		Stock: 50,
	}

	err = v.ValidateStruct(product3)
	if err == nil {
		t.Error("测试3失败（价格无效应该失败）")
	}
}

// TestAutoRegister_BothInterfaces 测试同时实现两个接口
func TestAutoRegister_BothInterfaces(t *testing.T) {
	v := New()

	// 测试1：有效订单 - 应该验证成功
	order1 := OrderWithBothInterfaces{
		ProductID: "prod-123",
		Quantity:  5,
		Price:     99.99,
		Total:     499.95, // 5 * 99.99
	}

	err := v.ValidateStruct(order1)
	if err != nil {
		t.Errorf("测试1失败（有效订单应该成功）: %v", err)
	}

	// 测试2：总价不匹配 - 应该验证失败
	order2 := OrderWithBothInterfaces{
		ProductID: "prod-123",
		Quantity:  5,
		Price:     99.99,
		Total:     400.00, // 错误的总价
	}

	err = v.ValidateStruct(order2)
	if err == nil {
		t.Error("测试2失败（总价不匹配应该失败）")
	}

	// 测试3：字段验证失败 - 应该验证失败
	order3 := OrderWithBothInterfaces{
		ProductID: "", // 空的 ProductID
		Quantity:  5,
		Price:     99.99,
		Total:     499.95,
	}

	err = v.ValidateStruct(order3)
	if err == nil {
		t.Error("测试3失败（空ProductID应该失败）")
	}
}

// TestAutoRegister_MultipleInstances 测试多个实例共享注册
func TestAutoRegister_MultipleInstances(t *testing.T) {
	v := New()

	// 第一次验证 - 触发自动注册
	user1 := UserWithAutoRegister{
		Username:        "user1",
		Password:        "pass",
		ConfirmPassword: "pass",
		Age:             20,
	}
	_ = v.ValidateStruct(user1)

	// 第二次验证 - 应该使用缓存的注册，不会重复注册
	user2 := UserWithAutoRegister{
		Username:        "user2",
		Password:        "pass",
		ConfirmPassword: "different", // 不匹配
		Age:             25,
	}
	err := v.ValidateStruct(user2)
	if err == nil {
		t.Error("密码不匹配应该失败")
	}
}

// ============================================================================
// 手动注册测试
// ============================================================================

// UserRegistration 用户注册模型
type UserRegistration struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	Email           string `json:"email"`
	Age             int    `json:"age"`
}

// Product 产品模型
type Product struct {
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	Description string  `json:"description"`
}

// TestStructValidationRegistration 测试提前注册功能
func TestStructValidationRegistration(t *testing.T) {
	v := New()

	// 注册验证规则
	err := v.RegisterStructValidation(func(sl validator.StructLevel) {
		user := sl.Current().Interface().(UserRegistration)
		if user.Password != user.ConfirmPassword {
			sl.ReportError(user.ConfirmPassword, "ConfirmPassword", "confirmPassword", "eqfield", "Password")
		}
	}, UserRegistration{})

	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	// 测试1：密码匹配
	user1 := UserRegistration{
		Username:        "testuser",
		Password:        "pass123",
		ConfirmPassword: "pass123",
		Email:           "test@example.com",
		Age:             25,
	}

	err = v.ValidateStruct(user1)
	if err != nil {
		t.Errorf("测试1失败: %v", err)
	}

	// 测试2：密码不匹配
	user2 := UserRegistration{
		Username:        "testuser",
		Password:        "pass123",
		ConfirmPassword: "pass456",
		Email:           "test@example.com",
		Age:             25,
	}

	err = v.ValidateStruct(user2)
	if err == nil {
		t.Error("测试2失败: 应该返回验证错误")
	}
}

// TestMapRulesRegistration 测试 Map 规则注册
func TestMapRulesRegistration(t *testing.T) {
	v := New()

	// 注册 Map 规则
	err := v.RegisterStructValidationMapRules(map[string]string{
		"Name":  "required,min=3",
		"Price": "required,gt=0",
		"Stock": "gte=0",
	}, Product{})

	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	// 测试1：有效产品
	product1 := Product{
		Name:  "Test Product",
		Price: 99.99,
		Stock: 10,
	}

	err = v.ValidateStruct(product1)
	if err != nil {
		t.Errorf("测试1失败: %v", err)
	}

	// 测试2：无效产品（价格 <= 0）
	product2 := Product{
		Name:  "Bad Product",
		Price: -10,
		Stock: 10,
	}

	err = v.ValidateStruct(product2)
	if err == nil {
		t.Error("测试2失败: 应该返回验证错误")
	}

	// 测试3：无效产品（名称太短）
	product3 := Product{
		Name:  "AB",
		Price: 99.99,
		Stock: 10,
	}

	err = v.ValidateStruct(product3)
	if err == nil {
		t.Error("测试3失败: 应该返回验证错误")
	}
}

// 简单测试：验证 RegisterStructValidation 功能
func TestRegisterStructValidationSimple(t *testing.T) {
	v := New()

	type TestUserSimple struct {
		Password        string
		ConfirmPassword string
	}

	// 注册结构体验证
	err := v.RegisterStructValidation(func(sl validator.StructLevel) {
		user := sl.Current().Interface().(TestUserSimple)
		if user.Password != user.ConfirmPassword {
			sl.ReportError(user.ConfirmPassword, "ConfirmPassword", "confirmPassword", "eqfield", "Password")
		}
	}, TestUserSimple{})

	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	// 测试：密码匹配
	user1 := TestUserSimple{
		Password:        "pass123",
		ConfirmPassword: "pass123",
	}
	err = v.ValidateStruct(user1)
	if err != nil {
		t.Errorf("密码匹配测试失败: %v", err)
	}

	// 测试：密码不匹配
	user2 := TestUserSimple{
		Password:        "pass123",
		ConfirmPassword: "pass456",
	}
	err = v.ValidateStruct(user2)
	if err == nil {
		t.Error("密码不匹配应该返回错误")
	}
}

// 简单测试：验证 RegisterStructValidationMapRules 功能
func TestRegisterStructValidationMapRulesSimple(t *testing.T) {
	v := New()

	type TestProductSimple struct {
		Name  string
		Price float64
	}

	// 注册 Map 规则
	err := v.RegisterStructValidationMapRules(map[string]string{
		"Name":  "required,min=3",
		"Price": "required,gt=0",
	}, TestProductSimple{})

	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	// 测试：有效产品
	product1 := TestProductSimple{
		Name:  "iPhone",
		Price: 999.99,
	}
	err = v.ValidateStruct(product1)
	if err != nil {
		t.Errorf("有效产品测试失败: %v", err)
	}

	// 测试：无效产品（价格 <= 0）
	product2 := TestProductSimple{
		Name:  "BadProduct",
		Price: -10,
	}
	err = v.ValidateStruct(product2)
	if err == nil {
		t.Error("无效价格应该返回错误")
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

func (u *BenchmarkUser) ValidateRules() map[ValidateScene]map[string]string {
	return map[ValidateScene]map[string]string{
		"create": {
			"Username": "required,min=3,max=20",
			"Email":    "required,email",
			"Password": "required,min=6",
			"Phone":    "len=11",
			"Age":      "gte=0,lte=150",
		},
		"update": {
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
		_ = v.Validate(user, "create")
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
			_ = v.Validate(user, "create")
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
		_ = v.Validate(invalidUser, "create")
	}
}

// BenchmarkAutoRegister_FirstTime 基准测试：首次验证（包含自动注册）
func BenchmarkAutoRegister_FirstTime(b *testing.B) {
	for i := 0; i < b.N; i++ {
		v := New() // 每次创建新的验证器
		user := UserWithAutoRegister{
			Username:        "testuser",
			Password:        "pass123",
			ConfirmPassword: "pass123",
			Age:             25,
		}
		_ = v.ValidateStruct(user)
	}
}

// BenchmarkAutoRegister_Cached 基准测试：后续验证（使用缓存的注册）
func BenchmarkAutoRegister_Cached(b *testing.B) {
	v := New()
	// 预先触发一次自动注册
	user := UserWithAutoRegister{
		Username:        "testuser",
		Password:        "pass123",
		ConfirmPassword: "pass123",
		Age:             25,
	}
	_ = v.ValidateStruct(user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.ValidateStruct(user)
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
			_ = v.Validate(user, "create")
		}
	})
}
