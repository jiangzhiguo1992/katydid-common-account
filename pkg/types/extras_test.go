package types

import (
	"encoding/json"
	"fmt"
	"math"
	"runtime"
	"sync"
	"testing"
	"time"
)

// ============================================================================
// 基础功能测试
// ============================================================================

// TestNewExtras 测试创建新实例
func TestNewExtras(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
	}{
		{"零容量", 0},
		{"小容量", 5},
		{"中等容量", 50},
		{"大容量", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extras := NewExtras(tt.capacity)
			if extras == nil {
				t.Error("NewExtras 返回 nil")
			}
			if len(extras) != 0 {
				t.Errorf("新创建的 Extras 长度应为 0，实际为 %d", len(extras))
			}
		})
	}
}

// TestExtrasSet 测试设置操作
func TestExtrasSet(t *testing.T) {
	extras := NewExtras(0)

	// 测试基本设置
	extras.Set("string", "value")
	extras.Set("int", 42)
	extras.Set("float", 3.14)
	extras.Set("bool", true)
	extras.Set("nil", nil)

	if val, ok := extras.GetString("string"); !ok || val != "value" {
		t.Error("字符串设置失败")
	}
	if val, ok := extras.GetInt("int"); !ok || val != 42 {
		t.Error("整数设置失败")
	}
	if val, ok := extras.GetFloat64("float"); !ok || val != 3.14 {
		t.Error("浮点数设置失败")
	}
	if val, ok := extras.GetBool("bool"); !ok || val != true {
		t.Error("布尔值设置失败")
	}

	// 测试空键
	extras.Set("", "should_not_set")
	if _, ok := extras.Get(""); ok {
		t.Error("空键不应该被设置")
	}
}

// TestExtrasSetOrDel 测试条件设置/删除
func TestExtrasSetOrDel(t *testing.T) {
	extras := NewExtras(0)

	// 设置值
	extras.SetOrDel("key", "value")
	if val, ok := extras.GetString("key"); !ok || val != "value" {
		t.Error("SetOrDel 设置失败")
	}

	// 删除值
	extras.SetOrDel("key", nil)
	if _, ok := extras.Get("key"); ok {
		t.Error("SetOrDel 删除失败")
	}

	// 空键测试
	extras.SetOrDel("", "value")
	if len(extras) != 0 {
		t.Error("空键不应该被处理")
	}
}

// TestExtrasSetMultiple 测试批量设置
func TestExtrasSetMultiple(t *testing.T) {
	extras := NewExtras(0)

	pairs := map[string]any{
		"key1": "value1",
		"key2": 42,
		"key3": 3.14,
		"":     "should_ignore",
	}

	extras.SetMultiple(pairs)

	if val, ok := extras.GetString("key1"); !ok || val != "value1" {
		t.Error("批量设置 key1 失败")
	}
	if val, ok := extras.GetInt("key2"); !ok || val != 42 {
		t.Error("批量设置 key2 失败")
	}
	if val, ok := extras.GetFloat64("key3"); !ok || val != 3.14 {
		t.Error("批量设置 key3 失败")
	}
	if _, ok := extras.Get(""); ok {
		t.Error("空键不应该被设置")
	}
}

