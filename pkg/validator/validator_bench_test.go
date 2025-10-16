package validator

import (
	"testing"
)

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

// BenchmarkMapValidator_AllowedKeys 测试 Map 验证器的允许键缓存性能
func BenchmarkMapValidator_AllowedKeys(b *testing.B) {
	mv := NewMapValidator().
		WithAllowedKeys("key1", "key2", "key3", "key4", "key5")

	data := map[string]any{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mv.Validate(data)
	}
}

// BenchmarkMapValidator_RequiredKeys 测试必填键验证性能
func BenchmarkMapValidator_RequiredKeys(b *testing.B) {
	mv := NewMapValidator().
		WithRequiredKeys("name", "email", "age")

	data := map[string]any{
		"name":  "Test User",
		"email": "test@example.com",
		"age":   25,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mv.Validate(data)
	}
}

// BenchmarkMapValidator_ComplexValidation 测试复杂 Map 验证场景
func BenchmarkMapValidator_ComplexValidation(b *testing.B) {
	mv := NewMapValidator().
		WithRequiredKeys("name", "price").
		WithAllowedKeys("name", "price", "brand", "warranty", "stock").
		WithKeyValidator("price", func(value interface{}) error {
			if v, ok := value.(float64); ok && v > 0 {
				return nil
			}
			return nil
		})

	data := map[string]any{
		"name":     "Product",
		"price":    99.99,
		"brand":    "BrandName",
		"warranty": 12,
		"stock":    100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mv.Validate(data)
	}
}

// BenchmarkValidateMapStringKey 测试字符串键验证性能
func BenchmarkValidateMapStringKey(b *testing.B) {
	data := map[string]any{
		"name": "TestProduct",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateMapStringKey(data, "name", 3, 50)
	}
}

// BenchmarkValidateMapIntKey 测试整数键验证性能
func BenchmarkValidateMapIntKey(b *testing.B) {
	data := map[string]any{
		"age": 25,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateMapIntKey(data, "age", 0, 120)
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

// BenchmarkMapValidator_Parallel 测试并发 Map 验证性能
func BenchmarkMapValidator_Parallel(b *testing.B) {
	mv := NewMapValidator().
		WithRequiredKeys("name", "price").
		WithAllowedKeys("name", "price", "brand")

	data := map[string]any{
		"name":  "Product",
		"price": 99.99,
		"brand": "BrandName",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = mv.Validate(data)
		}
	})
}
