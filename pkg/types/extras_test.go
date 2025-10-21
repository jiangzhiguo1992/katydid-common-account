package types

import (
	"encoding/json"
	"math"
	"strings"
	"sync"
	"testing"
)

// TestExtras_BasicOperations 测试基础操作
func TestExtras_BasicOperations(t *testing.T) {
	e := NewExtras(0)

	// 测试设置和获取字符串
	e.Set("name", "John Doe")
	if val, ok := e.GetString("name"); !ok || val != "John Doe" {
		t.Errorf("Expected 'John Doe', got '%s'", val)
	}

	// 测试设置和获取整数
	e.Set("age", 30)
	if val, ok := e.GetInt("age"); !ok || val != 30 {
		t.Errorf("Expected 30, got %d", val)
	}

	// 测试设置和获取布尔值
	e.Set("active", true)
	if val, ok := e.GetBool("active"); !ok || !val {
		t.Errorf("Expected true, got %v", val)
	}

	// 测试设置和获取浮点数
	e.Set("price", 99.99)
	if val, ok := e.GetFloat64("price"); !ok || val != 99.99 {
		t.Errorf("Expected 99.99, got %f", val)
	}

	// 测试Has方法
	if !e.Has("name") {
		t.Error("Expected 'name' key to exist")
	}

	// 测试Len方法
	if e.Len() != 4 {
		t.Errorf("Expected length 4, got %d", e.Len())
	}

	// 测试Delete方法
	e.Delete("age")
	if e.Has("age") {
		t.Error("Expected 'age' key to be deleted")
	}
}

// TestExtras_EmptyKey 测试空键名的防御性检查
func TestExtras_EmptyKey(t *testing.T) {
	e := NewExtras(0)

	// 设置空键名应该被忽略
	e.Set("", "value")
	if e.Has("") {
		t.Error("空键名不应该被设置")
	}

	// SetOrDel 空键名也应该被忽略
	e.SetOrDel("", "value")
	if e.Has("") {
		t.Error("SetOrDel 空键名不应该被设置")
	}

	if e.Len() != 0 {
		t.Errorf("设置空键名后，长度应该为 0，实际为 %d", e.Len())
	}
}

// TestExtras_ComplexTypes 测试复杂类型
func TestExtras_ComplexTypes(t *testing.T) {
	e := NewExtras(0)

	// 测试数组
	tags := []any{"go", "database", "api"}
	e.Set("tags", tags)
	if val, ok := e.GetSlice("tags"); !ok || len(val) != 3 {
		t.Errorf("Expected slice with 3 elements, got %v", val)
	}

	// 测试对象
	metadata := map[string]any{
		"version": "1.0",
		"author":  "Admin",
	}
	e.Set("metadata", metadata)
	if val, ok := e.GetMap("metadata"); !ok || val["version"] != "1.0" {
		t.Errorf("Expected map with version '1.0', got %v", val)
	}
}

// TestExtras_TypeConversion 测试类型转换和边界检查
func TestExtras_TypeConversion(t *testing.T) {
	e := NewExtras(0)

	// 测试 int 类型转换
	e.Set("int8_val", int8(100))
	if val, ok := e.GetInt("int8_val"); !ok || val != 100 {
		t.Errorf("int8 转 int 失败: got %d, ok=%v", val, ok)
	}

	// 测试溢出检查
	e.Set("overflow", uint64(math.MaxUint64))
	if _, ok := e.GetInt("overflow"); ok {
		t.Error("uint64 最大值转 int 应该失败")
	}

	// 测试浮点数转整数（整数值）
	e.Set("float_int", 42.0)
	if val, ok := e.GetInt("float_int"); !ok || val != 42 {
		t.Errorf("浮点数 42.0 转 int 应该成功: got %d, ok=%v", val, ok)
	}

	// 测试浮点数转整数（非整数值）
	e.Set("float_frac", 42.5)
	if _, ok := e.GetInt("float_frac"); ok {
		t.Error("浮点数 42.5 转 int 应该失败")
	}
}