// TestExtrasSetPath 测试路径设置
func TestExtrasSetPath(t *testing.T) {
	extras := NewExtras(0)

	tests := []struct {
		name    string
		path    string
		value   any
		wantErr bool
	}{
		{"简单路径", "name", "Alice", false},
		{"嵌套路径", "user.name", "Bob", false},
		{"深层嵌套", "user.address.city", "Beijing", false},
		{"空路径", "", "value", true},
		{"路径以点结尾", "user.", "value", true},
		{"路径以点结尾", "user..", "value", true},
		{"路径中有空键", "user..name", "value", true},
		{"路径中有空键", "user..name..", "value", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := extras.SetPath(tt.path, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// 验证嵌套值
	if val, ok := extras.GetStringPath("user.name"); !ok || val != "Bob" {
		t.Error("嵌套路径设置失败")
	}
	if val, ok := extras.GetStringPath("user.address.city"); !ok || val != "Beijing" {
		t.Error("深层嵌套路径设置失败")
	}
}

// TestExtrasDelete 测试删除操作
func TestExtrasDelete(t *testing.T) {
	extras := NewExtras(0)
	extras.Set("key1", "value1")
	extras.Set("key2", "value2")

	extras.Delete("key1")
	if _, ok := extras.Get("key1"); ok {
		t.Error("Delete 失败")
	}
	if _, ok := extras.Get("key2"); !ok {
		t.Error("Delete 误删了其他键")
	}

	// 删除不存在的键
	extras.Delete("nonexistent")
	if len(extras) != 1 {
		t.Error("删除不存在的键改变了 map 大小")
	}
}

// TestExtrasDeleteMultiple 测试批量删除
func TestExtrasDeleteMultiple(t *testing.T) {
	extras := NewExtras(0)
	extras.Set("key1", "value1")
	extras.Set("key2", "value2")
	extras.Set("key3", "value3")

	extras.DeleteMultiple("key1", "key3")

	if _, ok := extras.Get("key1"); ok {
		t.Error("批量删除 key1 失败")
	}
	if _, ok := extras.Get("key3"); ok {
		t.Error("批量删除 key3 失败")
	}
	if _, ok := extras.Get("key2"); !ok {
		t.Error("批量删除误删了 key2")
	}
}

// TestExtrasClear 测试清空操作
func TestExtrasClear(t *testing.T) {
	extras := NewExtras(0)
	extras.Set("key1", "value1")
	extras.Set("key2", "value2")

	extras.Clear()

	if len(extras) != 0 {
		t.Errorf("Clear 后长度应为 0，实际为 %d", len(extras))
	}
}

// ============================================================================
// 类型转换测试
// ============================================================================

// TestExtrasGetString 测试字符串获取
func TestExtrasGetString(t *testing.T) {
	extras := NewExtras(0)

	tests := []struct {
		name   string
		key    string
		value  any
		want   string
		wantOk bool
	}{
		{"字符串", "str", "hello", "hello", true},
		{"整数", "int", 42, "", false},       // GetString 只支持原生 string 类型
		{"浮点数", "float", 3.14, "", false}, // 不会自动转换
		{"布尔值", "bool", true, "", false},  // 不会自动转换
		{"nil", "nil", nil, "", false},
		{"不存在", "nonexistent", nil, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.key != "nonexistent" {
				extras.Set(tt.key, tt.value)
			}
			got, ok := extras.GetString(tt.key)
			if ok != tt.wantOk {
				t.Errorf("GetString() ok = %v, wantOk %v", ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtrasGetInt 测试整数获取
func TestExtrasGetInt(t *testing.T) {
	extras := NewExtras(0)

	tests := []struct {
		name   string
		key    string
		value  any
		want   int
		wantOk bool
	}{
		{"int", "int", 42, 42, true},
		{"int8", "int8", int8(8), 8, true},
		{"int16", "int16", int16(16), 16, true},
		{"int32", "int32", int32(32), 32, true},
		{"int64", "int64", int64(64), 64, true},
		{"uint", "uint", uint(10), 10, true},
		{"float64", "float", 42.0, 42, true},
		{"字符串数字", "str", "42", 0, false}, // convertToInt 不支持字符串转换
		{"字符串非数字", "str_invalid", "abc", 0, false},
		// 在 64 位系统上 int 是 int64，所以 MaxInt64 可以转换
		{"溢出", "overflow", int64(math.MaxInt64), math.MaxInt64, true},
		{"大数值", "bignum", int64(math.MaxInt32), math.MaxInt32, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extras.Set(tt.key, tt.value)
			got, ok := extras.GetInt(tt.key)
			if ok != tt.wantOk {
				t.Errorf("GetInt() ok = %v, wantOk %v", ok, tt.wantOk)
			}
			if ok && got != tt.want {
				t.Errorf("GetInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtrasGetFloat64 测试浮点数获取
func TestExtrasGetFloat64(t *testing.T) {
	extras := NewExtras(0)

	tests := []struct {
		name   string
		key    string
		value  any
		want   float64
		wantOk bool
	}{
		{"float64", "f64", 3.14, 3.14, true},
		{"float32", "f32", float32(2.5), 2.5, true},
		{"int", "int", 42, 42.0, true},
		{"字符串数字", "str", "3.14", 0, false}, // convertToFloat64 不支持字符串转换
		{"字符串非数字", "str_invalid", "abc", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extras.Set(tt.key, tt.value)
			got, ok := extras.GetFloat64(tt.key)
			if ok != tt.wantOk {
				t.Errorf("GetFloat64() ok = %v, wantOk %v", ok, tt.wantOk)
			}
			if ok && got != tt.want {
				t.Errorf("GetFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtrasGetBool 测试布尔值获取
func TestExtrasGetBool(t *testing.T) {
	extras := NewExtras(0)

	tests := []struct {
		name   string
		key    string
		value  any
		want   bool
		wantOk bool
	}{
		{"true", "true", true, true, true},
		{"false", "false", false, false, true},
		{"字符串true", "str_true", "true", false, false}, // GetBool 只支持原生 bool 类型
		{"字符串false", "str_false", "false", false, false},
		{"整数1", "int1", 1, false, false}, // 不支持整数转布尔
		{"整数0", "int0", 0, false, false},
		{"字符串无效", "str_invalid", "abc", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extras.Set(tt.key, tt.value)
			got, ok := extras.GetBool(tt.key)
			if ok != tt.wantOk {
				t.Errorf("GetBool() ok = %v, wantOk %v", ok, tt.wantOk)
			}
			if ok && got != tt.want {
				t.Errorf("GetBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtrasGetSlice 测试切片获取
func TestExtrasGetSlice(t *testing.T) {
	extras := NewExtras(0)

	slice := []any{1, "two", 3.0, true}
	extras.Set("slice", slice)

	got, ok := extras.GetSlice("slice")
	if !ok {
		t.Fatal("GetSlice 失败")
	}
	if len(got) != len(slice) {
		t.Errorf("切片长度不匹配: got %d, want %d", len(got), len(slice))
	}
}

// TestExtrasGetStringSlice 测试字符串切片获取
func TestExtrasGetStringSlice(t *testing.T) {
	extras := NewExtras(0)

	tests := []struct {
		name   string
		value  any
		want   []string
		wantOk bool
	}{
		{"字符串切片", []string{"a", "b", "c"}, []string{"a", "b", "c"}, true},
		{"any切片", []any{"x", "y", "z"}, []string{"x", "y", "z"}, true},
		{"混合类型", []any{1, "two", 3.0}, nil, false}, // GetStringSlice 不支持混合类型自动转换
		{"非切片", "not_a_slice", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extras.Set("key", tt.value)
			got, ok := extras.GetStringSlice("key")
			if ok != tt.wantOk {
				t.Errorf("GetStringSlice() ok = %v, wantOk %v", ok, tt.wantOk)
			}
			if ok && len(got) != len(tt.want) {
				t.Errorf("GetStringSlice() len = %d, want %d", len(got), len(tt.want))
			}
		})
	}
}

// TestExtrasGetMap 测试 map 获取
func TestExtrasGetMap(t *testing.T) {
	extras := NewExtras(0)

	m := map[string]any{"key": "value", "num": 42}
	extras.Set("map", m)

	got, ok := extras.GetMap("map")
	if !ok {
		t.Fatal("GetMap 失败")
	}
	if len(got) != len(m) {
		t.Errorf("map 长度不匹配: got %d, want %d", len(got), len(m))
	}
}

// TestExtrasGetExtras 测试嵌套 Extras 获取
func TestExtrasGetExtras(t *testing.T) {
	extras := NewExtras(0)

	nested := NewExtras(0)
	nested.Set("inner", "value")
	extras.Set("nested", nested)

	got, ok := extras.GetExtras("nested")
	if !ok {
		t.Fatal("GetExtras 失败")
	}
	if val, ok := got.GetString("inner"); !ok || val != "value" {
		t.Error("嵌套 Extras 值不正确")
	}
}

// ============================================================================
// 路径操作测试
// ============================================================================

// TestExtrasGetPath 测试路径获取
func TestExtrasGetPath(t *testing.T) {
	extras := NewExtras(0)

	// 构建嵌套结构
	user := NewExtras(0)
	user.Set("name", "Alice")
	user.Set("age", 30)

	address := NewExtras(0)
	address.Set("city", "Beijing")
	address.Set("zip", "100000")
	user.Set("address", address)

	extras.Set("user", user)

	tests := []struct {
		name   string
		path   string
		want   any
		wantOk bool
	}{
		{"简单路径", "user", user, true},
		{"嵌套路径", "user.name", "Alice", true},
		{"深层路径", "user.address.city", "Beijing", true},
		{"不存在路径", "user.nonexistent", nil, false},
		{"空路径", "", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extras.GetPath(tt.path)
			if ok != tt.wantOk {
				t.Errorf("GetPath() ok = %v, wantOk %v", ok, tt.wantOk)
			}
			if ok && tt.want != nil {
				// 简单比较
				if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", tt.want) {
					t.Errorf("GetPath() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// ============================================================================
// 工具方法测试
// ============================================================================

// TestExtrasHas 测试键存在性检查
func TestExtrasHas(t *testing.T) {
	extras := NewExtras(0)
	extras.Set("exists", "value")
	extras.Set("nil", nil)

	if !extras.Has("exists") {
		t.Error("Has('exists') 应返回 true")
	}
	if !extras.Has("nil") {
		t.Error("Has('nil') 应返回 true（nil 值也存在）")
	}
	if extras.Has("nonexistent") {
		t.Error("Has('nonexistent') 应返回 false")
	}
}

// TestExtrasHasAll 测试多键存在性检查
func TestExtrasHasAll(t *testing.T) {
	extras := NewExtras(0)
	extras.Set("key1", "value1")
	extras.Set("key2", "value2")

	if !extras.HasAll("key1", "key2") {
		t.Error("HasAll 应返回 true")
	}
	if extras.HasAll("key1", "nonexistent") {
		t.Error("HasAll 应返回 false")
	}
}

// TestExtrasHasAny 测试任意键存在性检查
func TestExtrasHasAny(t *testing.T) {
	extras := NewExtras(0)
	extras.Set("key1", "value1")

	if !extras.HasAny("key1", "key2") {
		t.Error("HasAny 应返回 true")
	}
	if extras.HasAny("nonexistent1", "nonexistent2") {
		t.Error("HasAny 应返回 false")
	}
}

// TestExtrasIsNil 测试 nil 值检查
func TestExtrasIsNil(t *testing.T) {
	extras := NewExtras(0)
	extras.Set("nil", nil)
	extras.Set("value", "not_nil")

	if !extras.IsNil("nil") {
		t.Error("IsNil('nil') 应返回 true")
	}
	if extras.IsNil("value") {
		t.Error("IsNil('value') 应返回 false")
	}
	if extras.IsNil("nonexistent") {
		t.Error("IsNil('nonexistent') 应返回 false（键不存在）")
	}
}

// TestExtrasIsEmpty 测试空检查
func TestExtrasIsEmpty(t *testing.T) {
	extras := NewExtras(0)

	if !extras.IsEmpty() {
		t.Error("新建的 Extras 应该为空")
	}

	extras.Set("key", "value")
	if extras.IsEmpty() {
		t.Error("设置值后 Extras 不应为空")
	}

	extras.Clear()
	if !extras.IsEmpty() {
		t.Error("清空后 Extras 应该为空")
	}
}

// TestExtrasKeys 测试获取所有键
func TestExtrasKeys(t *testing.T) {
	extras := NewExtras(0)
	extras.Set("key1", "value1")
	extras.Set("key2", "value2")
	extras.Set("key3", "value3")

	keys := extras.Keys()
	if len(keys) != 3 {
		t.Errorf("Keys() 长度应为 3，实际为 %d", len(keys))
	}

	// 检查所有键都存在
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}
	if !keyMap["key1"] || !keyMap["key2"] || !keyMap["key3"] {
		t.Error("Keys() 返回的键不完整")
	}
}

// TestExtrasLen 测试长度获取
func TestExtrasLen(t *testing.T) {
	extras := NewExtras(0)

	if extras.Len() != 0 {
		t.Error("新建的 Extras 长度应为 0")
	}

	extras.Set("key1", "value1")
	extras.Set("key2", "value2")

	if extras.Len() != 2 {
		t.Errorf("Len() 应为 2，实际为 %d", extras.Len())
	}
}

// ============================================================================
// 复制和合并测试
// ============================================================================

// TestExtrasClone 测试浅拷贝
func TestExtrasClone(t *testing.T) {
	original := NewExtras(0)
	original.Set("string", "value")
	original.Set("int", 42)

	nested := NewExtras(0)
	nested.Set("inner", "nested_value")
	original.Set("nested", nested)

	cloned := original.Clone()

	// 验证值相同
	if val, ok := cloned.GetString("string"); !ok || val != "value" {
		t.Error("Clone 后字符串值不正确")
	}

	// 修改克隆不应影响原始
	cloned.Set("string", "modified")
	if val, _ := original.GetString("string"); val == "modified" {
		t.Error("Clone 后修改影响了原始对象")
	}

	// 浅拷贝：修改嵌套对象会影响原始
	if nestedCloned, ok := cloned.GetExtras("nested"); ok {
		nestedCloned.Set("inner", "modified_nested")
		if nestedOriginal, ok := original.GetExtras("nested"); ok {
			if val, _ := nestedOriginal.GetString("inner"); val != "modified_nested" {
				t.Error("浅拷贝应共享嵌套对象")
			}
		}
	}
}

// TestExtrasDeepClone 测试深拷贝
func TestExtrasDeepClone(t *testing.T) {
	original := NewExtras(0)
	original.Set("string", "value")

	nested := NewExtras(0)
	nested.Set("inner", "nested_value")
	original.Set("nested", nested)

	cloned, err := original.DeepClone()
	if err != nil {
		t.Fatalf("DeepClone 失败: %v", err)
	}

	// 修改嵌套对象不应影响原始
	if nestedCloned, ok := cloned.GetExtras("nested"); ok {
		nestedCloned.Set("inner", "modified_nested")
		if nestedOriginal, ok := original.GetExtras("nested"); ok {
			if val, _ := nestedOriginal.GetString("inner"); val == "modified_nested" {
				t.Error("深拷贝不应共享嵌套对象")
			}
		}
	}
}

// TestExtrasCopyTo 测试复制到目标
func TestExtrasCopyTo(t *testing.T) {
	source := NewExtras(0)
	source.Set("key1", "value1")
	source.Set("key2", "value2")

	target := NewExtras(0)
	target.Set("key3", "value3")

	source.CopyTo(target)

	if !target.Has("key1") || !target.Has("key2") {
		t.Error("CopyTo 没有复制所有键")
	}
	if !target.Has("key3") {
		t.Error("CopyTo 删除了目标已有的键")
	}
}

// TestExtrasMerge 测试合并
func TestExtrasMerge(t *testing.T) {
	extras1 := NewExtras(0)
	extras1.Set("key1", "value1")
	extras1.Set("common", "original")

	extras2 := NewExtras(0)
	extras2.Set("key2", "value2")
	extras2.Set("common", "override")

	extras1.Merge(extras2)

	if val, _ := extras1.GetString("common"); val != "override" {
		t.Error("Merge 应该覆盖相同的键")
	}
	if !extras1.Has("key1") || !extras1.Has("key2") {
		t.Error("Merge 后应包含所有键")
	}
}

// ============================================================================
// JSON 序列化测试
// ============================================================================

// TestExtrasMarshalJSON 测试 JSON 序列化
func TestExtrasMarshalJSON(t *testing.T) {
	extras := NewExtras(0)
	extras.Set("string", "value")
	extras.Set("int", 42)
	extras.Set("float", 3.14)
	extras.Set("bool", true)
	extras.Set("nil", nil)

	data, err := json.Marshal(extras)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	// 验证可以反序列化
	var decoded Extras
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	if val, _ := decoded.GetString("string"); val != "value" {
		t.Error("反序列化后字符串值不正确")
	}
	if val, _ := decoded.GetFloat64("int"); val != 42 {
		t.Error("反序列化后整数值不正确")
	}
}

// TestExtrasUnmarshalJSON 测试 JSON 反序列化
func TestExtrasUnmarshalJSON(t *testing.T) {
	jsonData := `{"name":"Alice","age":30,"active":true}`

	var extras Extras
	err := json.Unmarshal([]byte(jsonData), &extras)
	if err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	if val, _ := extras.GetString("name"); val != "Alice" {
		t.Error("反序列化后 name 值不正确")
	}
	if val, _ := extras.GetFloat64("age"); val != 30 {
		t.Error("反序列化后 age 值不正确")
	}
	if val, _ := extras.GetBool("active"); !val {
		t.Error("反序列化后 active 值不正确")
	}
}

// TestExtrasNilJSON 测试 nil 的 JSON 处理
func TestExtrasNilJSON(t *testing.T) {
	var extras Extras

	// nil Extras 应该序列化为空对象 {}
	data, err := json.Marshal(extras)
	if err != nil {
		t.Fatalf("nil Extras 序列化失败: %v", err)
	}
	if string(data) != "{}" {
		t.Errorf("nil Extras 应序列化为 '{}'，实际为 %s", string(data))
	}
}

// ============================================================================
// 数据库操作测试
// ============================================================================

// TestExtrasValue 测试数据库 Value 方法
func TestExtrasValue(t *testing.T) {
	extras := NewExtras(0)
	extras.Set("key", "value")

	val, err := extras.Value()
	if err != nil {
		t.Fatalf("Value() 失败: %v", err)
	}

	if val == nil {
		t.Error("Value() 不应返回 nil")
	}
}

// TestExtrasScan 测试数据库 Scan 方法
func TestExtrasScan(t *testing.T) {
	jsonData := []byte(`{"name":"Alice","age":30}`)

	var extras Extras
	err := extras.Scan(jsonData)
	if err != nil {
		t.Fatalf("Scan() 失败: %v", err)
	}

	if val, _ := extras.GetString("name"); val != "Alice" {
		t.Error("Scan 后 name 值不正确")
	}

	// 测试 nil 输入
	var nilExtras Extras
	err = nilExtras.Scan(nil)
	if err != nil {
		t.Errorf("Scan(nil) 应该成功: %v", err)
	}
}

// ============================================================================
// 边界条件测试
// ============================================================================

// TestExtrasEdgeCases 测试边界情况
func TestExtrasEdgeCases(t *testing.T) {
	t.Run("nil Extras 操作", func(t *testing.T) {
		var extras Extras

		// nil Extras 的操作应该安全
		if !extras.IsEmpty() {
			t.Error("nil Extras 应该为空")
		}
		if extras.Len() != 0 {
			t.Error("nil Extras 长度应为 0")
		}
		if extras.Has("key") {
			t.Error("nil Extras 不应有任何键")
		}
	})

	t.Run("空字符串键", func(t *testing.T) {
		extras := NewExtras(0)
		extras.Set("", "value")
		if len(extras) != 0 {
			t.Error("空字符串键不应被设置")
		}
	})

	t.Run("大数值转换", func(t *testing.T) {
		extras := NewExtras(0)
		extras.Set("max_int64", int64(math.MaxInt64))
		extras.Set("min_int64", int64(math.MinInt64))

		if val, ok := extras.GetInt64("max_int64"); !ok || val != math.MaxInt64 {
			t.Error("MaxInt64 转换失败")
		}
		if val, ok := extras.GetInt64("min_int64"); !ok || val != math.MinInt64 {
			t.Error("MinInt64 转换失败")
		}
	})

	t.Run("特殊浮点数", func(t *testing.T) {
		extras := NewExtras(0)
		extras.Set("inf", math.Inf(1))
		extras.Set("nan", math.NaN())

		if val, ok := extras.GetFloat64("inf"); !ok || !math.IsInf(val, 1) {
			t.Error("Inf 转换失败")
		}
		if val, ok := extras.GetFloat64("nan"); !ok || !math.IsNaN(val) {
			t.Error("NaN 转换失败")
		}
	})
}

// ============================================================================
// 百万级性能测试 - Set 操作
// ============================================================================

// BenchmarkExtrasSet_1M 百万次 Set 操作基准测试
func BenchmarkExtrasSet_1M(b *testing.B) {
	const iterations = 1000000

	b.Run("Sequential", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			extras := NewExtras(iterations)
			b.ResetTimer()
			for i := 0; i < iterations; i++ {
				extras.Set(fmt.Sprintf("key_%d", i), i)
			}
			b.StopTimer()
		}
	})

	b.Run("SameKey", func(b *testing.B) {
		extras := NewExtras(1)
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			for i := 0; i < iterations; i++ {
				extras.Set("key", i)
			}
		}
	})

	b.Run("PreAllocated", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			extras := NewExtras(iterations)
			b.ResetTimer()
			for i := 0; i < iterations; i++ {
				extras.Set(fmt.Sprintf("key_%d", i), i)
			}
			b.StopTimer()
		}
	})
}

// BenchmarkExtrasGet_1M 百万次 Get 操作基准测试
func BenchmarkExtrasGet_1M(b *testing.B) {
	const iterations = 1000000

	// 准备数据
	extras := NewExtras(iterations)
	for i := 0; i < iterations; i++ {
		extras.Set(fmt.Sprintf("key_%d", i), i)
	}

	b.Run("Sequential", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			for i := 0; i < iterations; i++ {
				_, _ = extras.Get(fmt.Sprintf("key_%d", i))
			}
		}
	})

	b.Run("SameKey", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			for i := 0; i < iterations; i++ {
				_, _ = extras.Get("key_500000")
			}
		}
	})

	b.Run("NotFound", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			for i := 0; i < iterations; i++ {
				_, _ = extras.Get("nonexistent")
			}
		}
	})
}

// BenchmarkExtrasGetInt_1M 百万次类型转换基准测试
func BenchmarkExtrasGetInt_1M(b *testing.B) {
	const iterations = 1000000

	extras := NewExtras(iterations)
	for i := 0; i < iterations; i++ {
		extras.Set(fmt.Sprintf("key_%d", i), i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i := 0; i < iterations; i++ {
			_, _ = extras.GetInt(fmt.Sprintf("key_%d", i))
		}
	}
}

// BenchmarkExtrasGetString_1M 百万次字符串转换基准测试
func BenchmarkExtrasGetString_1M(b *testing.B) {
	const iterations = 1000000

	extras := NewExtras(iterations)
	for i := 0; i < iterations; i++ {
		extras.Set(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i := 0; i < iterations; i++ {
			_, _ = extras.GetString(fmt.Sprintf("key_%d", i))
		}
	}
}

// ============================================================================
// 百万级性能测试 - JSON 序列化
// ============================================================================

// BenchmarkExtrasJSONMarshal_1M 百万次 JSON 序列化基准测试
func BenchmarkExtrasJSONMarshal_1M(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			extras := NewExtras(size)
			for i := 0; i < size; i++ {
				extras.Set(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))
			}

			b.ReportAllocs()
			b.ResetTimer()
			iterations := 1000000 / size
			for n := 0; n < b.N; n++ {
				for i := 0; i < iterations; i++ {
					_, _ = json.Marshal(extras)
				}
			}
		})
	}
}

// BenchmarkExtrasJSONUnmarshal_1M 百万次 JSON 反序列化基准测试
func BenchmarkExtrasJSONUnmarshal_1M(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			extras := NewExtras(size)
			for i := 0; i < size; i++ {
				extras.Set(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))
			}
			data, _ := json.Marshal(extras)

			b.ReportAllocs()
			b.ResetTimer()
			iterations := 1000000 / size
			for n := 0; n < b.N; n++ {
				for i := 0; i < iterations; i++ {
					var result Extras
					_ = json.Unmarshal(data, &result)
				}
			}
		})
	}
}

// ============================================================================
// 百万级性能测试 - 批量操作
// ============================================================================

// BenchmarkExtrasSetMultiple_1M 百万次批量设置基准测试
func BenchmarkExtrasSetMultiple_1M(b *testing.B) {
	const batchSize = 100
	const iterations = 10000

	pairs := make(map[string]any, batchSize)
	for i := 0; i < batchSize; i++ {
		pairs[fmt.Sprintf("key_%d", i)] = i
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		extras := NewExtras(batchSize * iterations)
		for i := 0; i < iterations; i++ {
			extras.SetMultiple(pairs)
		}
	}
}

// BenchmarkExtrasClone_1M 百万元素克隆基准测试
func BenchmarkExtrasClone_1M(b *testing.B) {
	const size = 1000000

	extras := NewExtras(size)
	for i := 0; i < size; i++ {
		extras.Set(fmt.Sprintf("key_%d", i), i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = extras.Clone()
	}
}

// BenchmarkExtrasMerge_1M 百万元素合并基准测试
func BenchmarkExtrasMerge_1M(b *testing.B) {
	const size = 500000

	extras1 := NewExtras(size)
	extras2 := NewExtras(size)
	for i := 0; i < size; i++ {
		extras1.Set(fmt.Sprintf("key1_%d", i), i)
		extras2.Set(fmt.Sprintf("key2_%d", i), i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		temp := extras1.Clone()
		temp.Merge(extras2)
	}
}

// ============================================================================
// 百万级性能测试 - 路径操作
// ============================================================================

// BenchmarkExtrasSetPath_1M 百万次路径设置基准测试
func BenchmarkExtrasSetPath_1M(b *testing.B) {
	const iterations = 100000

	b.Run("SingleLevel", func(b *testing.B) {
		extras := NewExtras(iterations)
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			for i := 0; i < iterations; i++ {
				_ = extras.SetPath(fmt.Sprintf("key_%d", i), i)
			}
		}
	})

	b.Run("TwoLevels", func(b *testing.B) {
		extras := NewExtras(0)
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			for i := 0; i < iterations; i++ {
				_ = extras.SetPath(fmt.Sprintf("level1.key_%d", i), i)
			}
		}
	})

	b.Run("ThreeLevels", func(b *testing.B) {
		extras := NewExtras(0)
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			for i := 0; i < iterations; i++ {
				_ = extras.SetPath(fmt.Sprintf("level1.level2.key_%d", i), i)
			}
		}
	})
}

// BenchmarkExtrasGetPath_1M 百万次路径获取基准测试
func BenchmarkExtrasGetPath_1M(b *testing.B) {
	const iterations = 100000

	extras := NewExtras(0)
	for i := 0; i < iterations; i++ {
		_ = extras.SetPath(fmt.Sprintf("level1.level2.key_%d", i), i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i := 0; i < iterations; i++ {
			_, _ = extras.GetPath(fmt.Sprintf("level1.level2.key_%d", i))
		}
	}
}

// ============================================================================
// 内存占用分析测试
// ============================================================================

// TestExtrasMemoryFootprint 测试内存占用
func TestExtrasMemoryFootprint(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过内存测试")
	}

	sizes := []int{100, 1000, 10000, 100000, 1000000}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("Size_%d", size), func(t *testing.T) {
			runtime.GC()
			var m1 runtime.MemStats
			runtime.ReadMemStats(&m1)

			extras := NewExtras(size)
			for i := 0; i < size; i++ {
				extras.Set(fmt.Sprintf("key_%d", i), i)
			}

			runtime.GC()
			var m2 runtime.MemStats
			runtime.ReadMemStats(&m2)

			allocated := m2.Alloc - m1.Alloc
			perItem := float64(allocated) / float64(size)

			t.Logf("大小: %d, 总内存: %.2f MB, 每项: %.2f bytes",
				size, float64(allocated)/(1024*1024), perItem)
		})
	}
}

// ============================================================================
// 并发安全测试（需要外部同步）
// ============================================================================

// TestExtrasConcurrentReadUnsafe 测试并发读取（不安全，用于演示）
func TestExtrasConcurrentReadUnsafe(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过并发测试")
	}

	extras := NewExtras(100)
	for i := 0; i < 100; i++ {
		extras.Set(fmt.Sprintf("key_%d", i), i)
	}

	var wg sync.WaitGroup
	readers := 10
	iterations := 10000

	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				_, _ = extras.Get(fmt.Sprintf("key_%d", i%100))
			}
		}()
	}

	wg.Wait()
}

// TestExtrasConcurrentWithMutex 测试使用互斥锁的并发访问
func TestExtrasConcurrentWithMutex(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过并发测试")
	}

	extras := NewExtras(100)
	var mu sync.RWMutex

	for i := 0; i < 100; i++ {
		extras.Set(fmt.Sprintf("key_%d", i), i)
	}

	var wg sync.WaitGroup
	readers := 8
	writers := 2
	iterations := 1000

	// 读协程
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				mu.RLock()
				_, _ = extras.Get(fmt.Sprintf("key_%d", i%100))
				mu.RUnlock()
			}
		}()
	}

	// 写协程
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				mu.Lock()
				extras.Set(fmt.Sprintf("writer_%d_key_%d", id, i), i)
				mu.Unlock()
			}
		}(w)
	}

	wg.Wait()
}

