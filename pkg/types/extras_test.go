package types

import (
	"encoding/json"
	"math"
	"testing"
)

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

func TestExtras_JSONSerialization(t *testing.T) {
	e := NewExtras()
	e.Set("name", "Test")
	e.Set("count", 42)
	e.Set("enabled", true)
	e.Set("tags", []any{"a", "b", "c"})

	// 序列化
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// 反序列化
	var e2 Extras
	err = json.Unmarshal(data, &e2)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// 验证数据
	if name, ok := e2.GetString("name"); !ok || name != "Test" {
		t.Errorf("Expected 'Test', got '%s'", name)
	}

	// 注意：JSON 反序列化后，数字会变成 float64，但我们的 GetInt 应该能处理这种情况
	if count, ok := e2.GetInt("count"); !ok || count != 42 {
		// 如果失败，显示实际类型以便调试
		t.Errorf("Expected 42, got %d (ok=%v, actual type: %T, value: %v)", count, ok, e2["count"], e2["count"])
	}

	if enabled, ok := e2.GetBool("enabled"); !ok || !enabled {
		t.Errorf("Expected true, got %v", enabled)
	}
}

func TestExtras_DatabaseScan(t *testing.T) {
	e := NewExtras()
	e.Set("key1", "value1")
	e.Set("key2", 123)

	// 模拟数据库Value操作
	val, err := e.Value()
	if err != nil {
		t.Fatalf("Value() failed: %v", err)
	}

	// 模拟数据库Scan操作
	var e2 Extras
	err = e2.Scan(val)
	if err != nil {
		t.Fatalf("Scan() failed: %v", err)
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

func TestExtras_NilAndEmpty(t *testing.T) {
	// 测试空Extras
	var e Extras

	// Value应该返回nil
	val, err := e.Value()
	if err != nil {
		t.Fatalf("Value() failed: %v", err)
	}
	if val != nil {
		t.Errorf("Expected nil value for empty Extras, got %v", val)
	}

	// Scan nil
	err = e.Scan(nil)
	if err != nil {
		t.Fatalf("Scan(nil) failed: %v", err)
	}
}

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
		t.Error("Clone should not have key3")
	}

	if clone.Len() != 2 {
		t.Errorf("Expected clone length 2, got %d", clone.Len())
	}
}

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
		t.Errorf("Expected 'new_value2', got '%s'", val)
	}

	if !e1.Has("key3") {
		t.Error("Expected key3 to exist after merge")
	}

	if e1.Len() != 3 {
		t.Errorf("Expected length 3, got %d", e1.Len())
	}
}

func TestExtras_Clear(t *testing.T) {
	e := NewExtras()
	e.Set("key1", "value1")
	e.Set("key2", "value2")

	e.Clear()

	if !e.IsEmpty() {
		t.Error("Expected Extras to be empty after Clear()")
	}

	if e.Len() != 0 {
		t.Errorf("Expected length 0, got %d", e.Len())
	}
}

func TestExtras_Keys(t *testing.T) {
	e := NewExtras()
	e.Set("key1", "value1")
	e.Set("key2", "value2")
	e.Set("key3", "value3")

	keys := e.Keys()
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// 验证所有键都存在
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	if !keyMap["key1"] || !keyMap["key2"] || !keyMap["key3"] {
		t.Error("Not all expected keys found")
	}
}

// TestExtras_TypeConversion 专门测试类型转换功能
func TestExtras_TypeConversion(t *testing.T) {
	e := NewExtras()

	// 测试所有整数类型
	t.Run("Int Types", func(t *testing.T) {
		e.Set("int8", int8(127))
		e.Set("int16", int16(32767))
		e.Set("int32", int32(2147483647))
		e.Set("int64", int64(9223372036854775807))

		if v, ok := e.GetInt8("int8"); !ok || v != 127 {
			t.Errorf("GetInt8 failed: got %v, %v", v, ok)
		}
		if v, ok := e.GetInt16("int16"); !ok || v != 32767 {
			t.Errorf("GetInt16 failed: got %v, %v", v, ok)
		}
		if v, ok := e.GetInt32("int32"); !ok || v != 2147483647 {
			t.Errorf("GetInt32 failed: got %v, %v", v, ok)
		}
		if v, ok := e.GetInt64("int64"); !ok || v != 9223372036854775807 {
			t.Errorf("GetInt64 failed: got %v, %v", v, ok)
		}
	})

	// 测试所有无符号整数类型
	t.Run("Uint Types", func(t *testing.T) {
		e.Set("uint8", uint8(255))
		e.Set("uint16", uint16(65535))
		e.Set("uint32", uint32(4294967295))
		e.Set("uint64", uint64(math.MaxUint64))

		if v, ok := e.GetUint8("uint8"); !ok || v != 255 {
			t.Errorf("GetUint8 failed: got %v, %v", v, ok)
		}
		if v, ok := e.GetUint16("uint16"); !ok || v != 65535 {
			t.Errorf("GetUint16 failed: got %v, %v", v, ok)
		}
		if v, ok := e.GetUint32("uint32"); !ok || v != 4294967295 {
			t.Errorf("GetUint32 failed: got %v, %v", v, ok)
		}
		if v, ok := e.GetUint64("uint64"); !ok || v != math.MaxUint64 {
			t.Errorf("GetUint64 failed: got %v, %v", v, ok)
		}
	})

	// 测试浮点数类型
	t.Run("Float Types", func(t *testing.T) {
		e.Set("float32", float32(3.14))
		e.Set("float64", float64(3.141592653589793))

		if v, ok := e.GetFloat32("float32"); !ok || v != float32(3.14) {
			t.Errorf("GetFloat32 failed: got %v, %v", v, ok)
		}
		if v, ok := e.GetFloat64("float64"); !ok || v != 3.141592653589793 {
			t.Errorf("GetFloat64 failed: got %v, %v", v, ok)
		}
	})
}

