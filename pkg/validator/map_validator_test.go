package validator

import (
	"fmt"
	"testing"
)

// ============================================================================
// Map 验证测试
// ============================================================================

// TestValidateMap 测试基础的 map 验证功能
func TestValidateMap(t *testing.T) {
	tests := []struct {
		name      string
		extras    map[string]any
		validator *MapValidator
		wantErr   bool
		errMsg    string
	}{
		{
			name: "所有验证都通过",
			extras: map[string]any{
				"required_key": "value",
				"optional_key": "value",
			},
			validator: &MapValidator{
				RequiredKeys: []string{"required_key"},
				AllowedKeys:  []string{"required_key", "optional_key"},
			},
			wantErr: false,
		},
		{
			name: "缺少必填键",
			extras: map[string]any{
				"optional_key": "value",
			},
			validator: &MapValidator{
				RequiredKeys: []string{"required_key"},
			},
			wantErr: true,
		},
		{
			name: "包含不允许的键",
			extras: map[string]any{
				"required_key": "value",
				"invalid_key":  "value",
			},
			validator: &MapValidator{
				RequiredKeys: []string{"required_key"},
				AllowedKeys:  []string{"required_key", "optional_key"},
			},
			wantErr: true,
		},
		{
			name: "自定义键验证失败",
			extras: map[string]any{
				"name": "ab", // 太短
			},
			validator: &MapValidator{
				KeyValidators: map[string]func(value interface{}) error{
					"name": func(value interface{}) error {
						str, ok := value.(string)
						if !ok || len(str) < 3 {
							return fmt.Errorf("name 长度必须至少3个字符")
						}
						return nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "自定义键验证通过",
			extras: map[string]any{
				"name": "alice",
			},
			validator: &MapValidator{
				KeyValidators: map[string]func(value interface{}) error{
					"name": func(value interface{}) error {
						str, ok := value.(string)
						if !ok || len(str) < 3 {
							return fmt.Errorf("name 长度必须至少3个字符")
						}
						return nil
					},
				},
			},
			wantErr: false,
		},
		{
			name:      "空验证器不报错",
			extras:    map[string]any{"any_key": "any_value"},
			validator: nil,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMap(tt.extras, tt.validator)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMap() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("错误信息: %v", err)
			}
		})
	}
}

// TestValidateMapMustHaveKeys 测试必填键验证
func TestValidateMapMustHaveKeys(t *testing.T) {
	tests := []struct {
		name    string
		extras  map[string]any
		keys    []string
		wantErr bool
	}{
		{
			name: "包含所有必填键",
			extras: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			keys:    []string{"key1", "key2"},
			wantErr: false,
		},
		{
			name: "缺少一个必填键",
			extras: map[string]any{
				"key1": "value1",
			},
			keys:    []string{"key1", "key2"},
			wantErr: true,
		},
		{
			name:    "缺少所有必填键",
			extras:  map[string]any{},
			keys:    []string{"key1", "key2"},
			wantErr: true,
		},
		{
			name: "空必填键列表",
			extras: map[string]any{
				"key1": "value1",
			},
			keys:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMapMustHaveKeys(tt.extras, tt.keys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMapMustHaveKeys() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("错误信息: %v", err)
			}
		})
	}
}

// TestValidateMapStringKey 测试字符串类型键验证
func TestValidateMapStringKey(t *testing.T) {
	tests := []struct {
		name    string
		extras  map[string]any
		key     string
		minLen  int
		maxLen  int
		wantErr bool
	}{
		{
			name:    "字符串长度在范围内",
			extras:  map[string]any{"name": "alice"},
			key:     "name",
			minLen:  3,
			maxLen:  10,
			wantErr: false,
		},
		{
			name:    "字符串太短",
			extras:  map[string]any{"name": "ab"},
			key:     "name",
			minLen:  3,
			maxLen:  10,
			wantErr: true,
		},
		{
			name:    "字符串太长",
			extras:  map[string]any{"name": "verylongname"},
			key:     "name",
			minLen:  3,
			maxLen:  10,
			wantErr: true,
		},
		{
			name:    "不是字符串类型",
			extras:  map[string]any{"name": 123},
			key:     "name",
			minLen:  3,
			maxLen:  10,
			wantErr: true,
		},
		{
			name:    "键不存在",
			extras:  map[string]any{},
			key:     "name",
			minLen:  3,
			maxLen:  10,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMapStringKey(tt.extras, tt.key, tt.minLen, tt.maxLen)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMapStringKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("错误信息: %v", err)
			}
		})
	}
}