// ============================================================================
// 综合性能测试报告
// ============================================================================

// TestExtrasPerformanceReport 生成性能测试报告
func TestExtrasPerformanceReport(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过性能报告测试")
	}

	t.Log("\n========================================")
	t.Log("Extras 性能测试报告")
	t.Log("========================================\n")

	// 1. Set 操作性能
	t.Run("Set性能", func(t *testing.T) {
		sizes := []int{1000, 10000, 100000, 1000000}
		for _, size := range sizes {
			extras := NewExtras(size)
			start := time.Now()
			for i := 0; i < size; i++ {
				extras.Set(fmt.Sprintf("key_%d", i), i)
			}
			duration := time.Since(start)
			t.Logf("Set %d 项: %v (%.0f ops/s)",
				size, duration, float64(size)/duration.Seconds())
		}
	})

	// 2. Get 操作性能
	t.Run("Get性能", func(t *testing.T) {
		extras := NewExtras(1000000)
		for i := 0; i < 1000000; i++ {
			extras.Set(fmt.Sprintf("key_%d", i), i)
		}

		start := time.Now()
		for i := 0; i < 1000000; i++ {
			_, _ = extras.Get(fmt.Sprintf("key_%d", i))
		}
		duration := time.Since(start)
		t.Logf("Get 1M 项: %v (%.0f ops/s)",
			duration, 1000000/duration.Seconds())
	})

	// 3. 类型转换性能
	t.Run("类型转换性能", func(t *testing.T) {
		extras := NewExtras(100000)
		for i := 0; i < 100000; i++ {
			extras.Set(fmt.Sprintf("key_%d", i), i)
		}

		start := time.Now()
		for i := 0; i < 100000; i++ {
			_, _ = extras.GetInt(fmt.Sprintf("key_%d", i))
		}
		duration := time.Since(start)
		t.Logf("GetInt 100K 项: %v (%.0f ops/s)",
			duration, 100000/duration.Seconds())
	})

	// 4. JSON 序列化性能
	t.Run("JSON序列化性能", func(t *testing.T) {
		sizes := []int{10, 100, 1000, 10000}
		for _, size := range sizes {
			extras := NewExtras(size)
			for i := 0; i < size; i++ {
				extras.Set(fmt.Sprintf("key_%d", i), i)
			}

			start := time.Now()
			iterations := 1000
			for i := 0; i < iterations; i++ {
				_, _ = json.Marshal(extras)
			}
			duration := time.Since(start)
			t.Logf("Marshal %d 项 x %d 次: %v (%.0f ops/s)",
				size, iterations, duration, float64(iterations)/duration.Seconds())
		}
	})

	// 5. Clone 性能
	t.Run("Clone性能", func(t *testing.T) {
		sizes := []int{1000, 10000, 100000}
		for _, size := range sizes {
			extras := NewExtras(size)
			for i := 0; i < size; i++ {
				extras.Set(fmt.Sprintf("key_%d", i), i)
			}

			start := time.Now()
			iterations := 100
			for i := 0; i < iterations; i++ {
				_ = extras.Clone()
			}
			duration := time.Since(start)
			t.Logf("Clone %d 项 x %d 次: %v (%.2f ms/op)",
				size, iterations, duration, duration.Seconds()*1000/float64(iterations))
		}
	})

	// 6. 路径操作性能
	t.Run("路径操作性能", func(t *testing.T) {
		extras := NewExtras(0)
		iterations := 10000

		start := time.Now()
		for i := 0; i < iterations; i++ {
			_ = extras.SetPath(fmt.Sprintf("level1.level2.key_%d", i), i)
		}
		setDuration := time.Since(start)

		start = time.Now()
		for i := 0; i < iterations; i++ {
			_, _ = extras.GetPath(fmt.Sprintf("level1.level2.key_%d", i))
		}
		getDuration := time.Since(start)

		t.Logf("SetPath %d 项: %v (%.0f ops/s)", iterations, setDuration, float64(iterations)/setDuration.Seconds())
		t.Logf("GetPath %d 项: %v (%.0f ops/s)", iterations, getDuration, float64(iterations)/getDuration.Seconds())
	})

	t.Log("\n========================================")
	t.Log("性能测试报告完成")
	t.Log("========================================\n")
}

