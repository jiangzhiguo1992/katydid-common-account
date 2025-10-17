package idgen

import (
	"encoding/json"
	"testing"
)

// TestNewID 测试创建ID
func TestNewID(t *testing.T) {
	id := NewID(12345)
	if id.Int64() != 12345 {
		t.Errorf("ID值不匹配，期望12345，得到%d", id.Int64())
	}
}

// TestIDString 测试ID转字符串
func TestIDString(t *testing.T) {
	tests := []struct {
		name     string
		id       ID
		expected string
	}{
		{"正数", NewID(12345), "12345"},
		{"零", NewID(0), "0"},
		{"大数", NewID(9223372036854775807), "9223372036854775807"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.id.String()
			if result != tt.expected {
				t.Errorf("String() = %s, 期望 %s", result, tt.expected)
			}
		})
	}
}

// TestIDHex 测试ID转十六进制
func TestIDHex(t *testing.T) {
	id := NewID(255)
	hex := id.Hex()
	expected := "0xff"
	if hex != expected {
		t.Errorf("Hex() = %s, 期望 %s", hex, expected)
	}
}

// TestIDBinary 测试ID转二进制
func TestIDBinary(t *testing.T) {
	id := NewID(5)
	binary := id.Binary()
	expected := "0b101"
	if binary != expected {
		t.Errorf("Binary() = %s, 期望 %s", binary, expected)
	}
}

// TestIDMarshalJSON 测试JSON序列化
func TestIDMarshalJSON(t *testing.T) {
	id := NewID(12345)
	data, err := json.Marshal(id)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	// 应该序列化为字符串
	expected := `"12345"`
	if string(data) != expected {
		t.Errorf("MarshalJSON() = %s, 期望 %s", string(data), expected)
	}
}

// TestIDUnmarshalJSON 测试JSON反序列化
func TestIDUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected ID
		wantErr  bool
	}{
		{
			name:     "从字符串反序列化",
			jsonData: `"12345"`,
			expected: NewID(12345),
			wantErr:  false,
		},
		{
			name:     "从数字反序列化",
			jsonData: `67890`,
			expected: NewID(67890),
			wantErr:  false,
		},
		{
			name:     "无效JSON",
			jsonData: `invalid`,
			wantErr:  true,
		},
		{
			name:     "无效字符串",
			jsonData: `"abc"`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id ID
			err := json.Unmarshal([]byte(tt.jsonData), &id)

			if tt.wantErr {
				if err == nil {
					t.Error("期望得到错误，但没有返回错误")
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
					return
				}
				if id != tt.expected {
					t.Errorf("UnmarshalJSON() = %d, 期望 %d", id, tt.expected)
				}
			}
		})
	}
}

// TestIDIsZero 测试检查零值
func TestIDIsZero(t *testing.T) {
	tests := []struct {
		name     string
		id       ID
		expected bool
	}{
		{"零值", NewID(0), true},
		{"正数", NewID(1), false},
		{"负数", NewID(-1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.id.IsZero()
			if result != tt.expected {
				t.Errorf("IsZero() = %v, 期望 %v", result, tt.expected)
			}
		})
	}
}

// TestIDIsValid 测试检查有效性
func TestIDIsValid(t *testing.T) {
	tests := []struct {
		name     string
		id       ID
		expected bool
	}{
		{"正数_有效", NewID(1), true},
		{"零_无效", NewID(0), false},
		{"负数_无效", NewID(-1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.id.IsValid()
			if result != tt.expected {
				t.Errorf("IsValid() = %v, 期望 %v", result, tt.expected)
			}
		})
	}
}

// TestIDParse 测试解析ID
func TestIDParse(t *testing.T) {
	// 创建一个有效的Snowflake ID
	sf, _ := NewSnowflake(5, 10)
	rawID, _ := sf.NextID()
	id := NewID(rawID)

	info, err := id.Parse()
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if info.DatacenterID != 5 {
		t.Errorf("DatacenterID = %d, 期望 5", info.DatacenterID)
	}
	if info.WorkerID != 10 {
		t.Errorf("WorkerID = %d, 期望 10", info.WorkerID)
	}
}