// TestValidateMapIntKey 测试整数类型键验证
func TestValidateMapIntKey(t *testing.T) {
	tests := []struct {
		name    string
		extras  map[string]any
		key     string
		min     int
		max     int
		wantErr bool
	}{
		{
			name:    "整数在范围内",
			extras:  map[string]any{"age": 25},
			key:     "age",
			min:     0,
			max:     120,
			wantErr: false,
		},
		{
			name:    "整数太小",
			extras:  map[string]any{"age": -5},
			key:     "age",
			min:     0,
			max:     120,
			wantErr: true,
		},
		{
			name:    "整数太大",
			extras:  map[string]any{"age": 200},
			key:     "age",
			min:     0,
			max:     120,
			wantErr: true,
		},
		{
			name:    "不是整数类型",
			extras:  map[string]any{"age": "25"},
			key:     "age",
			min:     0,
			max:     120,
			wantErr: true,
		},
		{
			name:    "键不存在",
			extras:  map[string]any{},
			key:     "age",
			min:     0,
			max:     120,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMapIntKey(tt.extras, tt.key, tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMapIntKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("错误信息: %v", err)
			}
		})
	}
}

// TestValidateMapFloatKey 测试浮点数类型键验证
func TestValidateMapFloatKey(t *testing.T) {
	tests := []struct {
		name    string
		extras  map[string]any
		key     string
		min     float64
		max     float64
		wantErr bool
	}{
		{
			name:    "浮点数在范围内",
			extras:  map[string]any{"price": 99.99},
			key:     "price",
			min:     0.01,
			max:     999.99,
			wantErr: false,
		},
		{
			name:    "浮点数太小",
			extras:  map[string]any{"price": -10.5},
			key:     "price",
			min:     0.01,
			max:     999.99,
			wantErr: true,
		},
		{
			name:    "浮点数太大",
			extras:  map[string]any{"price": 1500.0},
			key:     "price",
			min:     0.01,
			max:     999.99,
			wantErr: true,
		},
		{
			name:    "不是浮点数类型",
			extras:  map[string]any{"price": "99.99"},
			key:     "price",
			min:     0.01,
			max:     999.99,
			wantErr: true,
		},
		{
			name:    "键不存在",
			extras:  map[string]any{},
			key:     "price",
			min:     0.01,
			max:     999.99,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMapFloatKey(tt.extras, tt.key, tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMapFloatKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("错误信息: %v", err)
			}
		})
	}
}

// TestValidateMapKey 测试自定义键验证
func TestValidateMapKey(t *testing.T) {
	tests := []struct {
		name          string
		extras        map[string]any
		key           string
		validatorFunc func(value interface{}) error
		wantErr       bool
	}{
		{
			name:   "自定义验证通过",
			extras: map[string]any{"status": "active"},
			key:    "status",
			validatorFunc: func(value interface{}) error {
				status, ok := value.(string)
				if !ok {
					return fmt.Errorf("status 必须是字符串")
				}
				validStatuses := map[string]bool{"active": true, "inactive": true}
				if !validStatuses[status] {
					return fmt.Errorf("status 必须是 active 或 inactive")
				}
				return nil
			},
			wantErr: false,
		},
		{
			name:   "自定义验证失败",
			extras: map[string]any{"status": "unknown"},
			key:    "status",
			validatorFunc: func(value interface{}) error {
				status, ok := value.(string)
				if !ok {
					return fmt.Errorf("status 必须是字符串")
				}
				validStatuses := map[string]bool{"active": true, "inactive": true}
				if !validStatuses[status] {
					return fmt.Errorf("status 必须是 active 或 inactive")
				}
				return nil
			},
			wantErr: true,
		},
		{
			name:   "键不存在",
			extras: map[string]any{},
			key:    "status",
			validatorFunc: func(value interface{}) error {
				return nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMapKey(tt.extras, tt.key, tt.validatorFunc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMapKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("错误信息: %v", err)
			}
		})
	}
}

// TestMapValidator_ChainedMethods 测试链式调用
func TestMapValidator_ChainedMethods(t *testing.T) {
	mv := NewMapValidator().
		WithRequiredKeys("name", "email").
		WithAllowedKeys("name", "email", "age", "phone").
		WithKeyValidator("email", func(value interface{}) error {
			email, ok := value.(string)
			if !ok {
				return fmt.Errorf("email 必须是字符串")
			}
			if len(email) < 5 {
				return fmt.Errorf("email 长度必须至少5个字符")
			}
			return nil
		})

	tests := []struct {
		name    string
		data    map[string]any
		wantErr bool
	}{
		{
			name: "所有验证通过",
			data: map[string]any{
				"name":  "John Doe",
				"email": "john@example.com",
				"age":   30,
			},
			wantErr: false,
		},
		{
			name: "缺少必填键",
			data: map[string]any{
				"name": "John Doe",
			},
			wantErr: true,
		},
		{
			name: "包含不允许的键",
			data: map[string]any{
				"name":    "John Doe",
				"email":   "john@example.com",
				"address": "123 Main St", // 不在允许列表中
			},
			wantErr: true,
		},
		{
			name: "自定义验证失败",
			data: map[string]any{
				"name":  "John Doe",
				"email": "abc", // 太短
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mv.Validate(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("错误信息: %v", err)
			}
		})
	}
}

// ============================================================================
// Map 验证性能基准测试
// ============================================================================

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

// ============================================================================
// 内存优化全面基准测试
// ============================================================================

