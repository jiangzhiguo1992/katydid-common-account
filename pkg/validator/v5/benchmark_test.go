package v5

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// 性能基准测试
// ============================================================================

// BenchmarkUser 测试用户结构
type BenchmarkUser struct {
	Username string `json:"username" validate:"required,min=3,max=20"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Age      int    `json:"age" validate:"required,gte=18,lte=120"`
	Phone    string `json:"phone" validate:"omitempty,e164"`
}

func (u *BenchmarkUser) ValidateRules() map[Scene]map[string]string {
	return map[Scene]map[string]string{
		SceneCreate: {
			"Username": "required,min=3,max=20",
			"Email":    "required,email",
			"Password": "required,min=6",
			"Age":      "required,gte=18,lte=120",
			"Phone":    "omitempty,e164",
		},
	}
}

// 使用测试文件中已定义的场景常量

// ============================================================================
// v5 验证器基准测试
// ============================================================================

// Benchmark_V5_Validate_Simple 简单验证
func Benchmark_V5_Validate_Simple(b *testing.B) {
	factory := NewValidatorFactory()
	validator := factory.CreateDefault()

	user := &BenchmarkUser{
		Username: "johndoe",
		Email:    "john@example.com",
		Password: "password123",
		Age:      25,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = validator.Validate(user, SceneCreate)
	}
}

// Benchmark_V5_Validate_WithError 含错误的验证
func Benchmark_V5_Validate_WithError(b *testing.B) {
	factory := NewValidatorFactory()
	validator := factory.CreateDefault()

	user := &BenchmarkUser{
		Username: "jo", // 太短，会失败
		Email:    "invalid-email",
		Password: "123", // 太短
		Age:      15,    // 未成年
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = validator.Validate(user, SceneCreate)
	}
}

// Benchmark_V5_Validate_Parallel 并发验证
func Benchmark_V5_Validate_Parallel(b *testing.B) {
	factory := NewValidatorFactory()
	validator := factory.CreateDefault()

	user := &BenchmarkUser{
		Username: "johndoe",
		Email:    "john@example.com",
		Password: "password123",
		Age:      25,
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = validator.Validate(user, SceneCreate)
		}
	})
}

// ============================================================================
// go-playground/validator 直接使用基准测试（对照组）
// ============================================================================

// Benchmark_GoPlayground_Validate_Simple 使用 go-playground/validator
func Benchmark_GoPlayground_Validate_Simple(b *testing.B) {
	validate := validator.New()

	user := &BenchmarkUser{
		Username: "johndoe",
		Email:    "john@example.com",
		Password: "password123",
		Age:      25,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = validate.Struct(user)
	}
}

// Benchmark_GoPlayground_Validate_WithError 含错误的验证
func Benchmark_GoPlayground_Validate_WithError(b *testing.B) {
	validate := validator.New()

	user := &BenchmarkUser{
		Username: "jo",
		Email:    "invalid-email",
		Password: "123",
		Age:      15,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = validate.Struct(user)
	}
}

// Benchmark_GoPlayground_Validate_Parallel 并发验证
func Benchmark_GoPlayground_Validate_Parallel(b *testing.B) {
	validate := validator.New()

	user := &BenchmarkUser{
		Username: "johndoe",
		Email:    "john@example.com",
		Password: "password123",
		Age:      25,
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = validate.Struct(user)
		}
	})
}

// ============================================================================
// 复杂场景基准测试
// ============================================================================

// ComplexUser 复杂用户结构（多层嵌套）
type ComplexUser struct {
	Username string        `json:"username" validate:"required,min=3,max=20"`
	Email    string        `json:"email" validate:"required,email"`
	Profile  *UserProfile  `json:"profile" validate:"required"`
	Address  []UserAddress `json:"address" validate:"required,dive"`
	Settings *UserSettings `json:"settings" validate:"omitempty"`
}

type UserProfile struct {
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Bio       string `json:"bio" validate:"max=500"`
}

type UserAddress struct {
	Street  string `json:"street" validate:"required"`
	City    string `json:"city" validate:"required"`
	ZipCode string `json:"zip_code" validate:"required,len=5"`
}

type UserSettings struct {
	Theme    string `json:"theme" validate:"oneof=light dark"`
	Language string `json:"language" validate:"required,len=2"`
}

func (u *ComplexUser) ValidateRules() map[Scene]map[string]string {
	return map[Scene]map[string]string{
		SceneCreate: {
			"Username": "required,min=3,max=20",
			"Email":    "required,email",
			"Profile":  "required",
			"Address":  "required,dive",
		},
	}
}

// Benchmark_V5_Validate_Complex 复杂结构验证
func Benchmark_V5_Validate_Complex(b *testing.B) {
	factory := NewValidatorFactory()
	validator := factory.CreateDefault()

	user := &ComplexUser{
		Username: "johndoe",
		Email:    "john@example.com",
		Profile: &UserProfile{
			FirstName: "John",
			LastName:  "Doe",
			Bio:       "Software Engineer",
		},
		Address: []UserAddress{
			{
				Street:  "123 Main St",
				City:    "New York",
				ZipCode: "10001",
			},
		},
		Settings: &UserSettings{
			Theme:    "dark",
			Language: "en",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = validator.Validate(user, SceneCreate)
	}
}

// Benchmark_GoPlayground_Validate_Complex 复杂结构验证（对照）
func Benchmark_GoPlayground_Validate_Complex(b *testing.B) {
	validate := validator.New()

	user := &ComplexUser{
		Username: "johndoe",
		Email:    "john@example.com",
		Profile: &UserProfile{
			FirstName: "John",
			LastName:  "Doe",
			Bio:       "Software Engineer",
		},
		Address: []UserAddress{
			{
				Street:  "123 Main St",
				City:    "New York",
				ZipCode: "10001",
			},
		},
		Settings: &UserSettings{
			Theme:    "dark",
			Language: "en",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = validate.Struct(user)
	}
}

// ============================================================================
// 内存分配测试
// ============================================================================

// Benchmark_V5_Memory_Allocation 内存分配测试
func Benchmark_V5_Memory_Allocation(b *testing.B) {
	factory := NewValidatorFactory()
	validator := factory.CreateDefault()

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		user := &BenchmarkUser{
			Username: "johndoe",
			Email:    "john@example.com",
			Password: "password123",
			Age:      25,
		}
		_ = validator.Validate(user, SceneCreate)
	}
}

// Benchmark_GoPlayground_Memory_Allocation 内存分配测试（对照）
func Benchmark_GoPlayground_Memory_Allocation(b *testing.B) {
	validate := validator.New()

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		user := &BenchmarkUser{
			Username: "johndoe",
			Email:    "john@example.com",
			Password: "password123",
			Age:      25,
		}
		_ = validate.Struct(user)
	}
}

// ============================================================================
// 缓存效果测试
// ============================================================================

// Benchmark_V5_WithCache 使用缓存
func Benchmark_V5_WithCache(b *testing.B) {
	factory := NewValidatorFactory()
	validator := factory.CreateDefault()

	// 预热缓存
	user := &BenchmarkUser{
		Username: "johndoe",
		Email:    "john@example.com",
		Password: "password123",
		Age:      25,
	}
	_ = validator.Validate(user, SceneCreate)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = validator.Validate(user, SceneCreate)
	}
}

// Benchmark_V5_ClearCache 清除缓存后验证
func Benchmark_V5_ClearCache(b *testing.B) {
	factory := NewValidatorFactory()
	validator := factory.CreateDefault()

	user := &BenchmarkUser{
		Username: "johndoe",
		Email:    "john@example.com",
		Password: "password123",
		Age:      25,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		validator.ClearCache() // 每次清除缓存
		_ = validator.Validate(user, SceneCreate)
	}
}