// TestParseID 测试从字符串解析ID
func TestParseID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ID
		wantErr  bool
	}{
		{
			name:     "十进制",
			input:    "12345",
			expected: NewID(12345),
			wantErr:  false,
		},
		{
			name:     "十六进制",
			input:    "0xFF",
			expected: NewID(255),
			wantErr:  false,
		},
		{
			name:     "二进制",
			input:    "0b101",
			expected: NewID(5),
			wantErr:  false,
		},
		{
			name:     "带空格",
			input:    "  123  ",
			expected: NewID(123),
			wantErr:  false,
		},
		{
			name:    "空字符串",
			input:   "",
			wantErr: true,
		},
		{
			name:    "无效字符",
			input:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ParseID(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("期望得到错误，但没有返回错误")
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
					return
				}
				if id != tt.expected {
					t.Errorf("ParseID() = %d, 期望 %d", id, tt.expected)
				}
			}
		})
	}
}

// TestIDSliceInt64Slice 测试转换为int64切片
func TestIDSliceInt64Slice(t *testing.T) {
	ids := IDSlice{NewID(1), NewID(2), NewID(3)}
	int64Slice := ids.Int64Slice()

	if len(int64Slice) != 3 {
		t.Errorf("长度不匹配，期望3，得到%d", len(int64Slice))
	}

	for i, v := range int64Slice {
		if v != int64(i+1) {
			t.Errorf("索引%d: 期望%d, 得到%d", i, i+1, v)
		}
	}
}

// TestIDSliceStringSlice 测试转换为字符串切片
func TestIDSliceStringSlice(t *testing.T) {
	ids := IDSlice{NewID(1), NewID(2), NewID(3)}
	stringSlice := ids.StringSlice()

	expected := []string{"1", "2", "3"}
	if len(stringSlice) != len(expected) {
		t.Errorf("长度不匹配，期望%d，得到%d", len(expected), len(stringSlice))
	}

	for i, v := range stringSlice {
		if v != expected[i] {
			t.Errorf("索引%d: 期望%s, 得到%s", i, expected[i], v)
		}
	}
}

// TestIDSliceContains 测试检查是否包含
func TestIDSliceContains(t *testing.T) {
	ids := IDSlice{NewID(1), NewID(2), NewID(3)}

	if !ids.Contains(NewID(2)) {
		t.Error("应该包含ID 2")
	}

	if ids.Contains(NewID(4)) {
		t.Error("不应该包含ID 4")
	}
}

// TestIDSliceDeduplicate 测试去重
func TestIDSliceDeduplicate(t *testing.T) {
	ids := IDSlice{NewID(1), NewID(2), NewID(2), NewID(3), NewID(1)}
	unique := ids.Deduplicate()

	if len(unique) != 3 {
		t.Errorf("去重后长度应为3，得到%d", len(unique))
	}

	// 验证原切片未被修改
	if len(ids) != 5 {
		t.Error("原切片不应被修改")
	}
}

// TestIDSliceFilter 测试过滤
func TestIDSliceFilter(t *testing.T) {
	ids := IDSlice{NewID(1), NewID(2), NewID(3), NewID(4), NewID(5)}

	// 过滤出大于2的ID
	filtered := ids.Filter(func(id ID) bool {
		return id > NewID(2)
	})

	expected := 3
	if len(filtered) != expected {
		t.Errorf("过滤后长度应为%d，得到%d", expected, len(filtered))
	}

	// 验证原切片未被修改
	if len(ids) != 5 {
		t.Error("原切片不应被修改")
	}
}

// TestNewIDSet 测试创建ID集合
func TestNewIDSet(t *testing.T) {
	set := NewIDSet(NewID(1), NewID(2), NewID(3))

	if set.Size() != 3 {
		t.Errorf("集合大小应为3，得到%d", set.Size())
	}
}

// TestIDSetAdd 测试添加ID
func TestIDSetAdd(t *testing.T) {
	set := NewIDSet()
	set.Add(NewID(1))
	set.Add(NewID(2))

	if set.Size() != 2 {
		t.Errorf("集合大小应为2，得到%d", set.Size())
	}

	// 添加重复ID
	set.Add(NewID(1))
	if set.Size() != 2 {
		t.Errorf("添加重复ID后大小应保持2，得到%d", set.Size())
	}
}

// TestIDSetRemove 测试移除ID
func TestIDSetRemove(t *testing.T) {
	set := NewIDSet(NewID(1), NewID(2), NewID(3))
	set.Remove(NewID(2))

	if set.Size() != 2 {
		t.Errorf("移除后大小应为2，得到%d", set.Size())
	}

	if set.Contains(NewID(2)) {
		t.Error("不应该包含已移除的ID")
	}
}

// TestIDSetContains 测试检查包含
func TestIDSetContains(t *testing.T) {
	set := NewIDSet(NewID(1), NewID(2), NewID(3))

	if !set.Contains(NewID(2)) {
		t.Error("应该包含ID 2")
	}

	if set.Contains(NewID(4)) {
		t.Error("不应该包含ID 4")
	}
}