// TestExtras_JSONSerialization 测试 JSON 序列化
func TestExtras_JSONSerialization(t *testing.T) {
	e := NewExtras(0)
	e.Set("name", "Test")
	e.Set("count", 42)
	e.Set("enabled", true)
	e.Set("tags", []any{"a", "b", "c"})

	// 序列化
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	// 反序列化
	var e2 Extras
	err = json.Unmarshal(data, &e2)
	if err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	// 验证数据
	if name, ok := e2.GetString("name"); !ok || name != "Test" {
		t.Errorf("Expected 'Test', got '%s'", name)
	}

	// 注意：JSON 反序列化后，数字会变成 float64，但我们的 GetInt 应该能处理这种情况
	if count, ok := e2.GetInt("count"); !ok || count != 42 {
		t.Errorf("Expected 42, got %d (ok=%v, actual type: %T, value: %v)", count, ok, e2["count"], e2["count"])
	}

	if enabled, ok := e2.GetBool("enabled"); !ok || !enabled {
		t.Errorf("Expected true, got %v", enabled)
	}
}

// TestExtras_DatabaseScan 测试数据库扫描
func TestExtras_DatabaseScan(t *testing.T) {
	e := NewExtras(0)
	e.Set("key1", "value1")
	e.Set("key2", 123)

	// 模拟数据库Value操作
	val, err := e.Value()
	if err != nil {
		t.Fatalf("Value() 失败: %v", err)
	}

	// 模拟数据库Scan操作
	var e2 Extras
	err = e2.Scan(val)
	if err != nil {
		t.Fatalf("Scan() 失败: %v", err)
	}

	// 验证数据
	if str, ok := e2.GetString("key1"); !ok || str != "value1" {
		t.Errorf("Expected 'value1', got '%s'", str)
	}

	// JSON 反序列化后数字会变成 float64
	if num, ok := e2.GetInt("key2"); !ok || num != 123 {
		t.Errorf("Expected 123, got %d (ok=%v, actual type: %T)", num, ok, e2["key2"])
	}
}

// TestExtras_NilAndEmpty 测试 nil 和空值
func TestExtras_NilAndEmpty(t *testing.T) {
	// 测试空Extras
	var e Extras

	// Value应该返回nil
	val, err := e.Value()
	if err != nil {
		t.Fatalf("Value() 失败: %v", err)
	}
	if val != nil {
		t.Errorf("空 Extras 的 Value 应该返回 nil，实际返回 %v", val)
	}

	// Scan nil
	err = e.Scan(nil)
	if err != nil {
		t.Fatalf("Scan(nil) 失败: %v", err)
	}
}

// TestExtras_Clone 测试克隆
func TestExtras_Clone(t *testing.T) {
	e := NewExtras(0)
	e.Set("key1", "value1")
	e.Set("key2", 42)

	// 克隆
	clone := e.Clone()

	// 修改原始对象
	e.Set("key3", "value3")

	// 验证克隆对象不受影响
	if clone.Has("key3") {
		t.Error("克隆对象不应该有 key3")
	}

	if clone.Len() != 2 {
		t.Errorf("克隆对象长度应该为 2，实际为 %d", clone.Len())
	}
}

// TestExtras_Merge 测试合并
func TestExtras_Merge(t *testing.T) {
	e1 := NewExtras(0)
	e1.Set("key1", "value1")
	e1.Set("key2", "value2")

	e2 := NewExtras(0)
	e2.Set("key2", "new_value2")
	e2.Set("key3", "value3")

	// 合并
	e1.Merge(e2)

	// 验证合并结果
	if val, ok := e1.GetString("key2"); !ok || val != "new_value2" {
		t.Errorf("key2 应该被覆盖为 'new_value2'，实际为 '%s'", val)
	}

	if !e1.Has("key3") {
		t.Error("合并后应该有 key3")
	}

	if e1.Len() != 3 {
		t.Errorf("合并后长度应该为 3，实际为 %d", e1.Len())
	}
}