// TestExtras_JSONFloatConversion 测试 JSON 反序列化后的 float64 转换
func TestExtras_JSONFloatConversion(t *testing.T) {
	// 这是最常见的问题：JSON 反序列化后数字会变成 float64
	jsonData := `{"age": 30, "price": 99.99, "count": 42.0}`

	var e Extras
	err := json.Unmarshal([]byte(jsonData), &e)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// 测试 float64 到 int 的转换（整数值）
	if age, ok := e.GetInt("age"); !ok || age != 30 {
		t.Errorf("Expected age=30, got %v (ok=%v, type=%T)", age, ok, e["age"])
	}

	// 测试 float64 到 float64 的转换
	if price, ok := e.GetFloat64("price"); !ok || price != 99.99 {
		t.Errorf("Expected price=99.99, got %v (ok=%v)", price, ok)
	}

	// 测试 float64(整数值) 到 int 的转换
	if count, ok := e.GetInt("count"); !ok || count != 42 {
		t.Errorf("Expected count=42, got %v (ok=%v, type=%T)", count, ok, e["count"])
	}
}

// TestExtras_BoundaryChecks 测试边界检查
func TestExtras_BoundaryChecks(t *testing.T) {
	e := NewExtras()

	t.Run("Overflow Prevention", func(t *testing.T) {
		// int64 到 int8 的溢出
		e.Set("big", int64(1000))
		if _, ok := e.GetInt8("big"); ok {
			t.Error("GetInt8 should fail for value > 127")
		}

		// 负数到 uint 的转换
		e.Set("negative", int(-10))
		if _, ok := e.GetUint("negative"); ok {
			t.Error("GetUint should fail for negative values")
		}

		// float 小数到 int 的转换
		e.Set("decimal", float64(3.14))
		if _, ok := e.GetInt("decimal"); ok {
			t.Error("GetInt should fail for non-integer float values")
		}
	})
}

// TestExtras_SliceConversion 测试切片转换
func TestExtras_SliceConversion(t *testing.T) {
	e := NewExtras()

	t.Run("String Slice", func(t *testing.T) {
		e.Set("tags", []string{"a", "b", "c"})
		if tags, ok := e.GetStringSlice("tags"); !ok || len(tags) != 3 {
			t.Errorf("GetStringSlice failed: got %v, %v", tags, ok)
		}
	})

	t.Run("Int Slice", func(t *testing.T) {
		e.Set("numbers", []int{1, 2, 3})
		if nums, ok := e.GetIntSlice("numbers"); !ok || len(nums) != 3 {
			t.Errorf("GetIntSlice failed: got %v, %v", nums, ok)
		}
	})

	t.Run("Any Slice to Typed Slice", func(t *testing.T) {
		e.Set("mixed", []any{1, 2, 3})
		if nums, ok := e.GetIntSlice("mixed"); !ok || len(nums) != 3 {
			t.Errorf("GetIntSlice from []any failed: got %v, %v", nums, ok)
		}
	})
}

// TestExtras_EdgeCases 测试边缘情况
func TestExtras_EdgeCases(t *testing.T) {
	t.Run("Nil Extras", func(t *testing.T) {
		var e Extras
		if !e.IsEmpty() {
			t.Error("Nil Extras should be empty")
		}
		if e.Len() != 0 {
			t.Error("Nil Extras length should be 0")
		}
	})

	t.Run("SetOrDel with nil", func(t *testing.T) {
		e := NewExtras()
		e.Set("key", "value")
		e.SetOrDel("key", nil)
		if e.Has("key") {
			t.Error("SetOrDel(nil) should delete the key")
		}
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		e := NewExtras()
		if _, ok := e.GetString("nonexistent"); ok {
			t.Error("Getting non-existent key should return false")
		}
	})
}
