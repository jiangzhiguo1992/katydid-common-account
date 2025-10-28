package v6_test

import (
	"testing"

	"katydid-common-account/pkg/validator/v6"
	"katydid-common-account/pkg/validator/v6/core"
)

// 定义场景
const (
	SceneCreate core.Scene = 1 << iota
	SceneUpdate
)

// User 用户模型
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

// GetRules 实现 RuleProvider 接口
func (u *User) GetRules() map[core.Scene]map[string]string {
	return map[core.Scene]map[string]string{
		SceneCreate: {
			"name":  "required,min=2,max=50",
			"email": "required,email",
			"age":   "required,min=18,max=120",
		},
		SceneUpdate: {
			"name":  "omitempty,min=2,max=50",
			"email": "omitempty,email",
			"age":   "omitempty,min=18,max=120",
		},
	}
}

// ValidateBusiness 实现 BusinessValidator 接口
func (u *User) ValidateBusiness(scene core.Scene, ctx core.ValidationContext) error {
	// 示例：业务逻辑验证
	if scene == SceneCreate && u.Age < 18 {
		ctx.ErrorCollector().Add(
			core.NewFieldError("age", "age_limit").
				WithMessage("创建用户时年龄必须大于18岁"),
		)
	}
	return nil
}

// TestBasicValidation 测试基本验证
func TestBasicValidation(t *testing.T) {
	tests := []struct {
		name      string
		user      *User
		scene     core.Scene
		wantError bool
	}{
		{
			name: "有效的创建场景",
			user: &User{
				Name:  "张三",
				Email: "zhangsan@example.com",
				Age:   25,
			},
			scene:     SceneCreate,
			wantError: false,
		},
		{
			name: "缺少必填字段",
			user: &User{
				Name: "",
				Age:  25,
			},
			scene:     SceneCreate,
			wantError: true,
		},
		{
			name: "邮箱格式错误",
			user: &User{
				Name:  "张三",
				Email: "invalid-email",
				Age:   25,
			},
			scene:     SceneCreate,
			wantError: true,
		},
		{
			name: "年龄不符合要求",
			user: &User{
				Name:  "张三",
				Email: "zhangsan@example.com",
				Age:   15,
			},
			scene:     SceneCreate,
			wantError: true,
		},
		{
			name: "更新场景 - 部分字段",
			user: &User{
				Name: "李四",
			},
			scene:     SceneUpdate,
			wantError: false,
		},
	}

	// 创建验证器
	validator := v6.NewValidator().BuildDefault()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.user, tt.scene)
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestValidateWithRequest 测试使用请求对象验证
func TestValidateWithRequest(t *testing.T) {
	validator := v6.NewValidator().BuildDefault()

	user := &User{
		Name:  "张三",
		Email: "zhangsan@example.com",
		Age:   25,
	}

	// 只验证指定字段
	req := core.NewValidationRequest(user, SceneCreate).
		WithFields("name", "email")

	result, err := validator.ValidateWithRequest(req)
	if err != nil {
		t.Fatalf("ValidateWithRequest() error = %v", err)
	}

	if result.HasErrors() {
		t.Errorf("Expected no errors, got %v", result.Errors)
	}
}

// TestExcludeFields 测试排除字段
func TestExcludeFields(t *testing.T) {
	validator := v6.NewValidator().BuildDefault()

	user := &User{
		Name: "张三",
		Age:  25,
		// 故意不设置 Email
	}

	// 排除 email 字段验证
	req := core.NewValidationRequest(user, SceneCreate).
		WithExcludeFields("email")

	result, err := validator.ValidateWithRequest(req)
	if err != nil {
		t.Fatalf("ValidateWithRequest() error = %v", err)
	}

	if result.HasErrors() {
		t.Errorf("Expected no errors when email is excluded, got %v", result.Errors)
	}
}

// TestGlobalValidator 测试全局验证器
func TestGlobalValidator(t *testing.T) {
	user := &User{
		Name:  "张三",
		Email: "zhangsan@example.com",
		Age:   25,
	}

	err := v6.Validate(user, SceneCreate)
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

// BenchmarkValidation 基准测试
func BenchmarkValidation(b *testing.B) {
	validator := v6.NewValidator().BuildDefault()

	user := &User{
		Name:  "张三",
		Email: "zhangsan@example.com",
		Age:   25,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(user, SceneCreate)
	}
}
