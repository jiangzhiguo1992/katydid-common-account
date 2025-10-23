package v5

import (
	"testing"
)

// ============================================================================
// 测试用例
// ============================================================================

// 测试用的模型
type TestUser struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Age      int    `json:"age"`
}

// 实现 RuleProvider 接口
func (u *TestUser) GetRules(scene Scene) map[string]string {
	switch scene {
	case SceneCreate:
		return map[string]string{
			"Username": "required,min=3,max=20",
			"Email":    "required,email",
			"Password": "required,min=6",
			"Age":      "required,gte=18",
		}
	case SceneUpdate:
		return map[string]string{
			"Username": "omitempty,min=3,max=20",
			"Email":    "omitempty,email",
			"Password": "omitempty,min=6",
			"Age":      "omitempty,gte=18",
		}
	default:
		return nil
	}
}

// 实现 BusinessValidator 接口
func (u *TestUser) ValidateBusiness(ctx *ValidationContext) error {
	// 业务规则：用户名不能是 admin
	if u.Username == "admin" {
		ctx.AddError(NewFieldError("TestUser.Username", "Username", "reserved").
			WithMessage("username 'admin' is reserved"))
	}

	// 业务规则：年龄必须在合理范围内
	if u.Age > 150 {
		ctx.AddError(NewFieldError("TestUser.Age", "Age", "invalid_age").
			WithMessage("age is not reasonable"))
	}

	return nil
}

// TestValidatorEngine_Validate 测试基本验证
func TestValidatorEngine_Validate(t *testing.T) {
	factory := NewValidatorFactory()
	validator := factory.CreateDefault()

	// 测试用例1：有效数据
	t.Run("valid data", func(t *testing.T) {
		user := &TestUser{
			Username: "john",
			Email:    "john@example.com",
			Password: "password123",
			Age:      25,
		}

		err := validator.Validate(user, SceneCreate)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	// 测试用例2：缺少必填字段
	t.Run("missing required fields", func(t *testing.T) {
		user := &TestUser{
			Username: "john",
		}

		err := validator.Validate(user, SceneCreate)
		if err == nil {
			t.Error("expected error, got nil")
		}

		ve, ok := err.(*ValidationError)
		if !ok {
			t.Error("expected ValidationError")
		}

		if len(ve.Errors) == 0 {
			t.Error("expected errors")
		}
	})

	// 测试用例3：业务验证失败
	t.Run("business validation failed", func(t *testing.T) {
		user := &TestUser{
			Username: "admin",
			Email:    "admin@example.com",
			Password: "password123",
			Age:      25,
		}

		err := validator.Validate(user, SceneCreate)
		if err == nil {
			t.Error("expected error, got nil")
		}

		ve, ok := err.(*ValidationError)
		if !ok {
			t.Error("expected ValidationError")
		}

		found := false
		for _, e := range ve.Errors {
			if e.Tag == "reserved" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected reserved username error")
		}
	})

	// 测试用例4：更新场景（字段可选）
	t.Run("update scene", func(t *testing.T) {
		user := &TestUser{
			Username: "john",
		}

		err := validator.Validate(user, SceneUpdate)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

// TestValidatorBuilder 测试构建器
func TestValidatorBuilder(t *testing.T) {
	validator := NewValidatorBuilder().
		WithRuleStrategy().
		WithBusinessStrategy().
		WithMaxDepth(50).
		WithMaxErrors(500).
		Build()

	user := &TestUser{
		Username: "john",
		Email:    "invalid-email",
		Password: "123",
		Age:      15,
	}

	err := validator.Validate(user, SceneCreate)
	if err == nil {
		t.Error("expected error, got nil")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Error("expected ValidationError")
	}

	if len(ve.Errors) == 0 {
		t.Error("expected errors")
	}

	t.Logf("got %d errors", len(ve.Errors))
	for _, e := range ve.Errors {
		t.Logf("error: %s", e.Error())
	}
}

// TestValidateFields 测试部分字段验证
func TestValidateFields(t *testing.T) {
	validator := Default()

	user := &TestUser{
		Username: "ab", // 太短
		Email:    "valid@example.com",
		Password: "12345", // 太短
		Age:      25,
	}

	// 只验证 Email 字段
	err := validator.ValidateFields(user, SceneCreate, "Email")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 验证 Username 字段（应该失败）
	err = validator.ValidateFields(user, SceneCreate, "Username")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// BenchmarkValidate 性能测试
func BenchmarkValidate(b *testing.B) {
	validator := Default()

	user := &TestUser{
		Username: "john",
		Email:    "john@example.com",
		Password: "password123",
		Age:      25,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(user, SceneCreate)
	}
}

// BenchmarkValidateWithCache 带缓存的性能测试
func BenchmarkValidateWithCache(b *testing.B) {
	validator := Default()

	user := &TestUser{
		Username: "john",
		Email:    "john@example.com",
		Password: "password123",
		Age:      25,
	}

	// 预热缓存
	_ = validator.Validate(user, SceneCreate)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(user, SceneCreate)
	}
}

// BenchmarkValidateParallel 并发性能测试
func BenchmarkValidateParallel(b *testing.B) {
	validator := Default()

	b.RunParallel(func(pb *testing.PB) {
		user := &TestUser{
			Username: "john",
			Email:    "john@example.com",
			Password: "password123",
			Age:      25,
		}

		for pb.Next() {
			_ = validator.Validate(user, SceneCreate)
		}
	})
}
