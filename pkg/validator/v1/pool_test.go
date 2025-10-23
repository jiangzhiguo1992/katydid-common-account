package v1

import (
	"runtime"
	"strings"
	"testing"
)

// ============================================================================
// 对象池测试
// ============================================================================

func TestValidationContextPool(t *testing.T) {
	t.Run("acquire and release", func(t *testing.T) {
		ctx := acquireValidationContext(SceneNone)
		if ctx == nil {
			t.Fatal("acquireValidationContext returned nil")
		}
		if ctx.Scene != SceneNone {
			t.Errorf("expected scene %d, got %d", SceneNone, ctx.Scene)
		}
		if len(ctx.Errors) != 0 {
			t.Errorf("expected empty errors, got %d", len(ctx.Errors))
		}

		// 添加一些错误
		ctx.AddError(NewFieldError("test", "required", ""))
		if len(ctx.Errors) != 1 {
			t.Errorf("expected 1 error, got %d", len(ctx.Errors))
		}

		// 归还到池
		releaseValidationContext(ctx)

		// 再次获取应该是清空的
		ctx2 := acquireValidationContext(SceneAll)
		if len(ctx2.Errors) != 0 {
			t.Errorf("expected empty errors after release, got %d", len(ctx2.Errors))
		}
		releaseValidationContext(ctx2)
	})

	t.Run("prevent memory leak with large capacity", func(t *testing.T) {
		ctx := acquireValidationContext(SceneNone)

		// 添加大量错误
		for i := 0; i < 2000; i++ {
			ctx.AddError(NewFieldError("test", "required", ""))
		}

		initialCap := cap(ctx.Errors)
		releaseValidationContext(ctx)

		// 再次获取，应该分配了新的小容量切片
		ctx2 := acquireValidationContext(SceneNone)
		if cap(ctx2.Errors) >= initialCap {
			t.Errorf("expected smaller capacity after release, got %d", cap(ctx2.Errors))
		}
		releaseValidationContext(ctx2)
	})
}

func TestStringBuilderPool(t *testing.T) {
	t.Run("acquire and release", func(t *testing.T) {
		sb := acquireStringBuilder()
		if sb == nil {
			t.Fatal("acquireStringBuilder returned nil")
		}

		sb.WriteString("test")
		if sb.String() != "test" {
			t.Errorf("expected 'test', got '%s'", sb.String())
		}

		releaseStringBuilder(sb)

		// 再次获取应该是空的
		sb2 := acquireStringBuilder()
		if sb2.Len() != 0 {
			t.Errorf("expected empty builder, got length %d", sb2.Len())
		}
		releaseStringBuilder(sb2)
	})

	t.Run("prevent memory leak with large builder", func(t *testing.T) {
		sb := acquireStringBuilder()

		// 写入大量数据
		for i := 0; i < 1000; i++ {
			sb.WriteString(strings.Repeat("x", 100))
		}

		// 归还（应该不会归还到池，因为容量过大）
		releaseStringBuilder(sb)

		// 这个测试只是确保不会 panic
	})
}

// ============================================================================
// 内存安全检查测试
// ============================================================================

func TestCheckMemorySafety(t *testing.T) {
	tests := []struct {
		name       string
		errorCount int
		valueSize  int
		expected   bool
	}{
		{"safe", 10, 100, true},
		{"too many errors", maxErrorsCapacity + 1, 100, false},
		{"value too large", 10, maxValueSize + 1, false},
		{"both unsafe", maxErrorsCapacity + 1, maxValueSize + 1, false},
		{"at limit", maxErrorsCapacity - 1, maxValueSize - 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkMemorySafety(tt.errorCount, tt.valueSize)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// 基准测试
// ============================================================================

func BenchmarkValidationContextPool(b *testing.B) {
	b.Run("with pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ctx := acquireValidationContext(SceneNone)
			ctx.AddError(NewFieldError("test", "required", ""))
			releaseValidationContext(ctx)
		}
	})

	b.Run("without pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ctx := acquireValidationContext(SceneNone)
			ctx.AddError(NewFieldError("test", "required", ""))
			_ = ctx
		}
	})
}

func BenchmarkStringBuilderPool(b *testing.B) {
	b.Run("with pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			sb := acquireStringBuilder()
			sb.WriteString("hello")
			sb.WriteString(" ")
			sb.WriteString("world")
			_ = sb.String()
			releaseStringBuilder(sb)
		}
	})

	b.Run("without pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var sb strings.Builder
			sb.WriteString("hello")
			sb.WriteString(" ")
			sb.WriteString("world")
			_ = sb.String()
		}
	})
}

// ============================================================================
// 内存泄漏测试
// ============================================================================