// TestExtras_SetOrDel 测试条件设置
func TestExtras_SetOrDel(t *testing.T) {
	t.Run("SetOrDel with nil", func(t *testing.T) {
		e := NewExtras(0)
		e.Set("key", "value")
		e.SetOrDel("key", nil)
		if e.Has("key") {
			t.Error("SetOrDel(nil) 应该删除键")
		}
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		e := NewExtras(0)
		if _, ok := e.GetString("nonexistent"); ok {
			t.Error("获取不存在的键应该返回 false")
		}
	})
}

// TestExtras_Capacity 测试预分配容量
func TestExtras_Capacity(t *testing.T) {
	// 测试使用容量创建
	e := NewExtras(10)
	if e == nil {
		t.Fatal("NewExtrasWithCapacity 不应该返回 nil")
	}

	// 测试负数容量
	e2 := NewExtras(-1)
	if e2 == nil {
		t.Fatal("负容量的 NewExtrasWithCapacity 应该返回有效的 Extras")
	}
}

// TestExtras_StringSliceEmpty 测试空切片优化
func TestExtras_StringSliceEmpty(t *testing.T) {
	e := NewExtras(0)
	e.Set("empty_slice", []any{})

	slice, ok := e.GetStringSlice("empty_slice")
	if !ok {
		t.Error("空切片应该能成功获取")
	}
	if len(slice) != 0 {
		t.Errorf("空切片长度应该为 0，实际为 %d", len(slice))
	}
}

// TestExtras_ConcurrentRead 测试并发读取（安全）
func TestExtras_ConcurrentRead(t *testing.T) {
	e := NewExtras(0)
	e.Set("key1", "value1")
	e.Set("key2", 42)
	e.Set("key3", true)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 并发读取是安全的
			_, _ = e.GetString("key1")
			_, _ = e.GetInt("key2")
			_, _ = e.GetBool("key3")
		}()
	}
	wg.Wait()
}

// BenchmarkExtras_Set 基准测试：Set 操作
func BenchmarkExtras_Set(b *testing.B) {
	e := NewExtras(0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Set("key", "value")
	}
}

// BenchmarkExtras_Get 基准测试：Get 操作
func BenchmarkExtras_Get(b *testing.B) {
	e := NewExtras(0)
	e.Set("key", "value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.GetString("key")
	}
}

// BenchmarkExtras_GetInt 基准测试：GetInt 带类型转换
func BenchmarkExtras_GetInt(b *testing.B) {
	e := NewExtras(0)
	e.Set("key", 42)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.GetInt("key")
	}
}

// BenchmarkExtras_JSONMarshal 基准测试：JSON 序列化
func BenchmarkExtras_JSONMarshal(b *testing.B) {
	e := NewExtras(0)
	e.Set("name", "test")
	e.Set("age", 30)
	e.Set("active", true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(e)
	}
}

// BenchmarkExtras_Clone 基准测试：Clone 操作
func BenchmarkExtras_Clone(b *testing.B) {
	e := NewExtras(0)
	for i := 0; i < 10; i++ {
		e.Set(string(rune('a'+i)), i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.Clone()
	}
}

// ==================== 安全性测试 ====================

// TestNilSafety 测试nil安全性
func TestNilSafety(t *testing.T) {
	t.Run("Set on nil map", func(t *testing.T) {
		var extras Extras // nil

		// 应该不会panic
		extras.Set("key", "value")

		// nil map上Set应该被忽略
		if extras != nil {
			t.Error("Expected nil map to remain nil after Set")
		}
	})

	t.Run("SetOrDel on nil map", func(t *testing.T) {
		var extras Extras

		// 不应该panic
		extras.SetOrDel("key", "value")
		extras.SetOrDel("key", nil)

		if extras != nil {
			t.Error("Expected nil map to remain nil")
		}
	})

	t.Run("SetPath on nil map", func(t *testing.T) {
		var extras Extras

		// 应该返回错误
		err := extras.SetPath("user.name", "Alice")
		if err == nil {
			t.Error("Expected error when SetPath on nil map")
		}
		if !strings.Contains(err.Error(), "nil") {
			t.Errorf("Expected nil error message, got: %v", err)
		}
	})
}

// TestPathInjectionPrevention 测试路径注入防护
func TestPathInjectionPrevention(t *testing.T) {
	extras := Extras{
		"user": Extras{
			"name": "Alice",
		},
	}

	tests := []struct {
		name     string
		path     string
		wantErr  bool
		wantFail bool
	}{
		{"empty path", "", false, true},
		{"valid path", "user.name", false, false},
		{"path with empty key start", ".user.name", false, true},
		{"path with empty key middle", "user..name", false, true},
		{"path with empty key end", "user.name.", false, true},
		{"only dots", "...", false, true},
		{"single dot", ".", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试GetPath
			_, ok := extras.GetPath(tt.path)
			if !tt.wantFail && !ok {
				t.Errorf("GetPath(%q) failed unexpectedly", tt.path)
			}
			if tt.wantFail && ok {
				t.Errorf("GetPath(%q) should fail but succeeded", tt.path)
			}

			// 测试SetPath
			err := extras.SetPath(tt.path, "test")
			if tt.wantErr && err == nil {
				t.Errorf("SetPath(%q) should return error", tt.path)
			}
		})
	}
}