// ============================================================================
// 压力测试
// ============================================================================

// TestExtrasStressTest 压力测试
func TestExtrasStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	t.Run("大规模数据写入", func(t *testing.T) {
		const size = 2000000 // 200万
		extras := NewExtras(size)

		start := time.Now()
		for i := 0; i < size; i++ {
			extras.Set(fmt.Sprintf("key_%d", i), i)
		}
		duration := time.Since(start)

		t.Logf("写入 %d 项耗时: %v (%.0f ops/s)",
			size, duration, float64(size)/duration.Seconds())

		if extras.Len() != size {
			t.Errorf("长度不匹配: got %d, want %d", extras.Len(), size)
		}
	})

	t.Run("大规模随机读取", func(t *testing.T) {
		const size = 1000000
		extras := NewExtras(size)
		for i := 0; i < size; i++ {
			extras.Set(fmt.Sprintf("key_%d", i), i)
		}

		start := time.Now()
		for i := 0; i < size; i++ {
			key := fmt.Sprintf("key_%d", (i*7919)%size) // 伪随机
			_, ok := extras.Get(key)
			if !ok {
				t.Errorf("键 %s 不存在", key)
			}
		}
		duration := time.Since(start)

		t.Logf("随机读取 %d 项耗时: %v (%.0f ops/s)",
			size, duration, float64(size)/duration.Seconds())
	})

	t.Run("混合读写操作", func(t *testing.T) {
		const operations = 1000000
		extras := NewExtras(0)

		start := time.Now()
		for i := 0; i < operations; i++ {
			if i%2 == 0 {
				extras.Set(fmt.Sprintf("key_%d", i), i)
			} else {
				_, _ = extras.Get(fmt.Sprintf("key_%d", i-1))
			}
		}
		duration := time.Since(start)

		t.Logf("混合操作 %d 次耗时: %v (%.0f ops/s)",
			operations, duration, float64(operations)/duration.Seconds())
	})
}