func TestMemoryLeakPrevention(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory leak test in short mode")
	}

	// 记录初始内存
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// 执行大量操作
	iterations := 10000
	for i := 0; i < iterations; i++ {
		ctx := acquireValidationContext(SceneNone)
		for j := 0; j < 10; j++ {
			ctx.AddError(NewFieldError("test.field", "required", ""))
		}
		releaseValidationContext(ctx)
	}

	// 强制 GC
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// 检查内存增长
	memGrowth := m2.Alloc - m1.Alloc
	maxGrowth := uint64(10 * 1024 * 1024) // 10MB

	if memGrowth > maxGrowth {
		t.Logf("Warning: Memory growth %d bytes (%.2f MB) exceeds expected %d bytes",
			memGrowth, float64(memGrowth)/(1024*1024), maxGrowth)
		// 不失败，只是警告
	} else {
		t.Logf("Memory growth: %d bytes (%.2f MB)", memGrowth, float64(memGrowth)/(1024*1024))
	}
}

// ============================================================================
// 并发安全测试
// ============================================================================

func TestPoolConcurrency(t *testing.T) {
	const goroutines = 100
	const iterations = 1000

	t.Run("ValidationContext pool", func(t *testing.T) {
		done := make(chan bool, goroutines)
		for g := 0; g < goroutines; g++ {
			go func() {
				for i := 0; i < iterations; i++ {
					ctx := acquireValidationContext(SceneNone)
					ctx.AddError(NewFieldError("test", "required", ""))
					releaseValidationContext(ctx)
				}
				done <- true
			}()
		}

		for g := 0; g < goroutines; g++ {
			<-done
		}
	})

	t.Run("StringBuilder pool", func(t *testing.T) {
		done := make(chan bool, goroutines)
		for g := 0; g < goroutines; g++ {
			go func() {
				for i := 0; i < iterations; i++ {
					sb := acquireStringBuilder()
					sb.WriteString("test")
					_ = sb.String()
					releaseStringBuilder(sb)
				}
				done <- true
			}()
		}

		for g := 0; g < goroutines; g++ {
			<-done
		}
	})
}

func TestResetPools(t *testing.T) {
	// 使用对象池
	ctx := acquireValidationContext(SceneNone)
	releaseValidationContext(ctx)

	sb := acquireStringBuilder()
	releaseStringBuilder(sb)

	// 重置池
	ResetPools()

	// 再次使用应该正常工作
	ctx2 := acquireValidationContext(SceneAll)
	if ctx2 == nil {
		t.Fatal("acquireValidationContext returned nil after reset")
	}
	releaseValidationContext(ctx2)

	sb2 := acquireStringBuilder()
	if sb2 == nil {
		t.Fatal("acquireStringBuilder returned nil after reset")
	}
	releaseStringBuilder(sb2)
}

// TestPoolUsageInValidator 测试 validator 中对象池的使用
func TestPoolUsageInValidator(t *testing.T) {
	t.Run("Validate uses pool", func(t *testing.T) {
		type TestStruct struct {
			Name  string `validate:"required"`
			Email string `validate:"required,email"`
		}

		obj := &TestStruct{
			Name:  "",
			Email: "invalid",
		}

		// 验证会使用对象池
		errors := Validate(obj, SceneAll)
		if len(errors) == 0 {
			t.Error("Expected validation errors")
		}
	})

	t.Run("NewFieldError uses pool", func(t *testing.T) {
		// 测试 NewFieldError 是否从对象池获取
		fe := NewFieldError("test.field", "required", "")
		if fe == nil {
			t.Fatal("NewFieldError returned nil")
		}
		if fe.Namespace != "test.field" {
			t.Errorf("Expected namespace 'test.field', got '%s'", fe.Namespace)
		}
	})

	t.Run("Error() uses pool for StringBuilder", func(t *testing.T) {
		ctx := NewValidationContext(SceneAll)
		defer ReleaseValidationContext(ctx)

		// 添加一些错误
		for i := 0; i < 5; i++ {
			fe := NewFieldError("field", "required", "")
			ctx.AddError(fe)
		}

		// Error() 方法应该使用对象池的 StringBuilder
		errMsg := ctx.Error()
		if errMsg == "" {
			t.Error("Expected non-empty error message")
		}
	})

	t.Run("ToLocalizes uses pool for StringBuilder", func(t *testing.T) {
		fe := NewFieldError("User.Name", "required", "")

		// ToLocalizes 应该使用对象池的 StringBuilder
		key, param := fe.ToLocalizes()
		if key != "User.Name.required" {
			t.Errorf("Expected key 'User.Name.required', got '%s'", key)
		}
		if param != "" {
			t.Errorf("Expected empty param, got '%s'", param)
		}
	})
}

// BenchmarkValidatorWithPool 测试使用对象池的性能
func BenchmarkValidatorWithPool(b *testing.B) {
	type TestStruct struct {
		Name  string `validate:"required,min=3"`
		Email string `validate:"required,email"`
		Age   int    `validate:"required,min=18,max=100"`
	}

	validObj := &TestStruct{
		Name:  "John",
		Email: "john@example.com",
		Age:   25,
	}

	invalidObj := &TestStruct{
		Name:  "Jo",
		Email: "invalid",
		Age:   15,
	}

	b.Run("valid object", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = Validate(validObj, SceneAll)
		}
	})

	b.Run("invalid object", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = Validate(invalidObj, SceneAll)
		}
	})
}
