package v2

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// 测试模型定义
// ============================================================================

// User 用户模型 - 演示如何实现接口
type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Age      int    `json:"age"`
	Phone    string `json:"phone"`
}

// GetRules 实现 RuleProvider 接口
func (u *User) GetRules(scene Scene) map[string]string {
	rules := make(map[string]string)

	switch {
	case scene.Has(SceneCreate):
		rules["Username"] = "required,min=3,max=20,alphanum"
		rules["Email"] = "required,email"
		rules["Password"] = "required,min=6,max=20"
		rules["Age"] = "omitempty,gte=0,lte=150"
		rules["Phone"] = "omitempty,len=11,numeric"

	case scene.Has(SceneUpdate):
		rules["Username"] = "omitempty,min=3,max=20,alphanum"
		rules["Email"] = "omitempty,email"
		rules["Password"] = "omitempty,min=6,max=20"
		rules["Age"] = "omitempty,gte=0,lte=150"
		rules["Phone"] = "omitempty,len=11,numeric"

	case scene.Has(SceneQuery):
		rules["ID"] = "omitempty,gt=0"
		rules["Username"] = "omitempty,min=1"
		rules["Email"] = "omitempty,email"
	}

	return rules
}

// CustomValidate 实现 CustomValidator 接口
func (u *User) CustomValidate(scene Scene, collector ErrorCollector) {
	if scene.Has(SceneCreate) {
		// 创建时，用户名不能是保留字
		if u.Username == "admin" || u.Username == "root" {
			collector.AddError("Username", "reserved", u.Username, "用户名 '"+u.Username+"' 是保留字，不能使用")
		}
	}

	// 跨字段验证：如果提供了密码，长度必须足够
	if u.Password != "" && len(u.Password) < 6 {
		collector.AddError("Password", "min_length", "6", "密码长度至少6位")
	}
}

// GetErrorMessage 实现 ErrorMessageProvider 接口
func (u *User) GetErrorMessage(field, tag, param string) string {
	messages := map[string]map[string]string{
		"Username": {
			"required": "用户名不能为空",
			"min":      "用户名长度不能少于3个字符",
			"max":      "用户名长度不能超过20个字符",
			"alphanum": "用户名只能包含字母和数字",
			"reserved": "用户名 '" + param + "' 是保留字",
		},
		"Email": {
			"required": "邮箱地址不能为空",
			"email":    "请输入有效的邮箱地址",
		},
		"Password": {
			"required":   "密码不能为空",
			"min":        "密码长度不能少于6位",
			"min_length": "密码长度至少" + param + "位",
		},
		"Phone": {
			"len":     "手机号码必须是11位数字",
			"numeric": "手机号码只能包含数字",
		},
	}

	if fieldMsgs, ok := messages[field]; ok {
		if msg, ok := fieldMsgs[tag]; ok {
			return msg
		}
	}

	return "" // 返回空字符串使用默认消息
}

// ============================================================================
// 基础验证测试
// ============================================================================