// TestSetPathTypeConflict 测试SetPath的类型冲突检测
func TestSetPathTypeConflict(t *testing.T) {
	t.Run("overwrite string with Extras", func(t *testing.T) {
		extras := Extras{
			"user": "Alice", // 字符串类型
		}

		// 尝试将user.age设置值，但user是字符串
		err := extras.SetPath("user.age", 30)

		// 应该返回错误
		if err == nil {
			t.Error("Expected error when setting path on non-Extras type")
		}

		if !strings.Contains(err.Error(), "conflict") && !strings.Contains(err.Error(), "not an Extras") {
			t.Errorf("Expected type conflict error, got: %v", err)
		}

		// 原值不应该被修改
		if val, ok := extras.GetString("user"); !ok || val != "Alice" {
			t.Error("Original value should not be modified")
		}
	})

	t.Run("overwrite int with Extras", func(t *testing.T) {
		extras := Extras{
			"count": 42,
		}

		err := extras.SetPath("count.value", 100)
		if err == nil {
			t.Error("Expected error when setting path on non-Extras type")
		}

		// 原值保持不变
		if val, ok := extras.GetInt("count"); !ok || val != 42 {
			t.Error("Original value should not be modified")
		}
	})

	t.Run("valid nested creation", func(t *testing.T) {
		extras := NewExtras(0)

		// 应该成功创建嵌套结构
		err := extras.SetPath("user.profile.name", "Bob")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// 验证结构
		if name, ok := extras.GetStringPath("user.profile.name"); !ok || name != "Bob" {
			t.Error("Failed to create nested structure")
		}
	})
}

// TestEmptyKeyProtection 测试空键保护
func TestEmptyKeyProtection(t *testing.T) {
	extras := NewExtras(0)

	// Set空键应该被忽略
	extras.Set("", "value")
	if extras.Has("") {
		t.Error("Empty key should not be stored")
	}

	// SetOrDel空键应该被忽略
	extras.SetOrDel("", "value")
	if extras.Has("") {
		t.Error("Empty key should not be stored")
	}

	// SetPath中的空键应该被拒绝
	err := extras.SetPath("valid..invalid", "value")
	if err == nil {
		t.Error("Expected error for path with empty key")
	}
}

// ==================== 性能测试 ====================

// BenchmarkSetWithNilCheck 测试nil检查的性能影响
func BenchmarkSetWithNilCheck(b *testing.B) {
	extras := NewExtras(0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extras.Set("key", "value")
	}
}

