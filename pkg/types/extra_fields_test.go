package types

import (
	"encoding/json"
	"testing"
)

func TestExtraFields_BasicOperations(t *testing.T) {
	ef := NewExtraFields()

	// 测试设置和获取字符串
	ef.Set("name", "John Doe")
	if val, ok := ef.GetString("name"); !ok || val != "John Doe" {
		t.Errorf("Expected 'John Doe', got '%s'", val)
	}

	// 测试设置和获取整数
	ef.Set("age", 30)
	if val, ok := ef.GetInt("age"); !ok || val != 30 {
		t.Errorf("Expected 30, got %d", val)
	}

	// 测试设置和获取布尔值
	ef.Set("active", true)
	if val, ok := ef.GetBool("active"); !ok || !val {
		t.Errorf("Expected true, got %v", val)
	}

	// 测试设置和获取浮点数
	ef.Set("price", 99.99)
	if val, ok := ef.GetFloat64("price"); !ok || val != 99.99 {
		t.Errorf("Expected 99.99, got %f", val)
	}

	// 测试Has方法
	if !ef.Has("name") {
		t.Error("Expected 'name' key to exist")
	}

	// 测试Len方法
	if ef.Len() != 4 {
		t.Errorf("Expected length 4, got %d", ef.Len())
	}

	// 测试Delete方法
	ef.Delete("age")
	if ef.Has("age") {
		t.Error("Expected 'age' key to be deleted")
	}
}

func TestExtraFields_ComplexTypes(t *testing.T) {
	ef := NewExtraFields()

	// 测试数组
	tags := []interface{}{"go", "database", "api"}
	ef.Set("tags", tags)
	if val, ok := ef.GetSlice("tags"); !ok || len(val) != 3 {
		t.Errorf("Expected slice with 3 elements, got %v", val)
	}

	// 测试对象
	metadata := map[string]interface{}{
		"version": "1.0",
		"author":  "Admin",
	}
	ef.Set("metadata", metadata)
	if val, ok := ef.GetMap("metadata"); !ok || val["version"] != "1.0" {
		t.Errorf("Expected map with version '1.0', got %v", val)
	}
}

func TestExtraFields_JSONSerialization(t *testing.T) {
	ef := NewExtraFields()
	ef.Set("name", "Test")
	ef.Set("count", 42)
	ef.Set("enabled", true)
	ef.Set("tags", []interface{}{"a", "b", "c"})

	// 序列化
	data, err := json.Marshal(ef)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// 反序列化
	var ef2 ExtraFields
	err = json.Unmarshal(data, &ef2)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// 验证数据
	if name, ok := ef2.GetString("name"); !ok || name != "Test" {
		t.Errorf("Expected 'Test', got '%s'", name)
	}

	if count, ok := ef2.GetInt("count"); !ok || count != 42 {
		t.Errorf("Expected 42, got %d", count)
	}

	if enabled, ok := ef2.GetBool("enabled"); !ok || !enabled {
		t.Errorf("Expected true, got %v", enabled)
	}
}

func TestExtraFields_DatabaseScan(t *testing.T) {
	ef := NewExtraFields()
	ef.Set("key1", "value1")
	ef.Set("key2", 123)

	// 模拟数据库Value操作
	val, err := ef.Value()
	if err != nil {
		t.Fatalf("Value() failed: %v", err)
	}

	// 模拟数据库Scan操作
	var ef2 ExtraFields
	err = ef2.Scan(val)
	if err != nil {
		t.Fatalf("Scan() failed: %v", err)
	}

	// 验证数据
	if str, ok := ef2.GetString("key1"); !ok || str != "value1" {
		t.Errorf("Expected 'value1', got '%s'", str)
	}

	if num, ok := ef2.GetInt("key2"); !ok || num != 123 {
		t.Errorf("Expected 123, got %d", num)
	}
}

func TestExtraFields_NilAndEmpty(t *testing.T) {
	// 测试空ExtraFields
	var ef ExtraFields

	// Value应该返回nil
	val, err := ef.Value()
	if err != nil {
		t.Fatalf("Value() failed: %v", err)
	}
	if val != nil {
		t.Errorf("Expected nil value for empty ExtraFields, got %v", val)
	}

	// Scan nil
	err = ef.Scan(nil)
	if err != nil {
		t.Fatalf("Scan(nil) failed: %v", err)
	}
}

func TestExtraFields_Clone(t *testing.T) {
	ef := NewExtraFields()
	ef.Set("key1", "value1")
	ef.Set("key2", 42)

	// 克隆
	clone := ef.Clone()

	// 修改原始对象
	ef.Set("key3", "value3")

	// 验证克隆对象不受影响
	if clone.Has("key3") {
		t.Error("Clone should not have key3")
	}

	if clone.Len() != 2 {
		t.Errorf("Expected clone length 2, got %d", clone.Len())
	}
}

func TestExtraFields_Merge(t *testing.T) {
	ef1 := NewExtraFields()
	ef1.Set("key1", "value1")
	ef1.Set("key2", "value2")

	ef2 := NewExtraFields()
	ef2.Set("key2", "new_value2")
	ef2.Set("key3", "value3")

	// 合并
	ef1.Merge(ef2)

	// 验证合并结果
	if val, ok := ef1.GetString("key2"); !ok || val != "new_value2" {
		t.Errorf("Expected 'new_value2', got '%s'", val)
	}

	if !ef1.Has("key3") {
		t.Error("Expected key3 to exist after merge")
	}

	if ef1.Len() != 3 {
		t.Errorf("Expected length 3, got %d", ef1.Len())
	}
}

func TestExtraFields_Clear(t *testing.T) {
	ef := NewExtraFields()
	ef.Set("key1", "value1")
	ef.Set("key2", "value2")

	ef.Clear()

	if !ef.IsEmpty() {
		t.Error("Expected ExtraFields to be empty after Clear()")
	}

	if ef.Len() != 0 {
		t.Errorf("Expected length 0, got %d", ef.Len())
	}
}

func TestExtraFields_Keys(t *testing.T) {
	ef := NewExtraFields()
	ef.Set("key1", "value1")
	ef.Set("key2", "value2")
	ef.Set("key3", "value3")

	keys := ef.Keys()
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
