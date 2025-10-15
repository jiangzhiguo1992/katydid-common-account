package types

import (
	"encoding/json"
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

	if count, ok := e2.GetInt("count"); !ok || count != 42 {
		t.Errorf("Expected 42, got %d", count)
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

	if num, ok := e2.GetInt("key2"); !ok || num != 123 {
		t.Errorf("Expected 123, got %d", num)
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
