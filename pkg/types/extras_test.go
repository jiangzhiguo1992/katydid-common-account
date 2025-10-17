package types

import (
	"encoding/json"
	"math"
	"sync"
	"testing"
)

// TestExtras_BasicOperations 测试基础操作
func TestExtras_BasicOperations(t *testing.T) {
	e := NewExtras()

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
	e := NewExtras()

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
	e := NewExtras()

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
	e := NewExtras()

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
	e := NewExtras()
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
	e := NewExtras()
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
	e := NewExtras()
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
	e1 := NewExtras()
	e1.Set("key1", "value1")
	e1.Set("key2", "value2")

	e2 := NewExtras()
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
		e := NewExtras()
		e.Set("key", "value")
		e.SetOrDel("key", nil)
		if e.Has("key") {
			t.Error("SetOrDel(nil) 应该删除键")
		}
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		e := NewExtras()
		if _, ok := e.GetString("nonexistent"); ok {
			t.Error("获取不存在的键应该返回 false")
		}
	})
}

// TestExtras_Capacity 测试预分配容量
func TestExtras_Capacity(t *testing.T) {
	// 测试使用容量创建
	e := NewExtrasWithCapacity(10)
	if e == nil {
		t.Fatal("NewExtrasWithCapacity 不应该返回 nil")
	}

	// 测试负数容量
	e2 := NewExtrasWithCapacity(-1)
	if e2 == nil {
		t.Fatal("负容量的 NewExtrasWithCapacity 应该返回有效的 Extras")
	}
}

// TestExtras_StringSliceEmpty 测试空切片优化
func TestExtras_StringSliceEmpty(t *testing.T) {
	e := NewExtras()
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
	e := NewExtras()
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
	e := NewExtras()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Set("key", "value")
	}
}

// BenchmarkExtras_Get 基准测试：Get 操作
func BenchmarkExtras_Get(b *testing.B) {
	e := NewExtras()
	e.Set("key", "value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.GetString("key")
	}
}

// BenchmarkExtras_GetInt 基准测试：GetInt 带类型转换
func BenchmarkExtras_GetInt(b *testing.B) {
	e := NewExtras()
	e.Set("key", 42)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.GetInt("key")
	}
}

// BenchmarkExtras_JSONMarshal 基准测试：JSON 序列化
func BenchmarkExtras_JSONMarshal(b *testing.B) {
	e := NewExtras()
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
	e := NewExtras()
	for i := 0; i < 10; i++ {
		e.Set(string(rune('a'+i)), i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.Clone()
	}
}