func TestValidator_Create(t *testing.T) {
	validator, err := NewDefaultValidator()
	if err != nil {
		t.Fatalf("创建验证器失败: %v", err)
	}

	tests := []struct {
		name    string
		user    *User
		wantErr bool
	}{
		{
			name: "有效的创建数据",
			user: &User{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
				Age:      25,
			},
			wantErr: false,
		},
		{
			name: "缺少必填字段",
			user: &User{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "无效的邮箱",
			user: &User{
				Username: "testuser",
				Email:    "invalid-email",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "用户名是保留字",
			user: &User{
				Username: "admin",
				Email:    "admin@example.com",
				Password: "password123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.user, SceneCreate)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("验证错误: %v", err)
			}
		})
	}
}

func TestValidator_Update(t *testing.T) {
	validator, err := NewDefaultValidator()
	if err != nil {
		t.Fatalf("创建验证器失败: %v", err)
	}

	tests := []struct {
		name    string
		user    *User
		wantErr bool
	}{
		{
			name: "更新用户名",
			user: &User{
				ID:       1,
				Username: "newname",
			},
			wantErr: false,
		},
		{
			name: "更新邮箱",
			user: &User{
				ID:    1,
				Email: "newemail@example.com",
			},
			wantErr: false,
		},
		{
			name: "空数据（更新时所有字段都是可选的）",
			user: &User{
				ID: 1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.user, SceneUpdate)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ============================================================================
// 部分字段验证测试
// ============================================================================

func TestValidator_Partial(t *testing.T) {
	validator, err := NewDefaultValidator()
	if err != nil {
		t.Fatalf("创建验证器失败: %v", err)
	}

	user := &User{
		Username: "ab", // 太短
		Email:    "valid@example.com",
		Password: "password123",
	}

	// 只验证Email字段（应该通过）
	err = validator.ValidatePartial(user, "Email")
	if err != nil {
		t.Errorf("部分验证失败: %v", err)
	}

	// 验证Username字段（应该失败）
	err = validator.ValidatePartial(user, "Username")
	t.Logf("验证 Username 结果: %v", err)
	if err == nil {
		t.Error("期望验证失败，但通过了")
	} else {
		t.Logf("验证正确失败: %v", err)
	}
}

// ============================================================================
// 策略模式测试
// ============================================================================

func TestValidator_WithStrategy(t *testing.T) {
	// 测试快速失败策略
	validator, err := NewFailFastValidator()
	if err != nil {
		t.Fatalf("创建验证器失败: %v", err)
	}

	user := &User{
		Username: "",        // 错误1
		Email:    "invalid", // 错误2
		Password: "123",     // 错误3
	}

	err = validator.Validate(user, SceneCreate)
	if err == nil {
		t.Error("期望验证失败，但通过了")
	}

	// 快速失败模式应该只返回第一个错误
	if verrs, ok := err.(ValidationErrors); ok {
		t.Logf("错误数量: %d, 错误: %v", len(verrs), verrs)
	}
}

// ============================================================================
// 建造者模式测试
// ============================================================================

func TestValidatorBuilder(t *testing.T) {
	validator, err := NewValidatorBuilder().
		WithCache(NewCacheManager()).
		WithPool(NewValidatorPool()).
		WithStrategy(NewDefaultStrategy()).
		RegisterCustomValidation("is_awesome", func(fl validator.FieldLevel) bool {
			return fl.Field().String() == "awesome"
		}).
		Build()

	if err != nil {
		t.Fatalf("构建验证器失败: %v", err)
	}

	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	err = validator.Validate(user, SceneCreate)
	if err != nil {
		t.Errorf("验证失败: %v", err)
	}
}

// ============================================================================
// 缓存测试
// ============================================================================

func TestValidator_Cache(t *testing.T) {
	cache := NewCacheManager()

	validator, err := NewValidatorBuilder().
		WithCache(cache).
		Build()

	if err != nil {
		t.Fatalf("创建验证器失败: %v", err)
	}

	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	// 第一次验证（缓存miss）
	err = validator.Validate(user, SceneCreate)
	if err != nil {
		t.Errorf("验证失败: %v", err)
	}

	// 第二次验证（缓存hit）
	err = validator.Validate(user, SceneCreate)
	if err != nil {
		t.Errorf("验证失败: %v", err)
	}
}

// ============================================================================
// 全局验证器测试
// ============================================================================

func TestGlobalValidator(t *testing.T) {
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	err := Validate(user, SceneCreate)
	if err != nil {
		t.Errorf("全局验证失败: %v", err)
	}
}

// ============================================================================
// 错误收集器测试
// ============================================================================

func TestErrorCollector(t *testing.T) {
	collector := NewErrorCollector()

	collector.AddError("Username", "required")
	collector.AddError("Email", "email", "invalid")

	if !collector.HasErrors() {
		t.Error("期望有错误，但没有错误")
	}

	errs := collector.GetErrors()
	if len(errs) != 2 {
		t.Errorf("期望2个错误，得到 %d 个", len(errs))
	}

	collector.Clear()
	if collector.HasErrors() {
		t.Error("期望没有错误，但有错误")
	}
}

// ============================================================================
// 场景组合测试
// ============================================================================

func TestScene_Combination(t *testing.T) {
	// 测试场景组合
	scene := SceneCreate | SceneUpdate

	if !scene.Has(SceneCreate) {
		t.Error("场景应该包含Create")
	}

	if !scene.Has(SceneUpdate) {
		t.Error("场景应该包含Update")
	}

	// 测试场景字符串
	t.Logf("场景名称: %s", scene.String())
}

// ============================================================================
// 性能测试
// ============================================================================

func BenchmarkValidator_WithCache(b *testing.B) {
	validator, _ := NewDefaultValidator()

	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(user, SceneCreate)
	}
}

func BenchmarkValidator_WithoutCache(b *testing.B) {
	validator, _ := NewSimpleValidator()

	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(user, SceneCreate)
	}
}

func BenchmarkValidator_WithPool(b *testing.B) {
	validator, _ := NewValidatorBuilder().
		WithPool(NewValidatorPool()).
		Build()

	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(user, SceneCreate)
	}
}