// BenchmarkNewFieldError 测试 FieldError 创建性能
func BenchmarkNewFieldError(b *testing.B) {
	b.Run("with_pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			fe := NewFieldError("User.Name", "required", "")
			_ = fe
		}
	})

	b.Run("without_pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			fe := &FieldError{
				Namespace: "User.Name",
				Tag:       "required",
				Param:     "",
			}
			_ = fe
		}
	})
}

// BenchmarkValidationContextError 测试 Error() 方法性能
func BenchmarkValidationContextError(b *testing.B) {
	// 准备测试数据
	ctx := NewValidationContext(SceneNone)
	for i := 0; i < 5; i++ {
		ctx.AddError(NewFieldError("User.Field", "required", ""))
	}

	b.Run("with_pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = ctx.Error()
		}
	})

	releaseValidationContext(ctx)
}

// BenchmarkFieldErrorString 测试 FieldError.String() 性能
func BenchmarkFieldErrorString(b *testing.B) {
	fe := NewFieldError("User.Email", "email", "")

	b.Run("with_pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = fe.String()
		}
	})
}

// BenchmarkToLocalizes 测试本地化转换性能
func BenchmarkToLocalizes(b *testing.B) {
	fe := NewFieldError("User.Profile.Email", "email", "")

	b.Run("with_pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = fe.ToLocalizes()
		}
	})
}

// BenchmarkValidateComplete 完整验证流程基准测试
func BenchmarkValidateComplete(b *testing.B) {
	type TestUser struct {
		Name  string `validate:"required,min=3"`
		Email string `validate:"required,email"`
		Age   int    `validate:"required,min=18"`
	}

	user := &TestUser{
		Name:  "Jo",      // 太短
		Email: "invalid", // 无效邮箱
		Age:   16,        // 太小
	}

	v := New()

	b.Run("with_pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			errors := v.Validate(user, SceneNone)
			_ = errors
		}
	})
}

// BenchmarkAddError 测试添加错误性能
func BenchmarkAddError(b *testing.B) {
	b.Run("single_error", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ctx := acquireValidationContext(SceneNone)
			ctx.AddError(NewFieldError("User.Name", "required", ""))
			releaseValidationContext(ctx)
		}
	})

	b.Run("multiple_errors", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ctx := acquireValidationContext(SceneNone)
			for j := 0; j < 10; j++ {
				ctx.AddError(NewFieldError("User.Field", "required", ""))
			}
			releaseValidationContext(ctx)
		}
	})
}

// BenchmarkStringBuilderUsage 测试 StringBuilder 使用性能
func BenchmarkStringBuilderUsage(b *testing.B) {
	b.Run("with_pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			sb := acquireStringBuilder()
			sb.WriteString("field '")
			sb.WriteString("User.Name")
			sb.WriteString("' validation failed")
			_ = sb.String()
			releaseStringBuilder(sb)
		}
	})

	b.Run("without_pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := "field '" + "User.Name" + "' validation failed"
			_ = result
		}
	})
}

// BenchmarkMemoryFootprint 内存占用基准测试
func BenchmarkMemoryFootprint(b *testing.B) {
	b.Run("validation_context", func(b *testing.B) {
		b.ReportAllocs()
		contexts := make([]*ValidationContext, b.N)
		for i := 0; i < b.N; i++ {
			contexts[i] = acquireValidationContext(SceneNone)
		}
		for i := 0; i < b.N; i++ {
			releaseValidationContext(contexts[i])
		}
	})

	b.Run("field_errors", func(b *testing.B) {
		b.ReportAllocs()
		errors := make([]*FieldError, b.N)
		for i := 0; i < b.N; i++ {
			errors[i] = NewFieldError("User.Name", "required", "")
		}
		// FieldError 不需要手动释放，由 ValidationContext 管理
	})
}

// BenchmarkConcurrentValidation 并发验证基准测试
func BenchmarkConcurrentValidation(b *testing.B) {
	type TestUser struct {
		Name string `validate:"required,min=3"`
	}

	user := &TestUser{Name: "Jo"}
	v := New()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			errors := v.Validate(user, SceneNone)
			_ = errors
		}
	})
}

// BenchmarkHighLoad 高负载场景基准测试
func BenchmarkHighLoad(b *testing.B) {
	type ComplexUser struct {
		Name    string `validate:"required,min=3,max=50"`
		Email   string `validate:"required,email"`
		Age     int    `validate:"required,min=18,max=120"`
		Phone   string `validate:"required,len=11"`
		Address string `validate:"required,min=10"`
		City    string `validate:"required"`
		Country string `validate:"required"`
		Zip     string `validate:"required"`
		Bio     string `validate:"max=500"`
		Website string `validate:"omitempty,url"`
	}

	user := &ComplexUser{
		Name:    "Jo",
		Email:   "invalid",
		Age:     16,
		Phone:   "123",
		Address: "short",
		City:    "",
		Country: "",
		Zip:     "",
		Bio:     "",
		Website: "not-a-url",
	}

	v := New()

	b.Run("pool_optimized", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			errors := v.Validate(user, SceneNone)
			_ = errors
		}
	})
}
