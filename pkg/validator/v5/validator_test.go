package v5

import (
	"errors"
	error2 "katydid-common-account/pkg/validator/v5/error"
	"testing"
)

// ============================================================================
// 测试用例
// ============================================================================

const (
	SceneCreate Scene = 1 << iota
	SceneUpdate
	SceneDelete
	SceneQuery
	SceneImport
	SceneExport
)

// 测试用的模型
type TestUser struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Age      int    `json:"age"`
}

// 实现 RuleValidation 接口
func (u *TestUser) ValidateRules() map[Scene]map[string]string {
	return map[Scene]map[string]string{
		SceneCreate: {
			"Username": "required,min=3,max=20",
			"Email":    "required,email",
			"Password": "required,min=6",
			"Age":      "required,gte=18",
		},
		SceneUpdate: {
			"Username": "omitempty,min=3,max=20",
			"Email":    "omitempty,email",
			"Password": "omitempty,min=6",
			"Age":      "omitempty,gte=18",
		},
	}
}

// 实现 BusinessValidation 接口
func (u *TestUser) ValidateBusiness(scene Scene, ctx *ValidationContext) error {
	// 业务规则：用户名不能是 admin
	if u.Username == "admin" {
		ctx.AddError(error2.NewFieldError("TestUser.Username", "Username").
			WithMessage("username 'admin' is reserved"))
	}

	// 业务规则：年龄必须在合理范围内
	if u.Age > 150 {
		ctx.AddError(error2.NewFieldError("TestUser.Age", "Age").
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

		var ve *error2.ValidationError
		ok := errors.As(err, &ve)
		if !ok {
			t.Error("expected ValidationError")
		}

		if len(ve.errors) == 0 {
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

		var ve *error2.ValidationError
		ok := errors.As(err, &ve)
		if !ok {
			t.Error("expected ValidationError")
		}

		found := false
		for _, e := range ve.errors {
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

//// TestValidatorBuilder 测试构建器
//func TestValidatorBuilder(t *testing.T) {
//	validator := NewValidatorBuilder().
//		WithRuleStrategy().
//		WithBusinessStrategy().
//		WithMaxDepth(50).
//		WithMaxErrors(500).
//		Build()
//
//	user := &TestUser{
//		Username: "john",
//		Email:    "invalid-email",
//		Password: "123",
//		Age:      15,
//	}
//
//	err := validator.Validate(user, SceneCreate)
//	if err == nil {
//		t.Error("expected error, got nil")
//	}
//
//	ve, ok := err.(*ValidationError)
//	if !ok {
//		t.Error("expected ValidationError")
//	}
//
//	if len(ve.errors) == 0 {
//		t.Error("expected errors")
//	}
//
//	t.Logf("got %d errors", len(ve.errors))
//	for _, e := range ve.errors {
//		t.Logf("error: %s", e.Error())
//	}
//}

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

const SceneTest Scene = 1 << 0

// Base 基础结构体（嵌入使用）
type Base struct {
	ID int `json:"id" validate:"required,gte=1"`
}

// User 嵌入 Base
type User struct {
	Base
	Email string `json:"email" validate:"required,email"`
}

// 测试嵌套验证的深度传递和上下文使用
func TestNestedValidation_ContextUsage(t *testing.T) {
	factory := NewValidatorFactory()
	validator := factory.CreateDefault()

	// 测试：嵌套字段验证应该生效
	t.Run("nested field validation with correct context", func(t *testing.T) {
		user := &User{
			Base:  Base{ID: 0}, // 违反规则：应该 >= 1
			Email: "test@example.com",
		}

		err := validator.Validate(user, SceneTest)
		if err == nil {
			t.Error("expected validation error for nested field ID, got nil")
		} else {
			t.Logf("Got expected error: %v", err)
			var ve *error2.ValidationError
			if errors.As(err, &ve) {
				t.Logf("Error count: %d", len(ve.errors))
				for _, e := range ve.errors {
					t.Logf("  Field: %s, Tag: %s", e.Namespace, e.Tag)
				}
			}
		}
	})

	// 测试：正确的数据应该通过
	t.Run("valid nested data", func(t *testing.T) {
		user := &User{
			Base:  Base{ID: 1},
			Email: "test@example.com",
		}

		err := validator.Validate(user, SceneTest)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		} else {
			t.Log("Validation passed as expected")
		}
	})
}

// 性能测试用的类型定义
const SceneBench Scene = 1

type BenchTestStruct struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func (t *BenchTestStruct) ValidateRules() map[Scene]map[string]string {
	return map[Scene]map[string]string{
		SceneBench: {
			"Type":  "required,min=3,max=20",
			"Email": "required,email",
			"Age":   "required,gte=18,lte=100",
		},
	}
}

type BenchUser struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Age      int    `json:"age"`
}

func (u *BenchUser) ValidateRules() map[Scene]map[string]string {
	return map[Scene]map[string]string{
		SceneBench: {
			"Username": "required,min=3,max=20",
			"Email":    "required,email",
			"Password": "required,min=6",
			"Age":      "required,gte=18",
		},
	}
}

// BenchmarkValidation 测试验证器性能
// 验证 registerStructValidator 的缓存优化效果
func BenchmarkValidation(b *testing.B) {
	factory := NewValidatorFactory()
	validator := factory.CreateDefault()

	data := &BenchTestStruct{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   25,
	}

	// 预热：触发类型注册
	_ = validator.Validate(data, SceneBench)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = validator.Validate(data, SceneBench)
		}
	})
}

// BenchmarkValidationWithCache 测试缓存的效果
func BenchmarkValidationWithCache(b *testing.B) {
	factory := NewValidatorFactory()
	validator := factory.CreateDefault()

	user := &BenchUser{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Age:      25,
	}

	// 第一次调用会注册类型
	b.Run("first_call", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// 每次创建新验证器（模拟冷启动）
			factory := NewValidatorFactory()
			validator := factory.CreateDefault()
			_ = validator.Validate(user, SceneBench)
		}
	})

	// 后续调用会命中缓存
	b.Run("cached_calls", func(b *testing.B) {
		// 预热
		_ = validator.Validate(user, SceneBench)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.Validate(user, SceneBench)
		}
	})
}