// TestIDSetToSlice 测试转换为切片
func TestIDSetToSlice(t *testing.T) {
	set := NewIDSet(NewID(1), NewID(2), NewID(3))
	slice := set.ToSlice()

	if len(slice) != 3 {
		t.Errorf("切片长度应为3，得到%d", len(slice))
	}
}

// TestIDSetUnion 测试并集
func TestIDSetUnion(t *testing.T) {
	set1 := NewIDSet(NewID(1), NewID(2))
	set2 := NewIDSet(NewID(2), NewID(3))

	union := set1.Union(set2)

	if union.Size() != 3 {
		t.Errorf("并集大小应为3，得到%d", union.Size())
	}

	// 验证原集合未被修改
	if set1.Size() != 2 || set2.Size() != 2 {
		t.Error("原集合不应被修改")
	}
}

// TestIDSetIntersect 测试交集
func TestIDSetIntersect(t *testing.T) {
	set1 := NewIDSet(NewID(1), NewID(2), NewID(3))
	set2 := NewIDSet(NewID(2), NewID(3), NewID(4))

	intersect := set1.Intersect(set2)

	if intersect.Size() != 2 {
		t.Errorf("交集大小应为2，得到%d", intersect.Size())
	}

	if !intersect.Contains(NewID(2)) || !intersect.Contains(NewID(3)) {
		t.Error("交集应包含2和3")
	}
}

// TestIDSetDifference 测试差集
func TestIDSetDifference(t *testing.T) {
	set1 := NewIDSet(NewID(1), NewID(2), NewID(3))
	set2 := NewIDSet(NewID(2), NewID(3), NewID(4))

	diff := set1.Difference(set2)

	if diff.Size() != 1 {
		t.Errorf("差集大小应为1，得到%d", diff.Size())
	}

	if !diff.Contains(NewID(1)) {
		t.Error("差集应包含1")
	}
}

// TestBatchIDGenerator 测试批量生成器
func TestBatchIDGenerator(t *testing.T) {
	sf, _ := NewSnowflake(1, 1)
	batch := NewBatchIDGenerator(sf)

	t.Run("生成指定数量的ID", func(t *testing.T) {
		count := 100
		ids, err := batch.Generate(count)
		if err != nil {
			t.Fatalf("批量生成失败: %v", err)
		}

		if len(ids) != count {
			t.Errorf("生成数量不匹配，期望%d，得到%d", count, len(ids))
		}

		// 检查唯一性
		idMap := make(map[int64]bool)
		for _, id := range ids {
			if idMap[id] {
				t.Errorf("发现重复ID: %d", id)
			}
			idMap[id] = true
		}
	})

	t.Run("无效数量", func(t *testing.T) {
		_, err := batch.Generate(0)
		if err == nil {
			t.Error("应该返回错误")
		}

		_, err = batch.Generate(-1)
		if err == nil {
			t.Error("应该返回错误")
		}
	})
}

// TestGenerateIDs 测试全局批量生成函数
func TestGenerateIDs(t *testing.T) {
	count := 50
	ids, err := GenerateIDs(count)
	if err != nil {
		t.Fatalf("批量生成失败: %v", err)
	}

	if len(ids) != count {
		t.Errorf("生成数量不匹配，期望%d，得到%d", count, len(ids))
	}

	// 检查唯一性
	idMap := make(map[int64]bool)
	for _, id := range ids {
		if idMap[id] {
			t.Errorf("发现重复ID: %d", id)
		}
		idMap[id] = true
	}
}

// BenchmarkIDString 基准测试：ID转字符串
func BenchmarkIDString(b *testing.B) {
	id := NewID(9223372036854775807)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = id.String()
	}
}

// BenchmarkIDMarshalJSON 基准测试：JSON序列化
func BenchmarkIDMarshalJSON(b *testing.B) {
	id := NewID(9223372036854775807)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(id)
	}
}

// BenchmarkIDUnmarshalJSON 基准测试：JSON反序列化
func BenchmarkIDUnmarshalJSON(b *testing.B) {
	data := []byte(`"9223372036854775807"`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var id ID
		_ = json.Unmarshal(data, &id)
	}
}

// BenchmarkParseID 基准测试：解析ID字符串
func BenchmarkParseID(b *testing.B) {
	str := "9223372036854775807"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseID(str)
	}
}

// BenchmarkBatchGenerate 基准测试：批量生成
func BenchmarkBatchGenerate(b *testing.B) {
	sf, _ := NewSnowflake(1, 1)
	batch := NewBatchIDGenerator(sf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = batch.Generate(100)
	}
}