// BenchmarkGetPathWithValidation 测试路径验证的性能影响
func BenchmarkGetPathWithValidation(b *testing.B) {
	extras := Extras{
		"user": Extras{
			"profile": Extras{
				"name": "Alice",
			},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extras.GetPath("user.profile.name")
	}
}

// BenchmarkFilterWithPrealloc 测试预分配的性能提升
func BenchmarkFilterWithPrealloc(b *testing.B) {
	extras := NewExtras(100)
	for i := 0; i < 100; i++ {
		extras.Set(string(rune('a'+i%26))+string(rune(i)), i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extras.Filter(func(k string, v any) bool {
			if num, ok := v.(int); ok {
				return num%2 == 0
			}
			return false
		})
	}
}

// ==================== 边界条件测试 ====================

// TestEdgeCases 测试边界情况
func TestEdgeCases(t *testing.T) {
	t.Run("very long path", func(t *testing.T) {
		extras := NewExtras(0)

		// 创建深层嵌套
		var parts []string
		for i := 0; i < 20; i++ {
			parts = append(parts, "level"+string(rune('0'+i)))
		}
		path := strings.Join(parts, ".")

		err := extras.SetPath(path, "deep value")
		if err != nil {
			t.Logf("Deep path rejected (expected if MAX_DEPTH limit added): %v", err)
		}
	})

	t.Run("very long key", func(t *testing.T) {
		extras := NewExtras(0)
		longKey := strings.Repeat("a", 1000)

		extras.Set(longKey, "value")
		// 当前实现会接受，但建议添加长度限制
		if !extras.Has(longKey) {
			t.Log("Long key rejected (good if MAX_KEY_LENGTH added)")
		}
	})

	t.Run("unicode keys", func(t *testing.T) {
		extras := NewExtras(0)

		extras.Set("用户", "Alice")
		extras.Set("🔑", "key emoji")

		if val, ok := extras.GetString("用户"); !ok || val != "Alice" {
			t.Error("Failed to handle Unicode key")
		}

		if val, ok := extras.GetString("🔑"); !ok || val != "key emoji" {
			t.Error("Failed to handle Emoji key")
		}
	})
}

// TestConcurrentSafetyWarning 测试并发问题（应该失败，证明需要锁）
func TestConcurrentSafetyWarning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent safety test in short mode")
	}

	t.Run("detect race condition", func(t *testing.T) {
		// 这个测试在race detector下应该会失败
		// 运行: go test -race

		extras := NewExtras(0)
		done := make(chan bool)

		// 并发写入
		go func() {
			for i := 0; i < 100; i++ {
				extras.Set("key1", i)
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 100; i++ {
				extras.Set("key2", i)
			}
			done <- true
		}()

		<-done
		<-done

		t.Log("Concurrent writes completed - run with -race to detect issues")
	})
}

// ==================== 数据完整性测试 ====================

// TestDataIntegrity 测试数据完整性
func TestDataIntegrity(t *testing.T) {
	t.Run("SetPath preserves existing data", func(t *testing.T) {
		extras := Extras{
			"user": Extras{
				"name":  "Alice",
				"email": "alice@example.com",
			},
		}

		// 添加新字段
		err := extras.SetPath("user.age", 30)
		if err != nil {
			t.Fatalf("SetPath failed: %v", err)
		}

		// 验证旧数据未被破坏
		if name, ok := extras.GetStringPath("user.name"); !ok || name != "Alice" {
			t.Error("Existing name field was corrupted")
		}

		if email, ok := extras.GetStringPath("user.email"); !ok || email != "alice@example.com" {
			t.Error("Existing email field was corrupted")
		}

		// 验证新数据正确
		if age, ok := extras.GetIntPath("user.age"); !ok || age != 30 {
			t.Error("New age field not set correctly")
		}
	})

	t.Run("Clone preserves all data", func(t *testing.T) {
		original := Extras{
			"string": "value",
			"int":    42,
			"float":  3.14,
			"bool":   true,
			"slice":  []int{1, 2, 3},
		}

		cloned := original.Clone()

		// 验证所有字段
		if v, ok := cloned.GetString("string"); !ok || v != "value" {
			t.Error("String field not cloned correctly")
		}

		if v, ok := cloned.GetInt("int"); !ok || v != 42 {
			t.Error("Int field not cloned correctly")
		}

		if v, ok := cloned.GetFloat64("float"); !ok || v != 3.14 {
			t.Error("Float field not cloned correctly")
		}

		if v, ok := cloned.GetBool("bool"); !ok || v != true {
			t.Error("Bool field not cloned correctly")
		}
	})
}
