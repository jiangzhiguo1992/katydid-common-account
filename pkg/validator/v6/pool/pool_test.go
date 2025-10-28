package pool

import (
	"testing"

	"katydid-common-account/pkg/validator/v6/core"
)

// TestValidationContextPool 测试验证上下文对象池
func TestValidationContextPool(t *testing.T) {
	pool := NewValidationContextPool()

	req := core.NewValidationRequest(&testUser{Name: "test"}, 1)

	// 获取上下文
	ctx := pool.Get(req, 10)
	if ctx == nil {
		t.Fatal("Expected context, got nil")
	}

	// 使用上下文
	ctx.Set("key", "value")
	if val, ok := ctx.Get("key"); !ok || val != "value" {
		t.Errorf("Expected value, got %v", val)
	}

	// 放回池中
	pool.Put(ctx)

	// 再次获取（应该复用）
	ctx2 := pool.Get(req, 10)
	if ctx2 == nil {
		t.Fatal("Expected context, got nil")
	}

	// 验证已清空
	if val, ok := ctx2.Get("key"); ok {
		t.Errorf("Expected empty context, got value: %v", val)
	}

	pool.Put(ctx2)
}

// TestErrorCollectorPool 测试错误收集器对象池
func TestErrorCollectorPool(t *testing.T) {
	pool := NewErrorCollectorPool(10)

	// 获取收集器
	ec := pool.Get()
	if ec == nil {
		t.Fatal("Expected error collector, got nil")
	}

	// 添加错误
	ec.Add(core.NewFieldError("field1", "required"))
	if ec.Count() != 1 {
		t.Errorf("Expected 1 error, got %d", ec.Count())
	}

	// 放回池中
	pool.Put(ec)

	// 再次获取（应该复用且已清空）
	ec2 := pool.Get()
	if ec2 == nil {
		t.Fatal("Expected error collector, got nil")
	}

	if ec2.Count() != 0 {
		t.Errorf("Expected 0 errors, got %d", ec2.Count())
	}

	pool.Put(ec2)
}

// TestFieldErrorPool 测试字段错误对象池
func TestFieldErrorPool(t *testing.T) {
	pool := NewFieldErrorPool()

	// 获取错误对象
	err := pool.Get()
	if err == nil {
		t.Fatal("Expected field error, got nil")
	}

	// 设置字段
	err.Namespace = "User.Name"
	err.Tag = "required"

	// 放回池中
	pool.Put(err)

	// 再次获取（应该复用且已清空）
	err2 := pool.Get()
	if err2 == nil {
		t.Fatal("Expected field error, got nil")
	}

	if err2.Namespace != "" || err2.Tag != "" {
		t.Errorf("Expected empty error, got Namespace=%s, Tag=%s", err2.Namespace, err2.Tag)
	}

	pool.Put(err2)
}

// TestGlobalPool 测试全局对象池
func TestGlobalPool(t *testing.T) {
	req := core.NewValidationRequest(&testUser{Name: "test"}, 1)

	// 使用全局池
	ctx := GlobalPool.ValidationContext.Get(req, 10)
	if ctx == nil {
		t.Fatal("Expected context from global pool, got nil")
	}
	GlobalPool.ValidationContext.Put(ctx)

	ec := GlobalPool.ErrorCollector.Get()
	if ec == nil {
		t.Fatal("Expected error collector from global pool, got nil")
	}
	GlobalPool.ErrorCollector.Put(ec)

	err := GlobalPool.FieldError.Get()
	if err == nil {
		t.Fatal("Expected field error from global pool, got nil")
	}
	GlobalPool.FieldError.Put(err)
}

// BenchmarkValidationContextPool 基准测试：对象池 vs 直接创建
func BenchmarkValidationContextPool(b *testing.B) {
	pool := NewValidationContextPool()
	req := core.NewValidationRequest(&testUser{Name: "test"}, 1)

	b.Run("WithPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx := pool.Get(req, 10)
			pool.Put(ctx)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = core.NewValidationRequest(&testUser{Name: "test"}, 1)
		}
	})
}

// BenchmarkErrorCollectorPool 基准测试：错误收集器对象池
func BenchmarkErrorCollectorPool(b *testing.B) {
	pool := NewErrorCollectorPool(10)

	b.Run("WithPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ec := pool.Get()
			ec.Add(core.NewFieldError("field", "required"))
			pool.Put(ec)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ec := GlobalPool.ErrorCollector.Get()
			ec.Add(core.NewFieldError("field", "required"))
			GlobalPool.ErrorCollector.Put(ec)
		}
	})
}

// testUser 测试用户结构
type testUser struct {
	Name string
}

func (u *testUser) GetRules() map[core.Scene]map[string]string {
	return map[core.Scene]map[string]string{
		1: {"name": "required"},
	}
}
